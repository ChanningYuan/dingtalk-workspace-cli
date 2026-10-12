package helpers

import (
	"strings"

	"github.com/spf13/cobra"

	"github.com/DingTalk-Real-AI/dingtalk-workspace-cli/internal/corecmd"
	"github.com/DingTalk-Real-AI/dingtalk-workspace-cli/internal/executor"
	"github.com/DingTalk-Real-AI/dingtalk-workspace-cli/internal/output"
)

func devMCPAuthConfigSaveMethod() corecmd.MethodDefinition {
	return corecmd.MethodDefinition{
		Identity: corecmd.MethodIdentity{
			CLIPath:     "dev mcp auth save",
			ProductID:   "dev",
			Name:        devMCPAuthConfigSaveTool,
			Short:       "保存 MCP 下游鉴权配置",
			Description: "保存 MCP 服务的下游鉴权配置",
			Example: `  dws dev mcp auth save --mcp-id 10520 --auth-type NO_AUTH --dry-run --format json
  # 静态 API key（SIGNATURE 直引）：authQuery/authHeaders 的 value 用 #("<authFields 的 dataId>")，key 放 header 则把 authQuery 换成 authHeaders
  dws dev mcp auth save --mcp-id 10520 --auth-type SIGNATURE --signature-auth-config '{"authFields":[{"dataId":"apiKey","type":"password","required":true}],"authQuery":[{"key":"api_key","type":"authField","value":"#(\"apiKey\")"}]}' --dry-run --format json`,
		},
		PostMount: devMCPToolReadPostMount(devMCPAuthConfigSaveTool),
		Validate:  validateDevMCPAuthSave,
		Input: corecmd.InputSpec{Parameters: []corecmd.ParameterSpec{
			{Name: "mcp-id", Type: corecmd.KindInt, Usage: "MCP 服务 ID", Bind: "mcpId"},
			{Name: "auth-type", Type: corecmd.KindString, Trim: true, Usage: "鉴权类型：NO_AUTH、BASIC、TOKEN 或 SIGNATURE；静态 API key 场景用 SIGNATURE 自定义字段+直引", Bind: "authType", Transform: func(raw string) (any, error) { return strings.ToUpper(raw), nil }},
			{Name: "basic-auth-config", Type: corecmd.KindString, Usage: "BASIC 鉴权配置 JSON 对象", Bind: "basicAuthConfig", OmitEmpty: true},
			{Name: "api-secret-auth-config", Type: corecmd.KindString, Usage: "", Bind: "apiSecretAuthConfig", Hidden: true, OmitEmpty: true},
			{Name: "token-auth-config", Type: corecmd.KindString, Usage: "TOKEN 换取及注入配置 JSON 对象：{authFields, fetchTokenRequest, 注入位, tokenExpireRules, refreshToken, testRequest}；注入位按下游要求三选一：authHeaders（token 放请求头）/ authQuery（token 放 query 参数）/ authBody。引用密钥字段用 #(\"<dataId>\") 函数语法（不是 {{}} 模板）、引换 token 响应用 $.Body.<字段>；完整可跑模板见 skill mcp.md 鉴权节", Bind: "tokenAuthConfig", OmitEmpty: true},
			{Name: "signature-auth-config", Type: corecmd.KindString, Usage: "SIGNATURE 自定义鉴权配置 JSON 对象（静态 API key 直引 / 自定义签名表达式两类场景）。直引写法见上方 Examples 与 skill mcp.md：value 用 #(\"<authFields 的 dataId>\") 函数语法", Bind: "signatureAuthConfig", OmitEmpty: true},
		}},
		Contract:      devMCPToolContractSpec(devMCPAuthConfigSaveTool, "保存 MCP 服务的下游鉴权配置", true, "dws dev mcp auth save"),
		Safety:        devMCPWriteSafety(),
		OutputRollout: output.RolloutUnifiedActive,
	}
}

func addDevMCPAuthConfigSaveMethod(runner executor.Runner, parent *cobra.Command) error {
	builder, err := corecmd.NewServiceBuilder(parent)
	if err != nil {
		return err
	}
	return builder.Add(devMCPAuthConfigSaveMethod(), corecmd.MethodBinding{
		Invoke: func(ctx *corecmd.Ctx, _ map[string]any) error {
			cmd := ctx.Command()
			mcpID, err := devMCPRequiredInt(cmd, "mcp-id")
			if err != nil {
				return err
			}
			authType, err := devMCPRequiredString(cmd, "auth-type")
			if err != nil {
				return err
			}
			authType = strings.ToUpper(authType)
			params := map[string]any{"mcpId": mcpID, "authType": authType}
			for _, mapping := range []struct{ flag, key string }{
				{"basic-auth-config", "basicAuthConfig"},
				{"api-secret-auth-config", "apiSecretAuthConfig"},
				{"token-auth-config", "tokenAuthConfig"},
				{"signature-auth-config", "signatureAuthConfig"},
			} {
				if err := devMCPPutJSONObjectFlag(cmd, params, mapping.flag, mapping.key); err != nil {
					return err
				}
			}
			return runDevMCPTool(runner, cmd, devMCPAuthConfigSaveTool, params)
		},
	})
}
