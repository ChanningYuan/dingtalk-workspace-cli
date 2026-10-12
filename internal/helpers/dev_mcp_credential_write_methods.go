package helpers

import (
	"github.com/spf13/cobra"

	"github.com/DingTalk-Real-AI/dingtalk-workspace-cli/internal/corecmd"
	"github.com/DingTalk-Real-AI/dingtalk-workspace-cli/internal/corecmd/contract"
	"github.com/DingTalk-Real-AI/dingtalk-workspace-cli/internal/executor"
	"github.com/DingTalk-Real-AI/dingtalk-workspace-cli/internal/output"
)

func devMCPCredentialDebugMethod() corecmd.MethodDefinition {
	return devMCPCredentialLocatorMethod(
		"debug",
		devMCPCredentialDebugTool,
		"调试 MCP 凭证账号（会真实调用下游接口，TOKEN 型含现场换 token）",
		"调试 MCP 凭证账号并验证下游鉴权",
		"  dws dev mcp credential debug --mcp-id 10520 --credential-id 10001 --dry-run --format json",
		devMCPWriteSafety(),
	)
}

func devMCPCredentialBindMethod() corecmd.MethodDefinition {
	return devMCPCredentialLocatorMethod(
		"bind",
		devMCPCredentialBindTool,
		"绑定 MCP 凭证账号",
		"绑定 MCP 发布实例使用的凭证账号",
		"  dws dev mcp credential bind --mcp-id 10520 --credential-id 10001 --dry-run --format json",
		devMCPWriteSafety(),
	)
}

func devMCPCredentialDeleteMethod() corecmd.MethodDefinition {
	return devMCPCredentialLocatorMethod(
		"delete",
		devMCPCredentialDeleteTool,
		"删除 MCP 凭证账号（不可恢复）",
		"永久删除 MCP 凭证账号",
		"  dws dev mcp credential delete --mcp-id 10520 --credential-id 10001 --dry-run --format json",
		devMCPDestructiveSafety(),
	)
}

func devMCPCredentialLocatorMethod(use, tool, short, description, example string, safety contract.SafetySpec) corecmd.MethodDefinition {
	return corecmd.MethodDefinition{
		Identity: corecmd.MethodIdentity{
			CLIPath:     "dev mcp credential " + use,
			ProductID:   "dev",
			Name:        tool,
			Short:       short,
			Description: description,
			Example:     example,
		},
		PostMount: devMCPToolReadPostMount(tool),
		Validate:  validateDevMCPCredentialLocator,
		Input: corecmd.InputSpec{Parameters: []corecmd.ParameterSpec{
			{Name: "mcp-id", Type: corecmd.KindInt, Usage: "MCP 服务 ID", Bind: "mcpId"},
			{Name: "credential-id", Type: corecmd.KindInt, Usage: "凭证账号 ID", Bind: "credentialId"},
		}},
		Contract:      devMCPToolContractSpec(tool, description, true, "dws dev mcp credential "+use),
		Safety:        safety,
		OutputRollout: output.RolloutUnifiedActive,
	}
}

func devMCPCredentialUnbindMethod() corecmd.MethodDefinition {
	return corecmd.MethodDefinition{
		Identity: corecmd.MethodIdentity{
			CLIPath:     "dev mcp credential unbind",
			ProductID:   "dev",
			Name:        devMCPCredentialUnbindTool,
			Short:       "解绑发布实例的生效凭证（bind 的逆操作；credential delete 报 credential_in_use 时先走本命令）",
			Description: "解绑 MCP 发布实例当前使用的凭证账号",
			Example:     "  dws dev mcp credential unbind --mcp-id 10520 --dry-run --format json",
		},
		PostMount: devMCPToolReadPostMount(devMCPCredentialUnbindTool),
		Validate:  validateDevMCPRequiredIntFlag("mcp-id"),
		Input: corecmd.InputSpec{Parameters: []corecmd.ParameterSpec{
			{Name: "mcp-id", Type: corecmd.KindInt, Usage: "MCP 服务 ID", Bind: "mcpId"},
		}},
		Contract:      devMCPToolContractSpec(devMCPCredentialUnbindTool, "解绑 MCP 发布实例当前使用的凭证账号", true, "dws dev mcp credential unbind"),
		Safety:        devMCPWriteSafety(),
		OutputRollout: output.RolloutUnifiedActive,
	}
}

