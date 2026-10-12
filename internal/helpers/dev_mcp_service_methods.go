// Copyright 2026 Alibaba Group
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package helpers

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/DingTalk-Real-AI/dingtalk-workspace-cli/internal/corecmd"
	"github.com/DingTalk-Real-AI/dingtalk-workspace-cli/internal/corecmd/contract"
	apperrors "github.com/DingTalk-Real-AI/dingtalk-workspace-cli/internal/errors"
	"github.com/DingTalk-Real-AI/dingtalk-workspace-cli/internal/executor"
	"github.com/DingTalk-Real-AI/dingtalk-workspace-cli/internal/output"
)

// The five dev mcp service operations below are the first ServiceBuilder
// provider (RFC §5.12.4). Their declaration facts preserve the hand-written
// leaves; execution bodies reuse runDevMCPTool.

// devMCPServiceListMethod declares `dev mcp service list`.
func devMCPServiceListMethod() corecmd.MethodDefinition {
	return corecmd.MethodDefinition{
		Identity: corecmd.MethodIdentity{
			CLIPath:     "dev mcp service list",
			ProductID:   "dev",
			Name:        devMCPServiceListTool,
			Short:       "查询有开发权限的 MCP 服务列表（含 serverName）",
			Description: "查询当前用户有开发权限的 MCP 服务",
			Example:     "  dws dev mcp service list --keyword 客户 --page-size 20 --format json",
		},
		PostMount: devMCPServicePostMount(devMCPServiceListTool),
		Input: corecmd.InputSpec{
			Parameters: []corecmd.ParameterSpec{
				{Name: "cursor", Type: corecmd.KindInt, Usage: "分页游标，从 1 开始", Bind: "cursor"},
				{Name: "page-size", Type: corecmd.KindInt, Usage: "每页条数，最大 100", Bind: "pageSize"},
				{Name: "keyword", Type: corecmd.KindString, Trim: true, Usage: "按服务名关键词过滤", Bind: "keyword", OmitEmpty: true},
				{Name: "creator-user-id", Type: corecmd.KindString, Trim: true, Usage: "按创建人 staffId 过滤", Bind: "creatorUserId", OmitEmpty: true},
			},
		},
		Contract:      devMCPServiceContractSpec(devMCPServiceListTool, "查询当前用户有开发权限的 MCP 服务", false, "dws dev mcp service list"),
		Safety:        devMCPReadSafety(),
		OutputRollout: output.RolloutUnifiedActive,
	}
}

// devMCPServiceCreateMethod declares `dev mcp service create`.
func devMCPServiceCreateMethod() corecmd.MethodDefinition {
	return corecmd.MethodDefinition{
		Identity: corecmd.MethodIdentity{
			CLIPath:     "dev mcp service create",
			ProductID:   "dev",
			Name:        devMCPServiceCreateTool,
			Short:       "新建 MCP 服务",
			Description: "创建 MCP 服务开发配置",
			Example:     "  dws dev mcp service create --name 客户信息查询 --server-name customer-info --description \"查询客户基础资料\" --dry-run --format json",
		},
		PostMount: devMCPServicePostMount(devMCPServiceCreateTool),
		Validate:  validateDevMCPServiceCreate,
		Input: corecmd.InputSpec{
			Parameters: []corecmd.ParameterSpec{
				{Name: "name", Type: corecmd.KindString, Trim: true, Usage: "服务名称，组织内唯一", Bind: "name", Required: true, RequiredHint: "--name 为必填"},
				{Name: "description", Type: corecmd.KindString, Trim: true, Usage: "服务用途描述", Bind: "description", Required: true, RequiredHint: "--description 为必填"},
				{Name: "icon-url", Type: corecmd.KindString, Trim: true, Usage: "服务图标 URL", Bind: "icon_url", OmitEmpty: true},
				{Name: "introduction", Type: corecmd.KindString, Trim: true, Usage: "服务详情介绍，支持 markdown", Bind: "introduction", OmitEmpty: true},
				{Name: "server-name", Type: corecmd.KindString, Trim: true, Transform: devMCPServiceServerNameTransform, Usage: "服务英文标识，kebab-case，用于稳定识别已发布 MCP 服务", Bind: "serverName", OmitEmpty: true},
			},
		},
		Contract:      devMCPServiceContractSpec(devMCPServiceCreateTool, "创建 MCP 服务开发配置", true, "dws dev mcp service create"),
		Safety:        devMCPWriteSafety(),
		OutputRollout: output.RolloutUnifiedActive,
	}
}

