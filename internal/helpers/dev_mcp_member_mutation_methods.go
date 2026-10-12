package helpers

import (
	"github.com/spf13/cobra"

	"github.com/DingTalk-Real-AI/dingtalk-workspace-cli/internal/corecmd"
	"github.com/DingTalk-Real-AI/dingtalk-workspace-cli/internal/corecmd/contract"
	"github.com/DingTalk-Real-AI/dingtalk-workspace-cli/internal/executor"
	"github.com/DingTalk-Real-AI/dingtalk-workspace-cli/internal/output"
)

func devMCPMemberAddMethod() corecmd.MethodDefinition {
	return devMCPMemberMutationMethod(
		"add",
		devMCPMemberAddTool,
		"新增 MCP 开发协作者",
		"新增 MCP 服务开发协作者",
		"  dws dev mcp member add --mcp-id 10520 --user-ids staff001,staff002 --dry-run --format json",
		devMCPWriteSafety(),
	)
}

func devMCPMemberRemoveMethod() corecmd.MethodDefinition {
	return devMCPMemberMutationMethod(
		"remove",
		devMCPMemberRemoveTool,
		"移除 MCP 开发协作者",
		"移除 MCP 服务开发协作者",
		"  dws dev mcp member remove --mcp-id 10520 --user-ids staff001 --dry-run --format json",
		devMCPDestructiveSafety(),
	)
}

func devMCPMemberMutationMethod(use, tool, short, description, example string, safety contract.SafetySpec) corecmd.MethodDefinition {
	return corecmd.MethodDefinition{
		Identity: corecmd.MethodIdentity{
			CLIPath:     "dev mcp member " + use,
			ProductID:   "dev",
			Name:        tool,
			Short:       short,
			Description: description,
			Example:     example,
		},
		PostMount: devMCPToolReadPostMount(tool),
		Validate:  validateDevMCPMemberMutation,
		Input: corecmd.InputSpec{Parameters: []corecmd.ParameterSpec{
			{Name: "mcp-id", Type: corecmd.KindInt, Usage: "MCP 服务 ID", Bind: "mcpId"},
			{Name: "user-ids", Type: corecmd.KindString, Usage: "成员 staffId 列表，多个用逗号或分号分隔", Bind: "memberUserIds", Transform: transformDevAppListParam, OmitEmpty: true},
		}},
		Contract:      devMCPToolContractSpec(tool, description, true, "dws dev mcp member "+use),
		Safety:        safety,
		OutputRollout: output.RolloutUnifiedActive,
	}
}

func addDevMCPMemberMutationMethods(runner executor.Runner, parent *cobra.Command) error {
	builder, err := corecmd.NewServiceBuilder(parent)
	if err != nil {
		return err
	}
	for _, method := range []struct {
		definition corecmd.MethodDefinition
		binding    corecmd.MethodBinding
	}{
		{devMCPMemberAddMethod(), corecmd.ToolBinding(devMCPMemberAddTool, devMCPToolDispatch(runner, devMCPMemberAddTool))},
		{devMCPMemberRemoveMethod(), corecmd.ToolBinding(devMCPMemberRemoveTool, devMCPToolDispatch(runner, devMCPMemberRemoveTool))},
	} {
		if err := builder.Add(method.definition, method.binding); err != nil {
			return err
		}
	}
	return nil
}
