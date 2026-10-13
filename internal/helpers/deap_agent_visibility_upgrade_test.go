// Copyright 2026 Alibaba Group
// Licensed under the Apache License, Version 2.0 (the "License");

package helpers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/DingTalk-Real-AI/dingtalk-workspace-cli/internal/auth"
	"github.com/DingTalk-Real-AI/dingtalk-workspace-cli/internal/testseam"
	"github.com/DingTalk-Real-AI/dingtalk-workspace-cli/pkg/edition"
)

type visibilityUpgradeCaller struct {
	digitalEmployeeProtocolCaller
	call func(context.Context, string, string, string, map[string]any) (*edition.ToolResult, error)
}

func (c *visibilityUpgradeCaller) CallToolWithToken(ctx context.Context, token, server, tool string, args map[string]any) (*edition.ToolResult, error) {
	return c.call(ctx, token, server, tool, args)
}

type visibilityUpgradeFixture struct {
	binding                               digitalEmployeeBinding
	file                                  string
	profiles                              []auth.Profile
	operators                             map[string]string
	lookupErr                             error
	profilesErr                           error
	tokenErr                              error
	token                                 *auth.TokenData
	detail                                map[string]any
	detailErr                             error
	beforeDetail                          func(context.Context)
	lookups                               []string
	tokenCalls, profileCalls, detailCalls int
}

func newVisibilityUpgradeFixture(t *testing.T) *visibilityUpgradeFixture {
	t.Helper()
	dir := t.TempDir()
	testseam.Swap(t, &deapConnectConfigDir, func() string { return dir })
	b := digitalEmployeeBinding{SchemaVersion: 1, AgentUUID: "upgrade-employee", DWSProfile: "employee-corp:employee-user", OperatorOpenDingTalkID: "original-operator", Channel: "dsh", BindingRevision: 9, BindingState: "bound", DesiredState: "running", RuntimeBindingID: "legacy-binding", DeviceID: "original-device"}
	if err := saveDigitalEmployeeBinding(dir, b); err != nil {
		t.Fatal(err)
	}
	f := &visibilityUpgradeFixture{
		binding: b, file: digitalEmployeeBindingPath(dir, b.DWSProfile),
		profiles:  []auth.Profile{{CorpID: "employee-corp", UserID: "employee-user"}, {CorpID: "manager-corp", UserID: "original-manager"}, {CorpID: "manager-corp", UserID: "original-manager"}, {CorpID: "different-corp", UserID: "other-user"}},
		operators: map[string]string{"original-manager": "original-operator", "other-user": "other-operator"},
		token:     &auth.TokenData{CorpID: "employee-corp", UserID: "employee-user", AccessToken: "employee-fixture-only"},
		detail:    map[string]any{"agentUuid": b.AgentUUID, "snapshot": "published", "type": "local_agent", "status": "online", "profile": map[string]any{"corpId": "employee-corp", "userId": "employee-user"}},
	}
	testseam.Swap(t, &deapConnectLoadProfiles, func(string) (*auth.ProfilesConfig, error) {
		f.profileCalls++
		return &auth.ProfilesConfig{Version: 2, CurrentProfile: "different-corp:other-user", PrimaryProfile: b.DWSProfile, Profiles: f.profiles}, f.profilesErr
	})
	testseam.Swap(t, &employeeVisibilityToken, func(_ context.Context, profile string) (*auth.TokenData, error) {
		f.tokenCalls++
		if profile != b.DWSProfile {
			t.Fatalf("employee identity must be exact, got %s", profile)
		}
		return f.token, f.tokenErr
	})
	testseam.Swap(t, &deapConnectLoadSupervisorToken, func(context.Context, string) (*auth.TokenData, error) {
		t.Fatal("must not choose current or primary supervisor")
		return nil, nil
	})
	testseam.Swap(t, &employeeVisibilityCall, func(ctx context.Context, profile, server, tool string, args map[string]any) (map[string]any, error) {
		f.detailCalls++
		if profile != "manager-corp:original-manager" || tool != deapAgentDetailTool || args["snapshot"] != "published" || args["agentUuid"] != b.AgentUUID {
			t.Fatalf("incorrect detail authority: %s %s %#v", profile, tool, args)
		}
		if f.beforeDetail != nil {
			f.beforeDetail(ctx)
		}
		return map[string]any{"data": f.detail}, f.detailErr
	})
	InitDepsForTest(t, &visibilityUpgradeCaller{call: func(ctx context.Context, token, server, tool string, args map[string]any) (*edition.ToolResult, error) {
		if token != "employee-fixture-only" || server != "contact" || tool != "search_contact_by_key_word" {
			t.Fatalf("wrong operator lookup identity: %s/%s", server, tool)
		}
		user, _ := args["keyword"].(string)
		f.lookups = append(f.lookups, user)
		if f.lookupErr != nil {
			return nil, f.lookupErr
		}
		openID := f.operators[user]
		if openID == "" {
			openID = "unrelated-operator"
		}
		raw, _ := json.Marshal(map[string]any{"result": []any{map[string]any{"userId": user, "openDingTalkId": openID}}})
		return &edition.ToolResult{Content: []edition.ContentBlock{{Type: "text", Text: string(raw)}}}, nil
	}})
	return f
}

