package helpers

import (
	"github.com/spf13/cobra"

	"github.com/DingTalk-Real-AI/dingtalk-workspace-cli/internal/corecmd"
	"github.com/DingTalk-Real-AI/dingtalk-workspace-cli/internal/executor"
	"github.com/DingTalk-Real-AI/dingtalk-workspace-cli/internal/output"
)

func devMCPMemberListMethod() corecmd.MethodDefinition {
	return corecmd.MethodDefinition{
		Identity: corecmd.MethodIdentity{
			CLIPath:     "dev mcp member list",
			ProductID:   "dev",
			Name:        devMCPMemberListTool,
			Short:       "查询 MCP 开发协作者列表",
			Description: "查询 MCP 服务开发协作者",
			Example:     "  dws dev mcp member list --mcp-id 10520 --format json",
		},
		PostMount: devMCPToolReadPostMount(devMCPMemberListTool),
		Validate:  validateDevMCPRequiredIntFlag("mcp-id"),
		Input: corecmd.InputSpec{Parameters: []corecmd.ParameterSpec{
			{Name: "mcp-id", Type: corecmd.KindInt, Usage: "MCP 服务 ID", Bind: "mcpId"},
		}},
		Contract:      devMCPToolContractSpec(devMCPMemberListTool, "查询 MCP 服务开发协作者", false, "dws dev mcp member list"),
		Safety:        devMCPReadSafety(),
		OutputRollout: output.RolloutUnifiedActive,
	}
}

func addDevMCPMemberListMethod(runner executor.Runner, parent *cobra.Command) error {
	builder, err := corecmd.NewServiceBuilder(parent)
	if err != nil {
		return err
	}
	return builder.Add(
		devMCPMemberListMethod(),
		corecmd.ToolBinding(devMCPMemberListTool, devMCPToolDispatch(runner, devMCPMemberListTool)),
	)
}
