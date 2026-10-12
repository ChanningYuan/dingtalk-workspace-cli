package helpers

import (
	"github.com/spf13/cobra"

	"github.com/DingTalk-Real-AI/dingtalk-workspace-cli/internal/corecmd"
	"github.com/DingTalk-Real-AI/dingtalk-workspace-cli/internal/executor"
	"github.com/DingTalk-Real-AI/dingtalk-workspace-cli/internal/output"
)

func devMCPToolCreateMethod() corecmd.MethodDefinition {
	return corecmd.MethodDefinition{
		Identity: corecmd.MethodIdentity{
			CLIPath:     "dev mcp tool create",
			ProductID:   "dev",
			Name:        devMCPToolCreateHTTPTool,
			Short:       "新建 MCP 工具草稿",
			Description: "创建 HTTP 类型的 MCP 工具草稿",
			Example:     "  dws dev mcp tool create --mcp-id 10487 --name get_weather --title 查询天气 --description 按经纬度查询实时天气 --http-info '{\"method\":\"GET\",\"url\":\"https://example.com\",\"auth\":{\"type\":\"NO_AUTH\"}}' --api-inputs '{\"query\":[{\"key\":\"lat\",\"type\":\"number\",\"description\":\"纬度\"}]}' --tool-inputs '[{\"key\":\"lat\",\"type\":\"number\",\"required\":true,\"description\":\"纬度，示例：39.9\"}]' --input-mappings '[{\"target\":\"$.Query.lat\",\"type\":\"reference\",\"source\":\"$.node_start.lat\"}]' --api-outputs '{\"body\":[{\"key\":\"temperature\",\"type\":\"number\",\"description\":\"温度\"}]}' --tool-outputs '[]' --output-mappings '[{\"target\":\"$\",\"type\":\"reference\",\"source\":\"$.node_service_activator.Body\"}]' --dry-run --format json",
		},
		PostMount:     devMCPToolReadPostMount(devMCPToolCreateHTTPTool),
		Validate:      validateDevMCPToolCreate,
		Input:         corecmd.InputSpec{Parameters: devMCPToolUpsertParameterSpecs(false)},
		Contract:      devMCPToolContractSpec(devMCPToolCreateHTTPTool, "创建 HTTP 类型的 MCP 工具草稿", true, "dws dev mcp tool create"),
		Safety:        devMCPWriteSafety(),
		OutputRollout: output.RolloutUnifiedActive,
	}
}

func devMCPToolUpdateMethod() corecmd.MethodDefinition {
	return corecmd.MethodDefinition{
		Identity: corecmd.MethodIdentity{
			CLIPath:     "dev mcp tool update",
			ProductID:   "dev",
			Name:        devMCPToolUpdateHTTPTool,
			Short:       "编辑 MCP 工具并保存为草稿",
			Description: "更新 HTTP 类型的 MCP 工具草稿",
			Example:     `  dws dev mcp tool update --mcp-id 10487 --tool-id G-ACT-example --name get_weather --title 查询天气 --description 按城市查询天气 --http-info '{"method":"GET","url":"https://example.com/weather","auth":{"type":"NO_AUTH"}}' --api-inputs '{"query":[{"key":"city","type":"string","description":"城市名"}]}' --tool-inputs '[{"key":"city","type":"string","required":true,"description":"城市名，例如杭州"}]' --input-mappings '[{"target":"$.Query.city","type":"reference","source":"$.node_start.city"}]' --api-outputs '{"body":[{"key":"temperature","type":"number","description":"温度"}]}' --tool-outputs '[]' --output-mappings '[{"target":"$","type":"reference","source":"$.node_service_activator.Body"}]' --dry-run --format json`,
		},
		PostMount:     devMCPToolReadPostMount(devMCPToolUpdateHTTPTool),
		Validate:      validateDevMCPToolUpdate,
		Input:         corecmd.InputSpec{Parameters: devMCPToolUpsertParameterSpecs(true)},
		Contract:      devMCPToolContractSpec(devMCPToolUpdateHTTPTool, "更新 HTTP 类型的 MCP 工具草稿", true, "dws dev mcp tool update"),
		Safety:        devMCPWriteSafety(),
		OutputRollout: output.RolloutUnifiedActive,
	}
}