// devMCPServiceUpdateMethod declares `dev mcp service update`.
func devMCPServiceUpdateMethod() corecmd.MethodDefinition {
	return corecmd.MethodDefinition{
		Identity: corecmd.MethodIdentity{
			CLIPath:     "dev mcp service update",
			ProductID:   "dev",
			Name:        devMCPServiceUpdateTool,
			Short:       "修改 MCP 服务信息",
			Description: "更新 MCP 服务开发配置",
			Example:     "  dws dev mcp service update --mcp-id 10487 --description \"新描述\" --dry-run --format json",
		},
		PostMount: devMCPServicePostMount(devMCPServiceUpdateTool),
		Validate:  validateDevMCPServiceUpdate,
		Input: corecmd.InputSpec{
			Parameters: []corecmd.ParameterSpec{
				{Name: "mcp-id", Type: corecmd.KindInt, Usage: "MCP 服务 ID", Bind: "mcpId", Required: true, RequiredHint: "--mcp-id 为必填"},
				{Name: "name", Type: corecmd.KindString, Trim: true, Usage: "新服务名称", Bind: "name", OmitEmpty: true},
				{Name: "description", Type: corecmd.KindString, Trim: true, Usage: "新服务描述", Bind: "description", OmitEmpty: true},
				{Name: "icon-url", Type: corecmd.KindString, Trim: true, Usage: "新图标 URL", Bind: "icon_url", OmitEmpty: true},
				{Name: "introduction", Type: corecmd.KindString, Trim: true, Usage: "新详情介绍", Bind: "introduction", OmitEmpty: true},
				{Name: "server-name", Type: corecmd.KindString, Trim: true, Transform: devMCPServiceServerNameTransform, Usage: "新服务英文标识，kebab-case", Bind: "serverName", OmitEmpty: true},
			},
		},
		Contract:      devMCPServiceContractSpec(devMCPServiceUpdateTool, "更新 MCP 服务开发配置", true, "dws dev mcp service update"),
		Safety:        devMCPWriteSafety(),
		OutputRollout: output.RolloutUnifiedActive,
	}
}

// devMCPServiceDeleteMethod declares `dev mcp service delete`.
func devMCPServiceDeleteMethod() corecmd.MethodDefinition {
	return corecmd.MethodDefinition{
		Identity: corecmd.MethodIdentity{
			CLIPath:     "dev mcp service delete",
			ProductID:   "dev",
			Name:        devMCPServiceDeleteTool,
			Short:       "删除 MCP 服务（不可恢复）",
			Description: "永久删除 MCP 服务开发配置",
			Example:     "  dws dev mcp service delete --mcp-id 10487 --dry-run --format json",
		},
		PostMount: devMCPServicePostMount(devMCPServiceDeleteTool),
		Input: corecmd.InputSpec{
			Parameters: []corecmd.ParameterSpec{
				{Name: "mcp-id", Type: corecmd.KindInt, Usage: "MCP 服务 ID", Bind: "mcpId", Required: true, RequiredHint: "--mcp-id 为必填"},
			},
		},
		Contract:      devMCPServiceContractSpec(devMCPServiceDeleteTool, "永久删除 MCP 服务开发配置", true, "dws dev mcp service delete"),
		Safety:        devMCPDestructiveSafety(),
		OutputRollout: output.RolloutUnifiedActive,
	}
}

// devMCPServiceContractSpec reproduces the hand-written devMCPContract facts
// for the five service operations, expressed as a MethodDefinition ContractSpec.
func devMCPServiceServerNameTransform(raw string) (any, error) {
	serverName := strings.ToLower(strings.TrimSpace(raw))
	if serverName == "" {
		return nil, nil
	}
	if len(serverName) > 255 {
		return nil, apperrors.NewValidation("--server-name 必须不超过 255 个字符")
	}
	for i, r := range serverName {
		isAlphaNum := r >= 'a' && r <= 'z' || r >= '0' && r <= '9'
		validDash := r == '-' && i > 0 && i < len(serverName)-1 && serverName[i-1] != '-'
		if !isAlphaNum && !validDash {
			return nil, apperrors.NewValidation("--server-name 必须是 kebab-case，仅含字母、数字和单个中划线")
		}
	}
	return serverName, nil
}

