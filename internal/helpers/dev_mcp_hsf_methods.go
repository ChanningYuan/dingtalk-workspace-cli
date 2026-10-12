package helpers

import (
	"github.com/spf13/cobra"

	"github.com/DingTalk-Real-AI/dingtalk-workspace-cli/internal/corecmd"
	"github.com/DingTalk-Real-AI/dingtalk-workspace-cli/internal/executor"
	"github.com/DingTalk-Real-AI/dingtalk-workspace-cli/internal/output"
)

func devMCPHsfMethodListMethod() corecmd.MethodDefinition {
	return corecmd.MethodDefinition{
		Identity: corecmd.MethodIdentity{
			CLIPath:     "dev mcp hsf method-list",
			ProductID:   "dev",
			Name:        devMCPHsfMethodListTool,
			Short:       "查询 HSF 接口的方法清单（建 hsf 工具前的方法发现，含每方法出入参 schema）",
			Description: "查询 HSF 接口方法和参数 Schema",
			Example:     "  dws dev mcp hsf method-list --interface-name com.dingtalk.open.connect.workbench.api.service.hsf.MCPHsfService --format json",
		},
		PostMount: devMCPToolReadPostMount(devMCPHsfMethodListTool),
		Validate: func(cmd *cobra.Command, _ []string) error {
			_, err := devMCPRequiredString(cmd, "interface-name")
			return err
		},
		Input: corecmd.InputSpec{Parameters: []corecmd.ParameterSpec{
			{Name: "interface-name", Type: corecmd.KindString, Trim: true, Usage: "必填。HSF 接口全限定名（不能只传简名），如 com.dingtalk.open.connect.workbench.api.service.hsf.MCPHsfService", Bind: "interfaceName"},
			{Name: "version", Type: corecmd.KindString, Trim: true, Usage: "可选。HSF 服务版本号，缺省 1.0.0", Bind: "version", OmitEmpty: true},
		}},
		Contract:      devMCPToolContractSpec(devMCPHsfMethodListTool, "查询 HSF 接口方法和参数 Schema", false, "dws dev mcp hsf method-list"),
		Safety:        devMCPReadSafety(),
		OutputRollout: output.RolloutUnifiedActive,
	}
}

func devMCPToolCreateHsfMethod() corecmd.MethodDefinition {
	return corecmd.MethodDefinition{
		Identity: corecmd.MethodIdentity{
			CLIPath:     "dev mcp tool create-hsf",
			ProductID:   "dev",
			Name:        devMCPToolCreateHsfTool,
			Short:       "新建 HSF 型 MCP 工具草稿（apiInputs/apiOutputs 由服务端按方法 schema 自动生成）",
			Description: "创建 HSF 类型的 MCP 工具草稿",
			Example:     `  dws dev mcp tool create-hsf --mcp-id 10520 --name search_mcp_services --title 搜索MCP服务 --description 按名称关键词搜索MCP服务 --hsf-info '{"interfaceName":"com.dingtalk.open.connect.workbench.api.service.hsf.MCPHsfService","methodName":"searchMCPs","version":"1.0.0"}' --tool-inputs '[]' --input-mappings '[]' --tool-outputs '[]' --output-mappings '[]' --dry-run --format json`,
		},
		PostMount: devMCPToolReadPostMount(devMCPToolCreateHsfTool),
		Validate:  validateDevMCPHsfCreate,
		Input:     corecmd.InputSpec{Parameters: devMCPHsfCreateParameterSpecs()},
		Contract:  devMCPToolContractSpec(devMCPToolCreateHsfTool, "创建 HSF 类型的 MCP 工具草稿", true, "dws dev mcp tool create-hsf"),
		Safety:    devMCPWriteSafety(), OutputRollout: output.RolloutUnifiedActive,
	}
}

func devMCPToolUpdateHsfMethod() corecmd.MethodDefinition {
	return corecmd.MethodDefinition{
		Identity: corecmd.MethodIdentity{
			CLIPath:     "dev mcp tool update-hsf",
			ProductID:   "dev",
			Name:        devMCPToolUpdateHsfTool,
			Short:       "编辑 HSF 型工具（部分更新语义：只传要改的字段，未传保持原值）",
			Description: "更新 HSF 类型的 MCP 工具草稿",
			Example:     "  dws dev mcp tool update-hsf --mcp-id 10520 --tool-id G-ACT-xxx --description 更准确的新描述 --dry-run --format json",
		},
		PostMount: devMCPToolReadPostMount(devMCPToolUpdateHsfTool),
		Validate:  validateDevMCPHsfUpdate,
		Input:     corecmd.InputSpec{Parameters: devMCPHsfUpdateParameterSpecs()},
		Contract:  devMCPToolContractSpec(devMCPToolUpdateHsfTool, "更新 HSF 类型的 MCP 工具草稿", true, "dws dev mcp tool update-hsf"),
		Safety:    devMCPWriteSafety(), OutputRollout: output.RolloutUnifiedActive,
	}
}