func assertVisibilityUpgradeFailure(t *testing.T, err error, want string) {
	t.Helper()
	var classified *employeeRunError
	if !errors.As(err, &classified) || classified.Code != want || classified.Retryable == nil || *classified.Retryable {
		t.Fatalf("want terminal %s, got %#v", want, err)
	}
	if strings.Contains(err.Error(), "private-service-detail") {
		t.Fatal("unsafe service error leaked")
	}
}

func TestCrossPlatformCoverageVisibilityUpgradeNonCurrentSupervisor(t *testing.T) {
	f := newVisibilityUpgradeFixture(t)
	got, err := prepareEmployeeVisibility(context.Background(), f.binding)
	if err != nil {
		t.Fatal(err)
	}
	want := f.binding
	want.SupervisorProfile = "manager-corp:original-manager"
	if got != want {
		t.Fatalf("migration changed unrelated fields: got %#v want %#v", got, want)
	}
	persisted, err := loadDigitalEmployeeBinding(deapConnectConfigDir(), f.binding.DWSProfile)
	if err != nil || persisted != want {
		t.Fatalf("persisted %#v err %v", persisted, err)
	}
	if len(f.lookups) != 2 || f.tokenCalls != 1 || f.detailCalls != 1 {
		t.Fatalf("lookups=%v token=%d detail=%d", f.lookups, f.tokenCalls, f.detailCalls)
	}
	for _, user := range f.lookups {
		if user == "employee-user" {
			t.Fatal("employee was considered its own supervisor")
		}
	}
}

func TestCrossPlatformCoverageVisibilityUpgradeNoop(t *testing.T) {
	for _, mode := range []string{"local_allowlist", "already_set"} {
		t.Run(mode, func(t *testing.T) {
			f := newVisibilityUpgradeFixture(t)
			b := f.binding
			if mode == "local_allowlist" {
				b.RuntimeBindingID = ""
			} else {
				b.SupervisorProfile = "recorded-corp:recorded-manager"
			}
			before, err := os.ReadFile(f.file)
			if err != nil {
				t.Fatal(err)
			}
			got, err := prepareEmployeeVisibility(context.Background(), b)
			if err != nil || got != b {
				t.Fatalf("noop=%#v err=%v", got, err)
			}
			after, err := os.ReadFile(f.file)
			if err != nil {
				t.Fatal(err)
			}
			if string(before) != string(after) || f.profileCalls != 0 || f.tokenCalls != 0 || len(f.lookups) != 0 || f.detailCalls != 0 {
				t.Fatal("no-migration path queried or rewrote state")
			}
		})
	}
}

