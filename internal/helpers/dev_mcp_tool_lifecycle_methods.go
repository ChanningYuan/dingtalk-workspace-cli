package helpers

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/DingTalk-Real-AI/dingtalk-workspace-cli/internal/corecmd"
	"github.com/DingTalk-Real-AI/dingtalk-workspace-cli/internal/corecmd/contract"
	apperrors "github.com/DingTalk-Real-AI/dingtalk-workspace-cli/internal/errors"
	"github.com/DingTalk-Real-AI/dingtalk-workspace-cli/internal/executor"
	"github.com/DingTalk-Real-AI/dingtalk-workspace-cli/internal/output"
)

func devMCPToolDebugMethod() corecmd.MethodDefinition {
	return corecmd.MethodDefinition{
		Identity: corecmd.MethodIdentity{
			CLIPath:     "dev mcp tool debug",
			ProductID:   "dev",
			Name:        devMCPToolDebugTool,
			Short:       "调试 MCP 工具",
			Description: "使用指定输入和凭证调试 MCP 工具",
			Example:     "  dws dev mcp tool debug --mcp-id 10487 --tool-id G-ACT-xxx --value '{\"city\":\"杭州\"}' --credential-id 10518 --dry-run --format json",
		},
		PostMount: devMCPToolReadPostMount(devMCPToolDebugTool),
		Validate:  validateDevMCPToolDebug,
		Input: corecmd.InputSpec{Parameters: []corecmd.ParameterSpec{
			{Name: "mcp-id", Type: corecmd.KindInt, Usage: "MCP 服务 ID", Bind: "mcpId"},
			{Name: "tool-id", Type: corecmd.KindString, Trim: true, Usage: "MCP 工具 ID，G-ACT- 开头", Bind: "toolId"},
			{Name: "action-id", Type: corecmd.KindString, Usage: "已更名为 --tool-id", Bind: "actionId", Hidden: true, OmitEmpty: true},
			{Name: "value", Type: corecmd.KindString, Usage: "调试入参 JSON 对象，结构须符合工具 toolInputs 定义；不要传空 {} 走过场", Bind: "value"},
			{Name: "version-id", Type: corecmd.KindString, Usage: "指定调试的版本 ID", Bind: "versionId", OmitEmpty: true},
			{Name: "credential-id", Type: corecmd.KindInt, Usage: "凭证账号 ID（credential list 可查）；服务已配置鉴权时必须指定，作为本次调试的实际运行时鉴权（debug 不吃 bind 绑定的凭证；缺省不传不会被直接拦，会降级空跑、下游返回 40014 等误导报错）", Bind: "credentialId", OmitEmpty: true},
			{Name: "no-credential", Type: corecmd.KindBool, Usage: "无鉴权工具的正常走法——声明本次调试不使用凭证（与 --credential-id 二选一必填其一）", Bind: "noCredential", OmitEmpty: true},
		}},
		Contract:      devMCPToolContractSpec(devMCPToolDebugTool, "使用指定输入和凭证调试 MCP 工具", true, "dws dev mcp tool debug"),
		Safety:        devMCPWriteSafety(),
		OutputRollout: output.RolloutUnifiedActive,
	}
}

func devMCPToolPublishMethod() corecmd.MethodDefinition {
	return devMCPToolLocatorMethod(
		"publish",
		devMCPToolPublishTool,
		"发布 MCP 工具草稿",
		"发布 MCP 工具草稿",
		"  dws dev mcp tool publish --mcp-id 10487 --tool-id G-ACT-xxx --dry-run --format json",
		devMCPWriteSafety(),
	)
}

func devMCPToolDeleteMethod() corecmd.MethodDefinition {
	return devMCPToolLocatorMethod(
		"delete",
		devMCPToolDeleteTool,
		"删除 MCP 工具（不可恢复）",
		"永久删除 MCP 工具",
		"  dws dev mcp tool delete --mcp-id 10487 --tool-id G-ACT-xxx --dry-run --format json",
		devMCPDestructiveSafety(),
	)
}