func devMCPCredentialSaveMethod() corecmd.MethodDefinition {
	return corecmd.MethodDefinition{
		Identity: corecmd.MethodIdentity{
			CLIPath:     "dev mcp credential save",
			ProductID:   "dev",
			Name:        devMCPCredentialSaveTool,
			Short:       "新增或修改 MCP 凭证账号（TOKEN 型会现场调换 token 接口验密钥，密钥无效则保存失败）",
			Description: "新增或更新 MCP 凭证账号",
			Example:     `  dws dev mcp credential save --mcp-id 10520 --name 示例账号 --content '{"apiKey":"example"}' --dry-run --format json`,
		},
		PostMount: devMCPToolReadPostMount(devMCPCredentialSaveTool),
		Validate:  validateDevMCPCredentialSave,
		Input: corecmd.InputSpec{Parameters: []corecmd.ParameterSpec{
			{Name: "mcp-id", Type: corecmd.KindInt, Usage: "MCP 服务 ID", Bind: "mcpId"},
			{Name: "credential-id", Type: corecmd.KindInt, Usage: "已有凭证账号 ID；不传表示新增", Bind: "credentialId", OmitEmpty: true},
			{Name: "name", Type: corecmd.KindString, Trim: true, Usage: "凭证账号名称", Bind: "name"},
			{Name: "content", Type: corecmd.KindString, Usage: "密钥键值 JSON 对象；推荐改用 --content-file", Bind: "content", OmitEmpty: true},
			{Name: "content-file", Type: corecmd.KindString, Usage: "密钥键值 JSON 文件路径，传 - 从 stdin 读取", Bind: "contentFile", OmitEmpty: true},
		}},
		Contract:      devMCPToolContractSpec(devMCPCredentialSaveTool, "新增或更新 MCP 凭证账号", true, "dws dev mcp credential save"),
		Safety:        devMCPWriteSafety(),
		OutputRollout: output.RolloutUnifiedActive,
	}
}

func addDevMCPCredentialWriteMethods(runner executor.Runner, parent *cobra.Command) error {
	builder, err := corecmd.NewServiceBuilder(parent)
	if err != nil {
		return err
	}
	for _, method := range []struct {
		definition corecmd.MethodDefinition
		binding    corecmd.MethodBinding
	}{
		{devMCPCredentialSaveMethod(), devMCPCredentialSaveBinding(runner)},
		{devMCPCredentialDebugMethod(), corecmd.ToolBinding(devMCPCredentialDebugTool, devMCPToolDispatch(runner, devMCPCredentialDebugTool))},
		{devMCPCredentialBindMethod(), corecmd.ToolBinding(devMCPCredentialBindTool, devMCPToolDispatch(runner, devMCPCredentialBindTool))},
		{devMCPCredentialUnbindMethod(), corecmd.ToolBinding(devMCPCredentialUnbindTool, devMCPToolDispatch(runner, devMCPCredentialUnbindTool))},
		{devMCPCredentialDeleteMethod(), corecmd.ToolBinding(devMCPCredentialDeleteTool, devMCPToolDispatch(runner, devMCPCredentialDeleteTool))},
	} {
		if err := builder.Add(method.definition, method.binding); err != nil {
			return err
		}
	}
	return nil
}

func devMCPCredentialSaveBinding(runner executor.Runner) corecmd.MethodBinding {
	return corecmd.MethodBinding{
		Invoke: func(ctx *corecmd.Ctx, _ map[string]any) error {
			cmd := ctx.Command()
			mcpID, err := devMCPRequiredInt(cmd, "mcp-id")
			if err != nil {
				return err
			}
			name, err := devMCPRequiredString(cmd, "name")
			if err != nil {
				return err
			}
			content, err := devMCPCredentialContent(cmd)
			if err != nil {
				return err
			}
			params := map[string]any{"mcpId": mcpID, "name": name, "content": content}
			devMCPPutInt(params, "credentialId", devMCPIntFlag(cmd, "credential-id"))
			if commandDryRun(cmd) {
				params["content"] = map[string]any{"redacted": true}
			}
			return runDevMCPTool(runner, cmd, devMCPCredentialSaveTool, params)
		},
	}
}