func TestCrossPlatformCoverageVisibilityUpgradeFailures(t *testing.T) {
	cases := []struct {
		name, code string
		change     func(*visibilityUpgradeFixture)
	}{
		{"no candidates", "visibility_supervisor_missing", func(f *visibilityUpgradeFixture) {
			f.profiles = []auth.Profile{{CorpID: "employee-corp", UserID: "employee-user"}}
		}},
		{"no matching operator", "visibility_supervisor_missing", func(f *visibilityUpgradeFixture) { f.operators["original-manager"] = "unrelated" }},
		{"multiple matching profiles", "visibility_supervisor_ambiguous", func(f *visibilityUpgradeFixture) { f.operators["other-user"] = "original-operator" }},
		{"candidate query failed", "visibility_supervisor_unavailable", func(f *visibilityUpgradeFixture) { f.lookupErr = errors.New("private-service-detail") }},
		{"profile load failed", "visibility_supervisor_unavailable", func(f *visibilityUpgradeFixture) { f.profilesErr = errors.New("private-service-detail") }},
		{"employee token failed", "visibility_supervisor_unavailable", func(f *visibilityUpgradeFixture) { f.tokenErr = errors.New("private-service-detail") }},
		{"employee token identity mismatch", "visibility_identity_mismatch", func(f *visibilityUpgradeFixture) { f.token.CorpID = "other-corp" }},
		{"published query failed", "visibility_supervisor_unavailable", func(f *visibilityUpgradeFixture) { f.detailErr = errors.New("private-service-detail") }},
		{"wrong snapshot", "visibility_identity_mismatch", func(f *visibilityUpgradeFixture) { f.detail["snapshot"] = "draft" }},
		{"wrong employee UUID", "visibility_identity_mismatch", func(f *visibilityUpgradeFixture) { f.detail["agentUuid"] = "other-employee" }},
		{"wrong published employee profile", "visibility_identity_mismatch", func(f *visibilityUpgradeFixture) {
			f.detail["profile"] = map[string]any{"corpId": "other-corp", "userId": "employee-user"}
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := newVisibilityUpgradeFixture(t)
			tc.change(f)
			before, err := os.ReadFile(f.file)
			if err != nil {
				t.Fatal(err)
			}
			_, err = prepareEmployeeVisibility(context.Background(), f.binding)
			assertVisibilityUpgradeFailure(t, err, tc.code)
			after, readErr := os.ReadFile(f.file)
			if readErr != nil {
				t.Fatal(readErr)
			}
			if string(before) != string(after) {
				t.Fatal("failed migration changed persisted binding")
			}
		})
	}
}

func TestCrossPlatformCoverageVisibilityUpgradeTimeout(t *testing.T) {
	f := newVisibilityUpgradeFixture(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Millisecond)
	defer cancel()
	f.beforeDetail = func(ctx context.Context) { <-ctx.Done(); f.detailErr = ctx.Err() }
	_, err := prepareEmployeeVisibility(ctx, f.binding)
	assertVisibilityUpgradeFailure(t, err, "visibility_migration_timeout")
	persisted, readErr := loadDigitalEmployeeBinding(deapConnectConfigDir(), f.binding.DWSProfile)
	if readErr != nil || persisted != f.binding {
		t.Fatalf("timeout changed binding: %#v %v", persisted, readErr)
	}
}

func TestCrossPlatformCoverageVisibilityUpgradeConcurrentBindingChange(t *testing.T) {
	for _, field := range []string{"stop", "operator", "revision", "channel", "runtime_binding"} {
		t.Run(field, func(t *testing.T) {
			f := newVisibilityUpgradeFixture(t)
			changed := f.binding
			switch field {
			case "stop":
				changed.DesiredState = "stopped"
			case "operator":
				changed.OperatorOpenDingTalkID = "new-operator"
			case "revision":
				changed.BindingRevision++
			case "channel":
				changed.Channel = "codex"
			case "runtime_binding":
				changed.RuntimeBindingID = "new-binding"
			}
			f.beforeDetail = func(context.Context) {
				raw, _ := json.Marshal(changed)
				if err := os.WriteFile(f.file, raw, 0o600); err != nil {
					t.Fatal(err)
				}
			}
			_, err := prepareEmployeeVisibility(context.Background(), f.binding)
			assertVisibilityUpgradeFailure(t, err, "visibility_binding_changed")
			persisted, readErr := loadDigitalEmployeeBinding(deapConnectConfigDir(), f.binding.DWSProfile)
			if readErr != nil || persisted != changed {
				t.Fatalf("concurrent update overwritten: %#v %v", persisted, readErr)
			}
		})
	}
}

