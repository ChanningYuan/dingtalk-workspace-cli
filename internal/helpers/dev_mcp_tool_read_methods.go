package helpers

import (
	"github.com/spf13/cobra"

	"github.com/DingTalk-Real-AI/dingtalk-workspace-cli/internal/corecmd"
	"github.com/DingTalk-Real-AI/dingtalk-workspace-cli/internal/corecmd/contract"
	"github.com/DingTalk-Real-AI/dingtalk-workspace-cli/internal/executor"
	"github.com/DingTalk-Real-AI/dingtalk-workspace-cli/internal/output"
)

func devMCPToolListMethod() corecmd.MethodDefinition {
	return corecmd.MethodDefinition{
		Identity: corecmd.MethodIdentity{
			CLIPath:     "dev mcp tool list",
			ProductID:   "dev",
			Name:        devMCPToolListTool,
			Short:       "查询 MCP 服务下的工具列表",
			Description: "查询 MCP 服务下的工具定义列表",
			Example:     "  dws dev mcp tool list --mcp-id 10487 --page-size 100 --format json",
		},
		PostMount: devMCPToolReadPostMount(devMCPToolListTool),
		Validate:  validateDevMCPToolList,
		Input: corecmd.InputSpec{
			Parameters: []corecmd.ParameterSpec{
				{Name: "mcp-id", Type: corecmd.KindInt, Usage: "MCP 服务 ID", Bind: "mcpId"},
				{Name: "cursor", Type: corecmd.KindInt, Usage: "分页游标，从 1 开始", Bind: "cursor", OmitEmpty: true},
				{Name: "page-size", Type: corecmd.KindInt, Usage: "每页条数，最大 100", Bind: "pageSize", OmitEmpty: true},
				{Name: "keyword", Type: corecmd.KindString, Trim: true, Usage: "按工具 name 关键词过滤", Bind: "keyword", OmitEmpty: true},
			},
		},
		Contract:      devMCPToolContractSpec(devMCPToolListTool, "查询 MCP 服务下的工具定义列表", false, "dws dev mcp tool list"),
		Safety:        devMCPReadSafety(),
		OutputRollout: output.RolloutUnifiedActive,
	}
}

func devMCPToolGetMethod() corecmd.MethodDefinition {
	return corecmd.MethodDefinition{
		Identity: corecmd.MethodIdentity{
			CLIPath:     "dev mcp tool get",
			ProductID:   "dev",
			Name:        devMCPToolGetTool,
			Short:       "读取 MCP 工具定义",
			Description: "读取指定 MCP 工具定义",
			Example:     "  dws dev mcp tool get --mcp-id 10487 --tool-id G-ACT-xxx --format json",
		},
		PostMount: devMCPToolReadPostMount(devMCPToolGetTool),
		Validate:  validateDevMCPToolGet,
		Input: corecmd.InputSpec{
			Parameters: []corecmd.ParameterSpec{
				{Name: "mcp-id", Type: corecmd.KindInt, Usage: "MCP 服务 ID", Bind: "mcpId"},
				{Name: "tool-id", Type: corecmd.KindString, Trim: true, Usage: "MCP 工具 ID，G-ACT- 开头", Bind: "toolId"},
				{Name: "action-id", Type: corecmd.KindString, Usage: "已更名为 --tool-id", Bind: "actionId", Hidden: true, OmitEmpty: true},
				{Name: "version-id", Type: corecmd.KindString, Trim: true, Usage: "指定读取的历史版本 ID", Bind: "versionId", OmitEmpty: true},
			},
		},
		Contract:      devMCPToolContractSpec(devMCPToolGetTool, "读取指定 MCP 工具定义", false, "dws dev mcp tool get"),
		Safety:        devMCPReadSafety(),
		OutputRollout: output.RolloutUnifiedActive,
	}
}

func addDevMCPToolReadMethods(runner executor.Runner, parent *cobra.Command) error {
	builder, err := corecmd.NewServiceBuilder(parent)
	if err != nil {
		return err
	}
	for _, method := range []struct {
		definition corecmd.MethodDefinition
		binding    corecmd.MethodBinding
	}{
		{devMCPToolListMethod(), corecmd.ToolBinding(devMCPToolListTool, devMCPToolDispatch(runner, devMCPToolListTool))},
		{devMCPToolGetMethod(), corecmd.ToolBinding(devMCPToolGetTool, devMCPToolDispatch(runner, devMCPToolGetTool))},
	} {
		if err := builder.Add(method.definition, method.binding); err != nil {
			return err
		}
	}
	return nil
}

func devMCPToolReadPostMount(tool string) func(*cobra.Command) {
	return func(cmd *cobra.Command) {
		preferLegacyLeaf(cmd)
		annotateDevMCPTool(cmd, tool)
	}
}

func devMCPToolDispatch(runner executor.Runner, tool string) func(*corecmd.Ctx, map[string]any) error {
	return func(ctx *corecmd.Ctx, toolArgs map[string]any) error {
		return runDevMCPTool(runner, ctx.Command(), tool, toolArgs)
	}
}

func validateDevMCPToolList(cmd *cobra.Command, _ []string) error {
	_, err := devMCPRequiredInt(cmd, "mcp-id")
	return err
}

func validateDevMCPToolGet(cmd *cobra.Command, _ []string) error {
	_, err := devMCPToolLocatorParams(cmd)
	return err
}

func devMCPToolContractSpec(tool, description string, dryRun bool, example string) corecmd.ContractSpec {
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
