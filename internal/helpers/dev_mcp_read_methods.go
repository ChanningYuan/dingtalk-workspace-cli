package helpers

import (
	"strings"

	"github.com/spf13/cobra"

	"github.com/DingTalk-Real-AI/dingtalk-workspace-cli/internal/corecmd"
	"github.com/DingTalk-Real-AI/dingtalk-workspace-cli/internal/errors"
	"github.com/DingTalk-Real-AI/dingtalk-workspace-cli/internal/executor"
	"github.com/DingTalk-Real-AI/dingtalk-workspace-cli/internal/output"
)

func devMCPURLGetMethod() corecmd.MethodDefinition {
	return corecmd.MethodDefinition{
		Identity: corecmd.MethodIdentity{
			CLIPath:     "dev mcp url get",
			ProductID:   "dev",
			Name:        devMCPServerURLGetTool,
			Short:       "获取 MCP 实例接入地址（按调用者个人身份生成，含个人 key 勿外发）",
			Description: "获取当前身份可用的 MCP 实例接入地址",
			Example:     "  dws dev mcp url get --mcp-id 10487 --source MARKET --format json",
		},
		PostMount: devMCPToolReadPostMount(devMCPServerURLGetTool),
		Validate:  validateDevMCPURLGet,
		Input: corecmd.InputSpec{Parameters: []corecmd.ParameterSpec{
			{Name: "mcp-id", Type: corecmd.KindInt, Usage: "MCP 服务 ID", Bind: "mcpId"},
			{Name: "source", Type: corecmd.KindString, Default: "MARKET", Trim: true, Usage: "服务来源：MARKET 或 PUBLISHED", Bind: "source", Transform: devMCPURLSourceTransform},
		}},
		Contract:      devMCPToolContractSpec(devMCPServerURLGetTool, "获取当前身份可用的 MCP 实例接入地址", false, "dws dev mcp url get"),
		Safety:        devMCPReadSafety(),
		OutputRollout: output.RolloutUnifiedActive,
	}
}

func devMCPAuthGetMethod() corecmd.MethodDefinition {
	return corecmd.MethodDefinition{
		Identity: corecmd.MethodIdentity{
			CLIPath:     "dev mcp auth get",
			ProductID:   "dev",
			Name:        devMCPAuthConfigGetTool,
			Short:       "查询 MCP 下游鉴权配置",
			Description: "查询 MCP 服务的下游鉴权配置",
			Example:     "  dws dev mcp auth get --mcp-id 10520 --format json",
		},
		PostMount: devMCPToolReadPostMount(devMCPAuthConfigGetTool),
		Validate:  validateDevMCPRequiredIntFlag("mcp-id"),
		Input: corecmd.InputSpec{Parameters: []corecmd.ParameterSpec{
			{Name: "mcp-id", Type: corecmd.KindInt, Usage: "MCP 服务 ID", Bind: "mcpId"},
		}},
		Contract:      devMCPToolContractSpec(devMCPAuthConfigGetTool, "查询 MCP 服务的下游鉴权配置", false, "dws dev mcp auth get"),
		Safety:        devMCPReadSafety(),
		OutputRollout: output.RolloutUnifiedActive,
	}
}

func addDevMCPURLGetMethod(runner executor.Runner, parent *cobra.Command) error {
	builder, err := corecmd.NewServiceBuilder(parent)
	if err != nil {
		return err
	}
	return builder.Add(devMCPURLGetMethod(), corecmd.MethodBinding{
		Invoke: func(ctx *corecmd.Ctx, toolArgs map[string]any) error {
			return runDevMCPURLGet(runner, ctx.Command(), toolArgs)
		},
	})
}

func addDevMCPAuthGetMethod(runner executor.Runner, parent *cobra.Command) error {
	builder, err := corecmd.NewServiceBuilder(parent)
	if err != nil {
		return err
	}
	return builder.Add(
		devMCPAuthGetMethod(),
		corecmd.ToolBinding(devMCPAuthConfigGetTool, devMCPToolDispatch(runner, devMCPAuthConfigGetTool)),
	)
}

func validateDevMCPURLGet(cmd *cobra.Command, _ []string) error {
	if _, err := devMCPRequiredInt(cmd, "mcp-id"); err != nil {
		return err
	}
	source, err := devMCPRequiredString(cmd, "source")
	if err != nil {
		return err
	}
	_, err = devMCPURLSourceTransform(source)
	return err
}

func devMCPURLSourceTransform(raw string) (any, error) {
	source := strings.ToUpper(strings.TrimSpace(raw))
	if source == "" {
		return nil, errors.NewValidation("--source 为必填")
	}
	if source != "MARKET" && source != "PUBLISHED" {
		return nil, errors.NewValidation("--source 只支持 MARKET 或 PUBLISHED")
	}
	return source, nil
}