func TestCrossPlatformCoverageVisibilityUpgradePersistFailure(t *testing.T) {
	f := newVisibilityUpgradeFixture(t)
	realRename := atomicRename
	testseam.Swap(t, &atomicRename, func(source, destination string) error {
		if destination == f.file {
			return errors.New("private-service-detail")
		}
		return realRename(source, destination)
	})
	_, err := prepareEmployeeVisibility(context.Background(), f.binding)
	assertVisibilityUpgradeFailure(t, err, "visibility_migration_persist_failed")
	persisted, readErr := loadDigitalEmployeeBinding(deapConnectConfigDir(), f.binding.DWSProfile)
	if readErr != nil || persisted != f.binding {
		t.Fatalf("failed persistence changed state: %#v %v", persisted, readErr)
	}
}

func TestCrossPlatformCoverageVisibilityUpgradeCandidateLimit(t *testing.T) {
	f := newVisibilityUpgradeFixture(t)
	for i := 0; i < 33; i++ {
		f.profiles = append(f.profiles, auth.Profile{CorpID: "candidate-corp", UserID: fmt.Sprintf("candidate-%d", i)})
	}
	_, err := prepareEmployeeVisibility(context.Background(), f.binding)
	assertVisibilityUpgradeFailure(t, err, "visibility_supervisor_unavailable")
	if len(f.lookups) != 0 || f.detailCalls != 0 {
		t.Fatal("candidate limit must be applied before RPC enumeration")
	}
}

