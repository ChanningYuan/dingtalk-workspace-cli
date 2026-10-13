// Copyright 2026 Alibaba Group
// Licensed under the Apache License, Version 2.0 (the "License");

package helpers

import (
	"context"
	"errors"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/DingTalk-Real-AI/dingtalk-workspace-cli/internal/auth"
	apperrors "github.com/DingTalk-Real-AI/dingtalk-workspace-cli/internal/errors"
)

const employeeVisibilityStartupTimeout = 10 * time.Second

// 启动时迁移旧受管绑定；调用方已经持有 worker 锁，成功前不得宣告租约或业务就绪。
// 不能借用当前 Profile：宿主的 --profile 是员工身份，用户也可能已切换主管账号。
func prepareEmployeeVisibility(parent context.Context, b digitalEmployeeBinding) (digitalEmployeeBinding, error) {
	if strings.TrimSpace(b.SupervisorProfile) != "" || b.RuntimeBindingID == "" {
		return b, nil
	}
	ctx, cancel := context.WithTimeout(parent, employeeVisibilityStartupTimeout)
	defer cancel()
	fail := func(code string) (digitalEmployeeBinding, error) {
		if ctx.Err() != nil {
			code = "visibility_migration_timeout"
		}
		return b, employeeTerminal(code)
	}
	if ctx.Err() != nil {
		return fail("visibility_migration_timeout")
	}
	employee, err := employeeVisibilityToken(ctx, b.DWSProfile)
	if err != nil || employee == nil || strings.TrimSpace(employee.AccessToken) == "" {
		return fail("visibility_supervisor_unavailable")
	}
	if auth.ProfileSelector(auth.Profile{CorpID: employee.CorpID, UserID: employee.UserID}) != b.DWSProfile {
		return fail("visibility_identity_mismatch")
	}
	profiles, err := deapConnectLoadProfiles(deapConnectConfigDir())
	if err != nil || profiles == nil {
		return fail("visibility_supervisor_unavailable")
	}
	// 候选仅来自本机精确身份元数据，名称、current、primary 和登录时间均不是权威。
	// 旧绑定没有保存主管企业，不能擅自假设主管 Profile 与员工 Profile 属于同一企业。
	candidates := make(map[string]string)
	for _, profile := range profiles.Profiles {
		if strings.TrimSpace(profile.CorpID) == "" || strings.TrimSpace(profile.UserID) == "" {
			continue
		}
		selector := auth.ProfileSelector(profile)
		if selector != b.DWSProfile {
			candidates[selector] = strings.TrimSpace(profile.UserID)
		}
	}
	if len(candidates) > 32 {
		return fail("visibility_supervisor_unavailable")
	}
	matched := ""
	selectors := make([]string, 0, len(candidates))
	for selector := range candidates {
		selectors = append(selectors, selector)
	}
	sort.Strings(selectors)
	for _, selector := range selectors {
		if ctx.Err() != nil {
			return fail("visibility_migration_timeout")
		}
		operator, err := resolveExactOperatorOpenDingTalkID(ctx, employee.AccessToken, candidates[selector])
		if err != nil {
			// 候选查询不完整时不能断言唯一；留给原主管显式 restart 恢复。
			return fail("visibility_supervisor_unavailable")
		}
		if operator != b.OperatorOpenDingTalkID {
			continue
		}
		if matched != "" {
			return fail("visibility_supervisor_ambiguous")
		}
		matched = selector
	}
	if matched == "" {
		return fail("visibility_supervisor_missing")
	}
	// employeeVisibilityCall 会刷新并核对该精确主管 Profile 的 token 身份。
	published, err := employeeVisibilityCall(ctx, matched, deapAgentServerID, deapAgentDetailTool, map[string]any{"agentUuid": b.AgentUUID, "snapshot": "published"})
	if err != nil {
		return fail("visibility_supervisor_unavailable")
	}
	data := businessDataMap(published)
	identity, ok := publishedDigitalEmployeeIdentity(published)
	if !ok || jsonScalar(data["snapshot"]) != "published" || jsonScalar(data["agentUuid"]) != b.AgentUUID || auth.ProfileSelector(auth.Profile{CorpID: identity.CorpID, UserID: identity.UserID}) != b.DWSProfile {
		return fail("visibility_identity_mismatch")
	}
	if ctx.Err() != nil {
		return fail("visibility_migration_timeout")
	}
	// 不取 operation 锁：connect restart 持有该锁并等待宿主启动租约。
	// 与所有绑定写入共享短锁，完整重读比较后只补主管字段，不能复活已停止/解绑实例。
	lock, err := auth.AcquireDualLock(ctx, employeeBindingLockPath(b.DWSProfile))
	if err != nil {
		return fail("visibility_migration_unavailable")
	}
	defer lock.Release()
	current, err := loadDigitalEmployeeBinding(deapConnectConfigDir(), b.DWSProfile)
	if err != nil || current != b || employeeBindingState(current) != "bound" || employeeDesiredState(current) != "running" {
		return fail("visibility_binding_changed")
	}
	if err := checkEmployeeServerOperation(current); err != nil {
		return fail("visibility_binding_changed")
	}
	if ctx.Err() != nil {
		return fail("visibility_migration_timeout")
	}
	current.SupervisorProfile = matched
	if err := writeEmployeeJSON(digitalEmployeeBindingPath(deapConnectConfigDir(), b.DWSProfile), current); err != nil {
		return fail("visibility_migration_persist_failed")
	}
	return current, nil
}

func employeeBindingLockPath(profile string) string {
	return filepath.Join(deapConnectConfigDir(), "digital-employees", digitalEmployeeScope(profile), "binding-lock")
}

// 私有租约协议只输出稳定分类和固定恢复提示，不回传身份查询的原始响应。
func employeeVisibilityStartupError(err error) error {
	code := "visibility_migration_unavailable"
	var failure *employeeRunError
	if errors.As(err, &failure) {
		code = failure.Code
	}
	return apperrors.NewValidation("旧数字员工绑定的可见范围身份迁移未完成，已停止启动",
		apperrors.WithReason(code), apperrors.WithRetryable(false),
		apperrors.WithHint("请确认原主管账号已登录，并以该精确 Profile 执行 dingtalk-tag connect restart；不要手工覆盖主管身份或重新发送历史消息。"))
}
