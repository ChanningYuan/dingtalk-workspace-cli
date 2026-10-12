package helpers

import (
	"github.com/spf13/cobra"

	"github.com/DingTalk-Real-AI/dingtalk-workspace-cli/internal/corecmd"
	"github.com/DingTalk-Real-AI/dingtalk-workspace-cli/internal/executor"
	"github.com/DingTalk-Real-AI/dingtalk-workspace-cli/internal/output"
)

func devMCPCredentialListMethod() corecmd.MethodDefinition {
	return corecmd.MethodDefinition{
		Identity: corecmd.MethodIdentity{
			CLIPath:     "dev mcp credential list",
			ProductID:   "dev",
			Name:        devMCPCredentialListTool,
			Short:       "查询 MCP 凭证账号列表",
			Description: "查询 MCP 凭证账号列表",
			Example:     "  dws dev mcp credential list --mcp-id 10520 --page-size 20 --format json",
		},
		PostMount: devMCPToolReadPostMount(devMCPCredentialListTool),
		Validate:  validateDevMCPRequiredIntFlag("mcp-id"),
		Input: corecmd.InputSpec{Parameters: []corecmd.ParameterSpec{
			{Name: "mcp-id", Type: corecmd.KindInt, Usage: "MCP 服务 ID", Bind: "mcpId"},
			{Name: "cursor", Type: corecmd.KindInt, Usage: "分页游标，从 1 开始", Bind: "cursor", OmitEmpty: true},
			{Name: "page-size", Type: corecmd.KindInt, Usage: "每页条数，最大 100", Bind: "pageSize", OmitEmpty: true},
		}},
		Contract:      devMCPToolContractSpec(devMCPCredentialListTool, "查询 MCP 凭证账号列表", false, "dws dev mcp credential list"),
		Safety:        devMCPReadSafety(),
		OutputRollout: output.RolloutUnifiedActive,
	}
}

func devMCPCredentialGetMethod() corecmd.MethodDefinition {
	return corecmd.MethodDefinition{
		Identity: corecmd.MethodIdentity{
			CLIPath:     "dev mcp credential get",
			ProductID:   "dev",
			Name:        devMCPCredentialGetTool,
			Short:       "查询 MCP 凭证账号详情",
			Description: "查询 MCP 凭证账号详情",
			Example:     "  dws dev mcp credential get --mcp-id 10520 --credential-id 10001 --format json",
		},
		PostMount: devMCPToolReadPostMount(devMCPCredentialGetTool),
		Validate:  validateDevMCPCredentialLocator,
		Input: corecmd.InputSpec{Parameters: []corecmd.ParameterSpec{
			{Name: "mcp-id", Type: corecmd.KindInt, Usage: "MCP 服务 ID", Bind: "mcpId"},
			{Name: "credential-id", Type: corecmd.KindInt, Usage: "凭证账号 ID", Bind: "credentialId"},
		}},
		Contract:      devMCPToolContractSpec(devMCPCredentialGetTool, "查询 MCP 凭证账号详情", false, "dws dev mcp credential get"),
		Safety:        devMCPReadSafety(),
		OutputRollout: output.RolloutUnifiedActive,
	}
}

func addDevMCPCredentialReadMethods(runner executor.Runner, parent *cobra.Command) error {
	builder, err := corecmd.NewServiceBuilder(parent)
	if err != nil {
		return err
	}
	for _, method := range []struct {
		definition corecmd.MethodDefinition
		binding    corecmd.MethodBinding
	}{
		{devMCPCredentialListMethod(), corecmd.ToolBinding(devMCPCredentialListTool, devMCPToolDispatch(runner, devMCPCredentialListTool))},
		{devMCPCredentialGetMethod(), corecmd.ToolBinding(devMCPCredentialGetTool, devMCPToolDispatch(runner, devMCPCredentialGetTool))},
	} {
		if err := builder.Add(method.definition, method.binding); err != nil {
			return err
		}
	}
	return nil
}