func addDevMCPToolUpsertMethods(runner executor.Runner, parent *cobra.Command) error {
	builder, err := corecmd.NewServiceBuilder(parent)
	if err != nil {
		return err
	}
	for _, method := range []struct {
		definition corecmd.MethodDefinition
		binding    corecmd.MethodBinding
	}{
		{devMCPToolCreateMethod(), devMCPToolUpsertBinding(runner, devMCPToolCreateHTTPTool, false)},
		{devMCPToolUpdateMethod(), devMCPToolUpsertBinding(runner, devMCPToolUpdateHTTPTool, true)},
	} {
		if err := builder.Add(method.definition, method.binding); err != nil {
			return err
		}
	}
	return nil
}

func devMCPToolUpsertBinding(runner executor.Runner, tool string, includeToolID bool) corecmd.MethodBinding {
	return corecmd.MethodBinding{
		Invoke: func(ctx *corecmd.Ctx, _ map[string]any) error {
			// The legacy parser preserves Changed and explicit-empty JSON semantics.
			params, err := devMCPToolUpsertParams(ctx.Command(), includeToolID)
			if err != nil {
				return err
			}
			return runDevMCPTool(runner, ctx.Command(), tool, params)
		},
	}
}

func devMCPToolUpsertParameterSpecs(includeToolID bool) []corecmd.ParameterSpec {
	parameters := []corecmd.ParameterSpec{
		{Name: "mcp-id", Type: corecmd.KindInt, Usage: "MCP 服务 ID", Bind: "mcpId"},
	}
	if includeToolID {
		parameters = append(parameters,
			corecmd.ParameterSpec{Name: "tool-id", Type: corecmd.KindString, Trim: true, Usage: "MCP 工具 ID，G-ACT- 开头", Bind: "toolId"},
			corecmd.ParameterSpec{Name: "action-id", Type: corecmd.KindString, Usage: "已更名为 --tool-id", Bind: "actionId", Hidden: true, OmitEmpty: true},
		)
	}
	parameters = append(parameters,
		corecmd.ParameterSpec{Name: "name", Type: corecmd.KindString, Trim: true, Usage: "工具唯一标识，snake_case", Bind: "name"},
		corecmd.ParameterSpec{Name: "title", Type: corecmd.KindString, Trim: true, Usage: "必填。工具中文标题：中文自然语言、≤30 字、与功能一致", Bind: "title", OmitEmpty: true},
		corecmd.ParameterSpec{Name: "description", Type: corecmd.KindString, Trim: true, Usage: "必填。工具功能完整描述（LLM 选择工具的核心依据）：动词开头，说明功能/何时用/入参来源/破坏性行为", Bind: "description", OmitEmpty: true},
		corecmd.ParameterSpec{Name: "http-info", Type: corecmd.KindString, Trim: true, Usage: "必填。HTTP 接口配置 JSON 对象：{method,url,auth:{type,...}}", Bind: "httpInfo"},
		corecmd.ParameterSpec{Name: "http", Type: corecmd.KindString, Usage: "已更名为 --http-info", Bind: "http", Hidden: true, OmitEmpty: true},
		corecmd.ParameterSpec{Name: "api-inputs", Type: corecmd.KindString, Trim: true, Usage: "必填。接口真实入参 JSON 对象：{headers,body,query,path} 四组，每组=字段数组", Bind: "apiInputs", OmitEmpty: true},
		corecmd.ParameterSpec{Name: "api-outputs", Type: corecmd.KindString, Trim: true, Usage: "必填。接口真实出参 JSON 对象：{headers,body} 两组", Bind: "apiOutputs", OmitEmpty: true},
		corecmd.ParameterSpec{Name: "tool-inputs", Type: corecmd.KindString, Trim: true, Usage: "必填。暴露给 LLM 的入参字段树 JSON 数组", Bind: "toolInputs", OmitEmpty: true},
		corecmd.ParameterSpec{Name: "tool-outputs", Type: corecmd.KindString, Trim: true, Usage: "必填。暴露给 LLM 的出参字段树 JSON 数组", Bind: "toolOutputs", OmitEmpty: true},
		corecmd.ParameterSpec{Name: "input-mappings", Type: corecmd.KindString, Trim: true, Usage: "必填。入参映射 JSON 数组", Bind: "inputMappings", OmitEmpty: true},
		corecmd.ParameterSpec{Name: "output-mappings", Type: corecmd.KindString, Trim: true, Usage: "必填。出参映射 JSON 数组", Bind: "outputMappings", OmitEmpty: true},
		corecmd.ParameterSpec{Name: "timeout", Type: corecmd.KindInt, Usage: "请求超时秒数（1-180 整数）", Bind: "timeout", OmitEmpty: true},
		corecmd.ParameterSpec{Name: "only-original-keys", Type: corecmd.KindBool, Usage: "开启后出参只含下游实际返回的 key", Bind: "onlyOriginalKeys"},
	)
	return parameters
}