func devMCPHsfCreateParameterSpecs() []corecmd.ParameterSpec {
	return []corecmd.ParameterSpec{
		{Name: "mcp-id", Type: corecmd.KindInt, Usage: "MCP 服务 ID", Bind: "mcpId"},
		{Name: "name", Type: corecmd.KindString, Trim: true, Usage: "必填。工具唯一标识，snake_case、动词开头、语义明确", Bind: "name"},
		{Name: "title", Type: corecmd.KindString, Trim: true, Usage: "必填。工具中文标题", Bind: "title", OmitEmpty: true},
		{Name: "description", Type: corecmd.KindString, Trim: true, Usage: "必填。工具功能完整描述（何时用+返回什么）", Bind: "description", OmitEmpty: true},
		{Name: "hsf-info", Type: corecmd.KindString, Trim: true, Usage: "必填。HSF 三元组 JSON 对象 {interfaceName,methodName,version(可省，缺省1.0.0)}——先用 hsf method-list 发现方法；与 http 版的 --http-info 对仗", Bind: "hsfInfo"},
		{Name: "tool-inputs", Type: corecmd.KindString, Trim: true, Usage: "必填。暴露给 LLM 的入参字段树 JSON 数组（与 http 版同构）", Bind: "toolInputs"},
		{Name: "input-mappings", Type: corecmd.KindString, Trim: true, Usage: "必填。toolInputs→DTO 映射 JSON 数组", Bind: "inputMappings"},
		{Name: "tool-outputs", Type: corecmd.KindString, Trim: true, Usage: "必填。对外出参字段树 JSON 数组；不做精修显式传 '[]'", Bind: "toolOutputs"},
		{Name: "output-mappings", Type: corecmd.KindString, Trim: true, Usage: "必填。出参映射 JSON 数组", Bind: "outputMappings"},
		{Name: "timeout", Type: corecmd.KindInt, Usage: "可选。请求超时秒数（1-180 整数）", Bind: "timeout", OmitEmpty: true},
		{Name: "only-original-keys", Type: corecmd.KindBool, Usage: "可选。仅映射非空字段", Bind: "onlyOriginalKeys", OmitEmpty: true},
	}
}

func devMCPHsfUpdateParameterSpecs() []corecmd.ParameterSpec {
	return []corecmd.ParameterSpec{
		{Name: "mcp-id", Type: corecmd.KindInt, Usage: "MCP 服务 ID", Bind: "mcpId"},
		{Name: "tool-id", Type: corecmd.KindString, Trim: true, Usage: "MCP 工具 ID，G-ACT- 开头", Bind: "toolId"},
		{Name: "action-id", Type: corecmd.KindString, Usage: "已更名为 --tool-id", Bind: "actionId", Hidden: true, OmitEmpty: true},
		{Name: "name", Type: corecmd.KindString, Trim: true, Usage: "可选。改工具标识名（snake_case）；不传=保持原值", Bind: "name", OmitEmpty: true},
		{Name: "title", Type: corecmd.KindString, Trim: true, Usage: "可选。改标题；不传=保持原值", Bind: "title", OmitEmpty: true},
		{Name: "description", Type: corecmd.KindString, Trim: true, Usage: "可选。改工具描述；不传=保持原值", Bind: "description", OmitEmpty: true},
		{Name: "hsf-info", Type: corecmd.KindString, Trim: true, Usage: "可选。改 HSF 三元组 JSON 对象", Bind: "hsfInfo", OmitEmpty: true},
		{Name: "tool-inputs", Type: corecmd.KindString, Trim: true, Usage: "可选。改入参字段树；不传=保持原值，传了=整块替换", Bind: "toolInputs", OmitEmpty: true},
		{Name: "input-mappings", Type: corecmd.KindString, Trim: true, Usage: "可选。改入参映射；不传=保持原值，传了=整块替换", Bind: "inputMappings", OmitEmpty: true},
		{Name: "tool-outputs", Type: corecmd.KindString, Trim: true, Usage: "可选。改出参字段树；不传=保持原值，传了=整块替换", Bind: "toolOutputs", OmitEmpty: true},
		{Name: "output-mappings", Type: corecmd.KindString, Trim: true, Usage: "可选。改出参映射；不传=保持原值，传数组=整块替换", Bind: "outputMappings", OmitEmpty: true},
		{Name: "timeout", Type: corecmd.KindInt, Usage: "可选。请求超时秒数（1-180 整数）", Bind: "timeout", OmitEmpty: true},
		{Name: "only-original-keys", Type: corecmd.KindBool, Usage: "可选。仅映射非空字段开关；漏传=保留原值", Bind: "onlyOriginalKeys", OmitEmpty: true},
	}
}

func addDevMCPHsfMethodListMethod(runner executor.Runner, parent *cobra.Command) error {
	builder, err := corecmd.NewServiceBuilder(parent)
	if err != nil {
		return err
	}
	return builder.Add(
		devMCPHsfMethodListMethod(),
		corecmd.ToolBinding(devMCPHsfMethodListTool, devMCPToolDispatch(runner, devMCPHsfMethodListTool)),
	)
}

func addDevMCPToolHsfMethods(runner executor.Runner, parent *cobra.Command) error {
	builder, err := corecmd.NewServiceBuilder(parent)
	if err != nil {
		return err
	}
	for _, method := range []struct {
		definition corecmd.MethodDefinition
		binding    corecmd.MethodBinding
	}{
		{devMCPToolCreateHsfMethod(), devMCPHsfInvokeBinding(runner, devMCPToolCreateHsfTool, devMCPToolHsfCreateParams)},
		{devMCPToolUpdateHsfMethod(), devMCPHsfInvokeBinding(runner, devMCPToolUpdateHsfTool, devMCPToolHsfUpdateParams)},
	} {
		if err := builder.Add(method.definition, method.binding); err != nil {
			return err
		}
	}
	return nil
}

func devMCPHsfInvokeBinding(runner executor.Runner, tool string, params func(*cobra.Command) (map[string]any, error)) corecmd.MethodBinding {
	return corecmd.MethodBinding{Invoke: func(ctx *corecmd.Ctx, _ map[string]any) error {
		payload, err := params(ctx.Command())
		if err != nil {
			return err
		}
		return runDevMCPTool(runner, ctx.Command(), tool, payload)
	}}
}