func devMCPToolLocatorMethod(use, tool, short, description, example string, safety contract.SafetySpec) corecmd.MethodDefinition {
	return corecmd.MethodDefinition{
		Identity: corecmd.MethodIdentity{
			CLIPath:     "dev mcp tool " + use,
			ProductID:   "dev",
			Name:        tool,
			Short:       short,
			Description: description,
			Example:     example,
		},
		PostMount: devMCPToolReadPostMount(tool),
		Validate:  validateDevMCPToolLocator,
		Input: corecmd.InputSpec{Parameters: []corecmd.ParameterSpec{
			{Name: "mcp-id", Type: corecmd.KindInt, Usage: "MCP 服务 ID", Bind: "mcpId"},
			{Name: "tool-id", Type: corecmd.KindString, Trim: true, Usage: "MCP 工具 ID，G-ACT- 开头", Bind: "toolId"},
			{Name: "action-id", Type: corecmd.KindString, Usage: "已更名为 --tool-id", Bind: "actionId", Hidden: true, OmitEmpty: true},
		}},
		Contract:      devMCPToolContractSpec(tool, description, true, "dws dev mcp tool "+use),
		Safety:        safety,
		OutputRollout: output.RolloutUnifiedActive,
	}
}

func addDevMCPToolLifecycleMethods(runner executor.Runner, parent *cobra.Command) error {
	builder, err := corecmd.NewServiceBuilder(parent)
	if err != nil {
		return err
	}
	for _, method := range []struct {
		definition corecmd.MethodDefinition
		binding    corecmd.MethodBinding
	}{
		{devMCPToolDebugMethod(), devMCPToolDebugBinding(runner)},
		{devMCPToolPublishMethod(), devMCPToolPublishBinding(runner)},
		{devMCPToolDeleteMethod(), corecmd.ToolBinding(devMCPToolDeleteTool, devMCPToolDispatch(runner, devMCPToolDeleteTool))},
	} {
		if err := builder.Add(method.definition, method.binding); err != nil {
			return err
		}
	}
	return nil
}

func devMCPToolDebugBinding(runner executor.Runner) corecmd.MethodBinding {
	return corecmd.MethodBinding{
		Invoke: func(ctx *corecmd.Ctx, _ map[string]any) error {
			cmd := ctx.Command()
			params, err := devMCPToolLocatorParams(cmd)
			if err != nil {
				return err
			}
			value, err := devMCPRequiredJSONObjectFlag(cmd, "value")
			if err != nil {
				return err
			}
			params["value"] = value
			devMCPPutString(params, "versionId", devMCPStringFlag(cmd, "version-id"))
			credentialID := devMCPIntFlag(cmd, "credential-id")
			if credentialID == 0 && !commandBoolFlag(cmd, "no-credential") {
				return apperrors.NewValidation("调试需指定本次运行时凭证，二选一：① 有鉴权工具 → --credential-id <id>（credential list 可查）；② 无鉴权工具 → --no-credential。debug 只认这里传的凭证（不吃 bind 绑定的）；有鉴权却漏传不会被直接拦，而是降级空跑、由下游返回 40014 / access_token is blank 等\"貌似 token 坏、实为没给凭证\"的误导报错。")
			}
			devMCPPutInt(params, "credentialId", credentialID)
			return runDevMCPTool(runner, cmd, devMCPToolDebugTool, params)
		},
	}
}

func devMCPToolPublishBinding(runner executor.Runner) corecmd.MethodBinding {
	return corecmd.MethodBinding{
		Invoke: func(ctx *corecmd.Ctx, _ map[string]any) error {
			cmd := ctx.Command()
			params, err := devMCPToolLocatorParams(cmd)
			if err != nil {
				return err
			}
			if !commandDryRun(cmd) {
				if err := devMCPPublishPreflight(runner, cmd, params); err != nil {
					return err
				}
			}
			if err := runDevMCPTool(runner, cmd, devMCPToolPublishTool, params); err != nil {
				return err
			}
			if !commandDryRun(cmd) {
				fmt.Fprintln(cmd.ErrOrStderr(), "提示：发布后可通过 dws mcp published tools/invoke 按 mcpId 使用；serverName 可用 service get 查询。发布实例按组织与当前登录身份隔离，跨组织调用需切换 --profile。")
			}
			return nil
		},
	}
}