func TestCrossPlatformCoverageVisibilityUpgradeCanceledBeforeLookup(t *testing.T) {
	f := newVisibilityUpgradeFixture(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := prepareEmployeeVisibility(ctx, f.binding)
	assertVisibilityUpgradeFailure(t, err, "visibility_migration_timeout")
	if f.tokenCalls != 0 || f.profileCalls != 0 || len(f.lookups) != 0 || f.detailCalls != 0 {
		t.Fatal("canceled migration queried identities")
	}
}

func TestCrossPlatformCoverageVisibilityUpgradeLockFailure(t *testing.T) {
	f := newVisibilityUpgradeFixture(t)
	lockPath := employeeBindingLockPath(f.binding.DWSProfile)
	if err := os.RemoveAll(lockPath); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(lockPath, []byte("not-a-directory"), 0600); err != nil {
		t.Fatal(err)
	}
	_, err := prepareEmployeeVisibility(context.Background(), f.binding)
	assertVisibilityUpgradeFailure(t, err, "visibility_migration_unavailable")
	persisted, readErr := loadDigitalEmployeeBinding(deapConnectConfigDir(), f.binding.DWSProfile)
	if readErr != nil || persisted != f.binding {
		t.Fatalf("failed lock changed binding: %#v %v", persisted, readErr)
	}
}

func TestCrossPlatformCoverageVisibilityUpgradeAndStopPreserveAuthority(t *testing.T) {
	for _, order := range []string{"migration_before_stale_stop", "stop_before_migration"} {
		t.Run(order, func(t *testing.T) {
			f := newVisibilityUpgradeFixture(t)
			staleStop := f.binding
			staleStop.DesiredState = "stopped"
			if order == "migration_before_stale_stop" {
				migrated, err := prepareEmployeeVisibility(context.Background(), f.binding)
				if err != nil {
					t.Fatal(err)
				}
				if err = updateEmployeeBinding(staleStop); err != nil {
					t.Fatal(err)
				}
				persisted, err := loadDigitalEmployeeBinding(deapConnectConfigDir(), f.binding.DWSProfile)
				migrated.DesiredState = "stopped"
				if err != nil || persisted != migrated {
					t.Fatalf("stale stop erased migrated authority: %#v %v", persisted, err)
				}
			} else {
				if err := updateEmployeeBinding(staleStop); err != nil {
					t.Fatal(err)
				}
				_, err := prepareEmployeeVisibility(context.Background(), f.binding)
				assertVisibilityUpgradeFailure(t, err, "visibility_binding_changed")
				persisted, err := loadDigitalEmployeeBinding(deapConnectConfigDir(), f.binding.DWSProfile)
				if err != nil || persisted != staleStop {
					t.Fatalf("migration revived stopped binding: %#v %v", persisted, err)
				}
			}
		})
	}
}

func TestCrossPlatformCoverageVisibilityUpgradeValidatesSupervisorTokenIdentity(t *testing.T) {
	for _, mode := range []string{"correct_exact_identity", "wrong_token_identity"} {
		t.Run(mode, func(t *testing.T) {
			f := newVisibilityUpgradeFixture(t)
			// 此处执行真实 Profile 调用边界，只有 TokenProvider 和 MCP 传输是合成数据。
			testseam.Swap(t, &employeeVisibilityCall, employeeVisibilityProfileCall)
			managerTokenCalls, publishedCalls := 0, 0
			testseam.Swap(t, &employeeVisibilityToken, func(_ context.Context, profile string) (*auth.TokenData, error) {
				if profile == f.binding.DWSProfile {
					return f.token, nil
				}
				if profile != "manager-corp:original-manager" {
					t.Fatalf("unexpected supervisor selector %s", profile)
				}
				managerTokenCalls++
				user := "original-manager"
				if mode == "wrong_token_identity" {
					user = "other-user"
				}
				return &auth.TokenData{CorpID: "manager-corp", UserID: user, AccessToken: "manager-fixture-only"}, nil
			})
			InitDepsForTest(t, &visibilityUpgradeCaller{call: func(_ context.Context, token, server, tool string, args map[string]any) (*edition.ToolResult, error) {
				var response map[string]any
				if tool == "search_contact_by_key_word" {
					if token != "employee-fixture-only" || server != "contact" {
						t.Fatal("operator resolution did not use employee token")
					}
					user, _ := args["keyword"].(string)
					response = map[string]any{"result": []any{map[string]any{"userId": user, "openDingTalkId": f.operators[user]}}}
				} else {
					publishedCalls++
					if token != "manager-fixture-only" || server != deapAgentServerID || tool != deapAgentDetailTool || args["snapshot"] != "published" || args["agentUuid"] != f.binding.AgentUUID {
						t.Fatal("published detail did not use verified supervisor authority")
					}
					response = map[string]any{"success": true, "data": f.detail}
				}
				raw, err := json.Marshal(response)
				if err != nil {
					t.Fatal(err)
				}
				return &edition.ToolResult{Content: []edition.ContentBlock{{Type: "text", Text: string(raw)}}}, nil
			}})
			before, err := os.ReadFile(f.file)
			if err != nil {
				t.Fatal(err)
			}
			got, err := prepareEmployeeVisibility(context.Background(), f.binding)
			if managerTokenCalls != 1 {
				t.Fatalf("supervisor exact token lookups=%d", managerTokenCalls)
			}
			if mode == "wrong_token_identity" {
				assertVisibilityUpgradeFailure(t, err, "visibility_supervisor_unavailable")
				if publishedCalls != 0 {
					t.Fatal("wrong supervisor token was used for published detail")
				}
				after, err := os.ReadFile(f.file)
				if err != nil {
					t.Fatal(err)
				}
				if string(before) != string(after) {
					t.Fatal("wrong supervisor token caused binding write")
				}
			} else {
				if err != nil {
					t.Fatal(err)
				}
				if publishedCalls != 1 || got.SupervisorProfile != "manager-corp:original-manager" {
					t.Fatalf("migration=%#v publishedCalls=%d", got, publishedCalls)
				}
				persisted, err := loadDigitalEmployeeBinding(deapConnectConfigDir(), f.binding.DWSProfile)
				if err != nil || persisted != got {
					t.Fatalf("verified identity not persisted: %#v %v", persisted, err)
				}
			}
		})
	}
}
