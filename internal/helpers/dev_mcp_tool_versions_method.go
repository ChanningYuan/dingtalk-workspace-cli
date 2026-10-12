package helpers

import (
	"github.com/spf13/cobra"

	"github.com/DingTalk-Real-AI/dingtalk-workspace-cli/internal/corecmd"
	"github.com/DingTalk-Real-AI/dingtalk-workspace-cli/internal/executor"
	"github.com/DingTalk-Real-AI/dingtalk-workspace-cli/internal/output"
)

func devMCPToolVersionsMethod() corecmd.MethodDefinition {
	return corecmd.MethodDefinition{
		Identity: corecmd.MethodIdentity{
			CLIPath:     "dev mcp tool versions",
			ProductID:   "dev",
			Name:        devMCPToolVersionsTool,
			Short:       "查询 MCP 工具版本历史",
			Description: "查询 MCP 工具版本历史",
			Example:     "  dws dev mcp tool versions --mcp-id 10487 --tool-id G-ACT-xxx --format json",
		},
		PostMount: devMCPToolReadPostMount(devMCPToolVersionsTool),
		Validate:  validateDevMCPToolGet,
		Input: corecmd.InputSpec{
			Parameters: []corecmd.ParameterSpec{
				{Name: "mcp-id", Type: corecmd.KindInt, Usage: "MCP 服务 ID", Bind: "mcpId"},
				{Name: "tool-id", Type: corecmd.KindString, Trim: true, Usage: "MCP 工具 ID，G-ACT- 开头", Bind: "toolId"},
				{Name: "action-id", Type: corecmd.KindString, Usage: "已更名为 --tool-id", Bind: "actionId", Hidden: true, OmitEmpty: true},
				{Name: "cursor", Type: corecmd.KindInt, Usage: "分页游标，从 1 开始", Bind: "cursor", OmitEmpty: true},
				{Name: "page-size", Type: corecmd.KindInt, Usage: "每页条数，最大 100", Bind: "pageSize", OmitEmpty: true},
			},
		},
		Contract:      devMCPToolContractSpec(devMCPToolVersionsTool, "查询 MCP 工具版本历史", false, "dws dev mcp tool versions"),
		Safety:        devMCPReadSafety(),
		OutputRollout: output.RolloutUnifiedActive,
	}
}

func addDevMCPToolVersionsMethod(runner executor.Runner, parent *cobra.Command) error {
	builder, err := corecmd.NewServiceBuilder(parent)
	if err != nil {
		return err
	}
	return builder.Add(
		devMCPToolVersionsMethod(),
		corecmd.ToolBinding(devMCPToolVersionsTool, devMCPToolDispatch(runner, devMCPToolVersionsTool)),
	)
}