func devMCPServiceContractSpec(tool, description string, dryRun bool, example string) corecmd.ContractSpec {
	spec := corecmd.ContractSpec{
		Selection: contract.SelectionSpec{
			AgentSummary: description,
			UseWhen:      []string{description},
			AvoidWhen:    []string{"只需调用已发布 MCP 工具时使用 dws mcp published，而不是开发配置命令"},
			Examples:     []string{example},
		},
		Interface: &contract.InterfaceSpec{
			Mode:         contract.InterfaceModeComposite,
			Availability: contract.InterfaceAvailable,
			Reason:       "Reviewed unpinned adapter: this static developer command calls the helper-only mcpdev/" + tool + " management endpoint, which is intentionally absent from pinned public MCP metadata.",
		},
	}
	if dryRun {
		spec.DryRun = &contract.DryRunSpec{
			PreviewKind: contract.DryRunPreviewInvocation,
			RemoteReads: false,
		}
	}
	return spec
}

type devMCPServiceMethod struct {
	def     corecmd.MethodDefinition
	binding corecmd.MethodBinding
}

// devMCPServiceMethods is the provider catalog for the five pilot operations.
// Each entry keeps the declaration and its constrained execution binding
// together as one authoring unit before ServiceBuilder compiles it.
func devMCPServiceMethods(runner executor.Runner) []devMCPServiceMethod {
	return []devMCPServiceMethod{
		{devMCPServiceListMethod(), corecmd.ToolBinding(devMCPServiceListTool, devMCPServiceDispatch(runner, devMCPServiceListTool))},
		{devMCPServiceCreateMethod(), devMCPServiceCreateBinding(runner)},
		{devMCPServiceUpdateMethod(), corecmd.ToolBinding(devMCPServiceUpdateTool, devMCPServiceDispatch(runner, devMCPServiceUpdateTool))},
		{devMCPServiceDeleteMethod(), corecmd.ToolBinding(devMCPServiceDeleteTool, devMCPServiceDispatch(runner, devMCPServiceDeleteTool))},
	}
}

// addDevMCPServiceMethods mounts the five service methods under the dev mcp
// tree through the ServiceBuilder, replacing the hand-written leaf factories.
func addDevMCPServiceMethods(runner executor.Runner, parent *cobra.Command) error {
	b, err := corecmd.NewServiceBuilder(parent)
	if err != nil {
		return err
	}
	for i, m := range devMCPServiceMethods(runner) {
		if i == 1 {
			if err := devMCPServiceGetOperation(runner).Mount(b); err != nil {
				return err
			}
		}
		if err := b.Add(m.def, m.binding); err != nil {
			return err
		}
	}
	return nil
}

func devMCPServicePostMount(tool string) func(*cobra.Command) {
	return func(cmd *cobra.Command) {
		preferLegacyLeaf(cmd)
		annotateDevMCPTool(cmd, tool)
	}
}

// devMCPServiceDispatch returns the standard single-call executor: assembled
// toolArgs routed to runDevMCPTool unchanged.
func devMCPServiceDispatch(runner executor.Runner, tool string) func(c *corecmd.Ctx, toolArgs map[string]any) error {
	return func(c *corecmd.Ctx, toolArgs map[string]any) error {
		return runDevMCPTool(runner, c.Command(), tool, toolArgs)
	}
}

// devMCPServiceCreateBinding keeps create's special post-success warning while
// retaining the standard Tool identity; the provider executor is still subject
// to the framework's assembled-args, safety, and output pipeline.
func devMCPServiceCreateBinding(runner executor.Runner) corecmd.MethodBinding {
	return corecmd.MethodBinding{
		Tool: devMCPServiceCreateTool,
		ToolExecutor: func(c *corecmd.Ctx, toolArgs map[string]any) error {
			cmd := c.Command()
			serverName, err := devMCPServerNameFlag(cmd)
			if err != nil {
				return err
			}
			if err := runDevMCPTool(runner, cmd, devMCPServiceCreateTool, toolArgs); err != nil {
				return err
			}
			if serverName == "" {
				fmt.Fprintln(cmd.ErrOrStderr(), "警告：未设置 --server-name，服务将缺少稳定的语义化标识。建议创建时传 --server-name <kebab-case>；或事后执行 dws dev mcp service update --mcp-id <mcpId> --server-name <kebab-case>。")
			}
			return nil
		},
	}
}
