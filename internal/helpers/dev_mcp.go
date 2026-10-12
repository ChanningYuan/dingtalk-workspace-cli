// Copyright 2026 Alibaba Group
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package helpers

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/DingTalk-Real-AI/dingtalk-workspace-cli/internal/cobracmd"
	apperrors "github.com/DingTalk-Real-AI/dingtalk-workspace-cli/internal/errors"
	"github.com/DingTalk-Real-AI/dingtalk-workspace-cli/internal/executor"
	"github.com/spf13/cobra"
)

const (
	devMCPProduct = "mcpdev"

	devMCPServerURLGetTool = "mcp_server_url_get"

	devMCPServiceCreateTool = "mcp_service_create"
	devMCPServiceDeleteTool = "mcp_service_delete"
	devMCPServiceGetTool    = "mcp_service_get"
	devMCPServiceListTool   = "mcp_service_list"
	devMCPServiceUpdateTool = "mcp_service_update"

	devMCPToolCreateHTTPTool = "mcp_tool_create_http"
	devMCPToolDebugTool      = "mcp_tool_debug"
	devMCPToolDeleteTool     = "mcp_tool_delete"
	devMCPToolGetTool        = "mcp_tool_get"
	devMCPToolListTool       = "mcp_tool_list"
	devMCPToolPublishTool    = "mcp_tool_publish"
	devMCPToolUpdateHTTPTool = "mcp_tool_update_http"
	devMCPToolVersionsTool   = "mcp_tool_versions"

	devMCPToolCreateHsfTool = "mcp_tool_create_hsf"
	devMCPToolUpdateHsfTool = "mcp_tool_update_hsf"
	devMCPHsfMethodListTool = "mcp_hsf_method_list"

	devMCPAuthConfigGetTool  = "mcp_auth_config_get"
	devMCPAuthConfigSaveTool = "mcp_auth_config_save"

	devMCPCredentialBindTool   = "mcp_credential_bind"
	devMCPCredentialDebugTool  = "mcp_credential_debug"
	devMCPCredentialDeleteTool = "mcp_credential_delete"
	devMCPCredentialGetTool    = "mcp_credential_get"
	devMCPCredentialListTool   = "mcp_credential_list"
	devMCPCredentialSaveTool   = "mcp_credential_save"
	devMCPCredentialUnbindTool = "mcp_credential_unbind"

	devMCPMemberAddTool    = "mcp_member_add"
	devMCPMemberListTool   = "mcp_member_list"
	devMCPMemberRemoveTool = "mcp_member_remove"
)

// newDevMCPCommand builds the reviewed static `mcp` authoring subtree under
// `dws dev`. Cobra owns the ergonomic CLI paths while ContractFinal metadata
// owns identity, safety, and interface disposition.
func newDevMCPCommand(runner executor.Runner, parent *cobra.Command) *cobra.Command {
	root := &cobra.Command{
		Use:               "mcp",
		Short:             "MCP 服务与工具管理",
		Long:              "管理 MCP 开发平台服务和工具：服务创建/查询/更新/删除，工具创建/查询/调试/发布/版本管理，以及接入地址获取。",
		Args:              cobra.NoArgs,
		TraverseChildren:  true,
		DisableAutoGenTag: true,
	}
	newGroupCommand(root)
	parent.AddCommand(root)

	service := &cobra.Command{
		Use:               "service",
		Short:             "MCP 服务管理",
		Args:              cobra.NoArgs,
		TraverseChildren:  true,
		DisableAutoGenTag: true,
	}
	newGroupCommand(service)

	tool := &cobra.Command{
		Use:               "tool",
		Short:             "MCP 工具管理",
		Args:              cobra.NoArgs,
		TraverseChildren:  true,
		DisableAutoGenTag: true,
	}
	newGroupCommand(tool)

	url := &cobra.Command{
		Use:               "url",
		Short:             "MCP 接入地址",
		Args:              cobra.NoArgs,
		TraverseChildren:  true,
		DisableAutoGenTag: true,
	}
	newGroupCommand(url)

	auth := newDevMCPCommandGroup("auth", "MCP 下游鉴权配置")
	credential := newDevMCPCommandGroup("credential", "MCP 凭证账号管理")
	member := newDevMCPCommandGroup("member", "MCP 开发协作者管理")
	hsf := newDevMCPCommandGroup("hsf", "HSF 方法发现（建 hsf 型工具前查方法与 DTO 字段名）")

	root.AddCommand(service, tool, url, auth, credential, member, hsf)

	if err := addDevMCPServiceMethods(runner, service); err != nil {
		panic(fmt.Sprintf("dev mcp service methods: %v", err))
	}
	if err := addDevMCPToolReadMethods(runner, tool); err != nil {
		panic(fmt.Sprintf("dev mcp tool read methods: %v", err))
	}
	if err := addDevMCPToolVersionsMethod(runner, tool); err != nil {
		panic(fmt.Sprintf("dev mcp tool versions method: %v", err))
	}
	if err := addDevMCPToolUpsertMethods(runner, tool); err != nil {
		panic(fmt.Sprintf("dev mcp tool upsert methods: %v", err))
	}
	if err := addDevMCPToolLifecycleMethods(runner, tool); err != nil {
		panic(fmt.Sprintf("dev mcp tool lifecycle methods: %v", err))
	}
	if err := addDevMCPToolHsfMethods(runner, tool); err != nil {
		panic(fmt.Sprintf("dev mcp tool hsf methods: %v", err))
	}
	if err := addDevMCPURLGetMethod(runner, url); err != nil {
		panic(fmt.Sprintf("dev mcp url get method: %v", err))
	}
	if err := addDevMCPAuthGetMethod(runner, auth); err != nil {
		panic(fmt.Sprintf("dev mcp auth get method: %v", err))
	}
	if err := addDevMCPAuthConfigSaveMethod(runner, auth); err != nil {
		panic(fmt.Sprintf("dev mcp auth save method: %v", err))
	}
	if err := addDevMCPCredentialReadMethods(runner, credential); err != nil {
		panic(fmt.Sprintf("dev mcp credential read methods: %v", err))
	}
	if err := addDevMCPCredentialWriteMethods(runner, credential); err != nil {
		panic(fmt.Sprintf("dev mcp credential write methods: %v", err))
	}
	if err := addDevMCPMemberListMethod(runner, member); err != nil {
		panic(fmt.Sprintf("dev mcp member list method: %v", err))
	}
	if err := addDevMCPMemberMutationMethods(runner, member); err != nil {
		panic(fmt.Sprintf("dev mcp member mutation methods: %v", err))
	}
	if err := addDevMCPHsfMethodListMethod(runner, hsf); err != nil {
		panic(fmt.Sprintf("dev mcp hsf method list method: %v", err))
	}
	return root
}

func newDevMCPCommandGroup(name, short string) *cobra.Command {
	cmd := &cobra.Command{
		Use:               name,
		Short:             short,
		Args:              cobra.NoArgs,
		TraverseChildren:  true,
		DisableAutoGenTag: true,
	}
	newGroupCommand(cmd)
	return cmd
}

func devMCPValidAuthType(authType string) bool {
	switch authType {
	case "NO_AUTH", "BASIC", "API_SECRET", "TOKEN", "SIGNATURE":
		return true
	default:
		return false
	}
}

func devMCPServerNameFlag(cmd *cobra.Command) (string, error) {
	serverName := strings.ToLower(devMCPStringFlag(cmd, "server-name"))
	if serverName == "" {
		return "", nil
	}
	if len(serverName) > 255 {
		return "", apperrors.NewValidation("--server-name 必须不超过 255 个字符")
	}
	for i, r := range serverName {
		isAlphaNum := r >= 'a' && r <= 'z' || r >= '0' && r <= '9'
		validDash := r == '-' && i > 0 && i < len(serverName)-1 && serverName[i-1] != '-'
		if !isAlphaNum && !validDash {
			return "", apperrors.NewValidation("--server-name 必须是 kebab-case，仅含字母、数字和单个中划线")
		}
	}
	return serverName, nil
}

func addDevMCPCredentialLocatorFlags(cmd *cobra.Command) {
	addDevMCPMCPIDFlag(cmd)
	cmd.Flags().Int("credential-id", 0, "凭证账号 ID")
}

func devMCPCredentialLocatorParams(cmd *cobra.Command) (map[string]any, error) {
	mcpID, err := devMCPRequiredInt(cmd, "mcp-id")
	if err != nil {
		return nil, err
	}
	credentialID, err := devMCPRequiredInt(cmd, "credential-id")
	if err != nil {
		return nil, err
	}
	return map[string]any{"mcpId": mcpID, "credentialId": credentialID}, nil
}

func devMCPCredentialContent(cmd *cobra.Command) (map[string]any, error) {
	raw := devMCPStringFlag(cmd, "content")
	path := devMCPStringFlag(cmd, "content-file")
	if raw != "" && path != "" {
		return nil, apperrors.NewValidation("--content 与 --content-file 只能使用一个")
	}
	if path != "" {
		var data []byte
		var err error
		if path == "-" {
			data, err = io.ReadAll(cmd.InOrStdin())
		} else {
			data, err = os.ReadFile(path)
		}
		if err != nil {
			return nil, apperrors.NewValidation(fmt.Sprintf("读取 --content-file 失败: %v", err))
		}
		raw = strings.TrimSpace(string(data))
	}
	if raw == "" {
		return nil, apperrors.NewValidation("--content 或 --content-file 为必填")
	}
	var content map[string]any
	if err := json.Unmarshal([]byte(raw), &content); err != nil || content == nil {
		return nil, apperrors.NewValidation("凭证 content 必须是 JSON 对象")
	}
	return content, nil
}

func addDevMCPMCPIDFlag(cmd *cobra.Command) {
	cmd.Flags().Int("mcp-id", 0, "MCP 服务 ID")
}

func addDevMCPToolLocatorFlags(cmd *cobra.Command) {
	addDevMCPMCPIDFlag(cmd)
	addDevMCPToolIDFlag(cmd)
}

func addDevMCPToolIDFlag(cmd *cobra.Command) {
	cmd.Flags().String("tool-id", "", "MCP 工具 ID，G-ACT- 开头")
	cmd.Flags().String("action-id", "", "已更名为 --tool-id")
	_ = cmd.Flags().MarkHidden("action-id")
}

// devMCPRequiredToolID reads --tool-id and rejects the pre-0714 --action-id
// spelling with a rename hint instead of silently ignoring it.
func devMCPRequiredToolID(cmd *cobra.Command) (string, error) {
	if cmd.Flags().Changed("action-id") {
		return "", apperrors.NewValidation("--action-id 已更名为 --tool-id（脚手架 0714 契约变更），请改用 --tool-id")
	}
	return devMCPRequiredString(cmd, "tool-id")
}

func addDevMCPPagingFlags(cmd *cobra.Command) {
	cmd.Flags().Int("cursor", 0, "分页游标，从 1 开始")
	cmd.Flags().Int("page-size", 0, "每页条数，最大 100")
}

func addDevMCPToolUpsertFlags(cmd *cobra.Command, includeToolID bool) {
	addDevMCPMCPIDFlag(cmd)
	if includeToolID {
		addDevMCPToolIDFlag(cmd)
	}
	cmd.Flags().String("name", "", "工具唯一标识，snake_case")
	cmd.Flags().String("title", "", "必填。工具中文标题：中文自然语言、≤30 字、与功能一致")
	cmd.Flags().String("description", "", "必填。工具功能完整描述（LLM 选择工具的核心依据）：动词开头，说明功能/何时用/入参来源/破坏性行为")
	cmd.Flags().String("http-info", "", "必填。HTTP 接口配置 JSON 对象：{method,url,auth:{type,...}}")
	cmd.Flags().String("http", "", "已更名为 --http-info")
	_ = cmd.Flags().MarkHidden("http")
	cmd.Flags().String("api-inputs", "", "必填。接口真实入参 JSON 对象：{headers,body,query,path} 四组，每组=字段数组，字段项={key,title,type,required,description,children}；⚠️平台不支持 enum/default/example 属性——枚举/默认值/示例写进字段 description 文本")
	cmd.Flags().String("api-outputs", "", "必填。接口真实出参 JSON 对象：{headers,body} 两组，字段结构同 --api-inputs；⚠️出参按此 schema 精确裁剪——声明什么字段就返回什么，未声明的被过滤（漏声明=业务数据被整段吞掉）；结构未知时先按材料尽力声明→debug 取样→update 修正")
	cmd.Flags().String("tool-inputs", "", "必填。暴露给 LLM 的入参字段树 JSON 数组（array 型 children 固定一项 key=items）；每字段必须写自包含 description：含义+取值格式+示例，可对 api-inputs 裁剪/改名/加防呆")
	cmd.Flags().String("tool-outputs", "", "必填。暴露给 LLM 的出参字段树 JSON 数组；不做出参精修时显式传 '[]'（配整体透传=返回按 --api-outputs 裁剪后的完整响应体）；精修时与 --output-mappings 配套（裁字段/改名/补语义）")
	cmd.Flags().String("input-mappings", "", "必填。入参映射 JSON 数组；reference/fixed 使用 {target,type,source}，express 使用 {target,type,expression,displayText}；target 位置名必须 Pascal（$.Body./$.Query./$.Head./$.Path.）")
	cmd.Flags().String("output-mappings", "", "必填。出参映射 JSON 数组：整体透传 [{\"target\":\"$\",\"type\":\"reference\",\"source\":\"$.node_service_activator.Body\"}] 或字段级精修（配合 --tool-outputs 裁剪/改名，详见 skill mapping-rules）；⚠️必须与 --api-outputs 同批提交并静态互验（source 引用的字段必须已声明，红线#13）；⚠️省略或传 []＝草稿仍建成但运行时多包一层 Body 且 publish 被拦，需先 update 补齐")
	addDevMCPTimeoutOOKFlags(cmd, includeToolID)
}

func addDevMCPTimeoutOOKFlags(cmd *cobra.Command, update bool) {
	timeoutNote := "可选。请求超时秒数（1-180 整数），不传=系统默认 10 秒；⚠️一旦设置无法清回系统默认（传 0 会被 1-180 校验拒）"
	ookNote := "可选。仅映射非空字段：开启后出参只含下游实际返回的 key（无 null/空占位）；不传=默认关闭"
	if update {
		timeoutNote += "。全量提交语义的定点例外：漏传=保留原值"
		ookNote = "可选。仅映射非空字段开关；漏传=保留原值（定点例外），要关闭必须显式传 false"
	}
	cmd.Flags().Int("timeout", 0, timeoutNote)
	cmd.Flags().Bool("only-original-keys", false, ookNote)
}

func devMCPPutTimeoutOOK(cmd *cobra.Command, params map[string]any) error {
	if cmd.Flags().Changed("timeout") {
		t := devMCPIntFlag(cmd, "timeout")
		if t < 1 || t > 180 {
			return apperrors.NewValidation("--timeout 取值范围 1-180 秒（不支持传 0 清回系统默认）")
		}
		params["timeout"] = t
	}
	if cmd.Flags().Changed("only-original-keys") {
		v, _ := cmd.Flags().GetBool("only-original-keys")
		params["onlyOriginalKeys"] = v
	}
	return nil
}

func annotateDevMCPTool(cmd *cobra.Command, tool string) *cobra.Command {
	if cmd.Annotations == nil {
		cmd.Annotations = map[string]string{}
	}
	cmd.Annotations["mcp-tool"] = tool
	cmd.Annotations["mcp-source"] = devMCPProduct
	return cmd
}

func runDevMCPTool(runner executor.Runner, cmd *cobra.Command, tool string, params map[string]any) error {
	invocation := executor.NewHelperInvocation(
		cobracmd.LegacyCommandPath(cmd),
		devMCPProduct,
		tool,
		params,
	)
	invocation.DryRun = commandDryRun(cmd)
	result, err := runner.Run(cmd.Context(), invocation)
	if err != nil {
		return err
	}
	return writeDevAppEnvelope(cmd, result)
}

func devMCPToolLocatorParams(cmd *cobra.Command) (map[string]any, error) {
	mcpID, err := devMCPRequiredInt(cmd, "mcp-id")
	if err != nil {
		return nil, err
	}
	toolID, err := devMCPRequiredToolID(cmd)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"mcpId":  mcpID,
		"toolId": toolID,
	}, nil
}

func devMCPToolUpsertParams(cmd *cobra.Command, includeToolID bool) (map[string]any, error) {
	mcpID, err := devMCPRequiredInt(cmd, "mcp-id")
	if err != nil {
		return nil, err
	}
	name, err := devMCPRequiredString(cmd, "name")
	if err != nil {
		return nil, err
	}
	params := map[string]any{
		"mcpId": mcpID,
		"name":  name,
	}
	if includeToolID {
		toolID, err := devMCPRequiredToolID(cmd)
		if err != nil {
			return nil, err
		}
		params["toolId"] = toolID
	}

	if cmd.Flags().Changed("http") {
		return nil, apperrors.NewValidation("--http 已更名为 --http-info（脚手架契约变更：请求体键名 http→httpInfo），请改用 --http-info")
	}
	httpInfo, err := devMCPRequiredJSONObjectFlag(cmd, "http-info")
	if err != nil {
		return nil, err
	}
	params["httpInfo"] = httpInfo

	devMCPPutString(params, "title", devMCPStringFlag(cmd, "title"))
	devMCPPutString(params, "description", devMCPStringFlag(cmd, "description"))

	if err := devMCPPutJSONObjectFlag(cmd, params, "api-inputs", "apiInputs"); err != nil {
		return nil, err
	}
	if err := devMCPPutJSONObjectFlag(cmd, params, "api-outputs", "apiOutputs"); err != nil {
		return nil, err
	}
	if err := devMCPPutJSONArrayFlag(cmd, params, "tool-inputs", "toolInputs"); err != nil {
		return nil, err
	}
	if err := devMCPPutJSONArrayFlag(cmd, params, "tool-outputs", "toolOutputs"); err != nil {
		return nil, err
	}
	if err := devMCPPutJSONArrayFlag(cmd, params, "input-mappings", "inputMappings"); err != nil {
		return nil, err
	}
	if err := devMCPPutJSONArrayFlag(cmd, params, "output-mappings", "outputMappings"); err != nil {
		return nil, err
	}
	if err := devMCPValidateToolUpsertParams(params); err != nil {
		return nil, err
	}
	var missingOutputs []string
	for _, f := range []struct{ key, flag string }{
		{"apiOutputs", "--api-outputs"},
		{"toolOutputs", "--tool-outputs"},
		{"outputMappings", "--output-mappings"},
	} {
		if _, ok := params[f.key]; !ok {
			missingOutputs = append(missingOutputs, f.flag)
		}
	}
	if err := devMCPPutTimeoutOOK(cmd, params); err != nil {
		return nil, err
	}
	if len(missingOutputs) > 0 {
		return nil, apperrors.NewValidation(strings.Join(missingOutputs, "、") + " 为必填（0720 契约：出参三件套）。漏传 apiOutputs 的故障形态：工具建成且调用看似成功，但下游复杂结构字段被精确裁剪整段吞掉、UI 出参映射标「变量已失效」。不做出参精修时：--tool-outputs 传 '[]'、--output-mappings 用整体透传 [{\"target\":\"$\",\"type\":\"reference\",\"source\":\"$.node_service_activator.Body\"}]、--api-outputs 按接口真实出参如实声明（结构未知先按材料尽力声明，debug 取样后 update 修正）")
	}
	return params, nil
}

func devMCPValidateToolUpsertParams(params map[string]any) error {
	if err := devMCPValidateToolName(stringValue(params["name"])); err != nil {
		return err
	}
	if fields, ok := params["toolInputs"].([]any); ok {
		if err := devMCPValidateFieldsFlag("tool-inputs", fields, true); err != nil {
			return err
		}
	}
	if fields, ok := params["toolOutputs"].([]any); ok {
		if err := devMCPValidateFieldsFlag("tool-outputs", fields, true); err != nil {
			return err
		}
	}
	if inputs, ok := params["apiInputs"].(map[string]any); ok {
		if err := devMCPValidateAPIFieldsFlag("api-inputs", inputs); err != nil {
			return err
		}
	}
	if outputs, ok := params["apiOutputs"].(map[string]any); ok {
		if err := devMCPValidateAPIFieldsFlag("api-outputs", outputs); err != nil {
			return err
		}
	}
	if mappings, ok := params["inputMappings"].([]any); ok {
		if err := devMCPValidateMappingsFlag("input-mappings", mappings); err != nil {
			return err
		}
	}
	if mappings, ok := params["outputMappings"].([]any); ok {
		if err := devMCPValidateMappingsFlag("output-mappings", mappings); err != nil {
			return err
		}
	}
	if err := devMCPLintMappingReferences(params); err != nil {
		return err
	}
	return nil
}

func devMCPValidateToolName(name string) error {
	if name == "" {
		return nil
	}
	if len([]rune(name)) > 32 {
		return apperrors.NewValidation("--name 必须不超过 32 个字符，便于生成稳定 DWS 命令")
	}
	for i, r := range name {
		valid := r == '_' || r >= 'a' && r <= 'z' || r >= '0' && r <= '9'
		if !valid || i == 0 && !(r >= 'a' && r <= 'z') {
			return apperrors.NewValidation("--name 必须是 snake_case，且以小写英文动词开头，例如 lookup_english_word")
		}
	}
	if !devMCPContainsKnownVerb(name) {
		return apperrors.NewValidation("--name 必须包含清晰动作词，例如 get/list/search/query/lookup/create/update/delete，便于 Agent 选择工具")
	}
	return nil
}

func devMCPContainsKnownVerb(name string) bool {
	for _, token := range strings.Split(name, "_") {
		if devMCPKnownVerb(token) {
			return true
		}
	}
	return false
}

func devMCPKnownVerb(verb string) bool {
	switch verb {
	case "get", "list", "search", "query", "lookup", "find", "fetch", "read", "check", "validate",
		"create", "add", "new", "update", "set", "edit", "delete", "remove",
		"send", "submit", "publish", "debug", "run", "call", "sync", "refresh",
		"convert", "generate", "calculate", "parse", "build", "upload", "download", "export", "import":
		return true
	default:
		return false
	}
}

func devMCPValidateAPIFieldsFlag(flag string, groups map[string]any) error {
	for _, group := range []string{"headers", "head", "body", "query", "path"} {
		raw, ok := groups[group]
		if !ok {
			continue
		}
		fields, ok := raw.([]any)
		if !ok {
			return apperrors.NewValidation(fmt.Sprintf("--%s.%s 必须是字段数组", flag, group))
		}
		if err := devMCPValidateFieldsFlag(flag+"."+group, fields, false); err != nil {
			return err
		}
	}
	return nil
}

func devMCPValidateFieldsFlag(flag string, fields []any, requireDescription bool) error {
	for i, raw := range fields {
		field, ok := raw.(map[string]any)
		if !ok {
			return apperrors.NewValidation(fmt.Sprintf("--%s[%d] 必须是 JSON 对象", flag, i))
		}
		path := fmt.Sprintf("--%s[%d]", flag, i)
		if err := devMCPValidateField(path, field, requireDescription); err != nil {
			return err
		}
	}
	return nil
}

func devMCPValidateField(path string, field map[string]any, requireDescription bool) error {
	key := stringValue(field["key"])
	if key == "" {
		return apperrors.NewValidation(path + ".key 为必填，DWS flag 需要稳定字段名")
	}
	typ := stringValue(field["type"])
	if typ == "" {
		return apperrors.NewValidation(path + ".type 为必填，DWS 需要据此生成参数类型")
	}
	if !devMCPValidFieldType(typ) {
		return apperrors.NewValidation(path + ".type 只支持 string/number/integer/boolean/object/array")
	}
	if requireDescription && stringValue(field["description"]) == "" {
		return apperrors.NewValidation(path + ".description 为必填，DWS 命令和 Agent 选参依赖字段描述")
	}
	children, hasChildren := field["children"].([]any)
	if typ == "array" {
		if !hasChildren || len(children) != 1 {
			return apperrors.NewValidation(path + ".children 必须且只能包含一个 key=items 的数组元素描述")
		}
		child, ok := children[0].(map[string]any)
		if !ok || stringValue(child["key"]) != "items" {
			return apperrors.NewValidation(path + ".children[0].key 必须是 items")
		}
		return devMCPValidateField(path+".children[0]", child, requireDescription)
	}
	if typ == "object" && hasChildren {
		for i, raw := range children {
			child, ok := raw.(map[string]any)
			if !ok {
				return apperrors.NewValidation(fmt.Sprintf("%s.children[%d] 必须是 JSON 对象", path, i))
			}
			if err := devMCPValidateField(fmt.Sprintf("%s.children[%d]", path, i), child, requireDescription); err != nil {
				return err
			}
		}
	}
	return nil
}

func devMCPValidFieldType(typ string) bool {
	switch typ {
	case "string", "number", "integer", "boolean", "object", "array":
		return true
	default:
		return false
	}
}

func stringValue(value any) string {
	switch typed := value.(type) {
	case nil:
		return ""
	case string:
		return strings.TrimSpace(typed)
	case fmt.Stringer:
		return strings.TrimSpace(typed.String())
	default:
		return strings.TrimSpace(fmt.Sprint(value))
	}
}

func devMCPValidateMappingsFlag(flag string, mappings []any) error {
	for i, raw := range mappings {
		mapping, ok := raw.(map[string]any)
		if !ok {
			return apperrors.NewValidation(fmt.Sprintf("--%s[%d] 必须是 JSON 对象", flag, i))
		}
		path := fmt.Sprintf("--%s[%d]", flag, i)
		if stringValue(mapping["target"]) == "" {
			return apperrors.NewValidation(path + ".target 为必填")
		}
		typ := stringValue(mapping["type"])
		if typ == "" {
			return apperrors.NewValidation(path + ".type 为必填")
		}
		if typ != "reference" && typ != "fixed" && typ != "express" {
			return apperrors.NewValidation(path + ".type 只支持 reference/fixed/express")
		}
		switch typ {
		case "reference", "fixed":
			if stringValue(mapping["source"]) == "" {
				return apperrors.NewValidation(path + ".source 为必填")
			}
			if stringValue(mapping["expression"]) != "" {
				return apperrors.NewValidation(path + ".expression 不适用于 reference/fixed；取值只能放 source 字段")
			}
		case "express":
			expression, valid := mapping["expression"].(string)
			if !valid || strings.TrimSpace(expression) == "" {
				return apperrors.NewValidation(path + ".expression 为必填；express 表达式必须放 expression 字段，不能放 source")
			}
			if stringValue(mapping["source"]) != "" {
				return apperrors.NewValidation(path + ".source 不适用于 express；表达式只能放 expression 字段")
			}
		}
	}
	return nil
}

func devMCPRequiredString(cmd *cobra.Command, flag string) (string, error) {
	value := devMCPStringFlag(cmd, flag)
	if value == "" {
		return "", apperrors.NewValidation(fmt.Sprintf("--%s 为必填", flag))
	}
	return value, nil
}

func devMCPRequiredInt(cmd *cobra.Command, flag string) (int, error) {
	value := devMCPIntFlag(cmd, flag)
	if value <= 0 {
		return 0, apperrors.NewValidation(fmt.Sprintf("--%s 为必填", flag))
	}
	return value, nil
}

func devMCPStringFlag(cmd *cobra.Command, name string) string {
	value, _ := cmd.Flags().GetString(name)
	return strings.TrimSpace(value)
}

func devMCPIntFlag(cmd *cobra.Command, name string) int {
	value, _ := cmd.Flags().GetInt(name)
	return value
}

func devMCPPutString(params map[string]any, key, value string) int {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0
	}
	params[key] = value
	return 1
}

func devMCPPutInt(params map[string]any, key string, value int) {
	if value != 0 {
		params[key] = value
	}
}

func devMCPRequiredJSONObjectFlag(cmd *cobra.Command, flag string) (map[string]any, error) {
	value := devMCPStringFlag(cmd, flag)
	if value == "" {
		return nil, apperrors.NewValidation(fmt.Sprintf("--%s 为必填", flag))
	}
	var out map[string]any
	if err := json.Unmarshal([]byte(value), &out); err != nil || out == nil {
		return nil, apperrors.NewValidation(fmt.Sprintf("--%s 必须是 JSON 对象", flag))
	}
	return out, nil
}

func devMCPPutJSONObjectFlag(cmd *cobra.Command, params map[string]any, flag, key string) error {
	value := devMCPStringFlag(cmd, flag)
	if value == "" {
		return nil
	}
	var out map[string]any
	if err := json.Unmarshal([]byte(value), &out); err != nil || out == nil {
		return apperrors.NewValidation(fmt.Sprintf("--%s 必须是 JSON 对象", flag))
	}
	params[key] = out
	return nil
}

func devMCPPutJSONArrayFlag(cmd *cobra.Command, params map[string]any, flag, key string) error {
	value := devMCPStringFlag(cmd, flag)
	if value == "" {
		return nil
	}
	var out []any
	if err := json.Unmarshal([]byte(value), &out); err != nil || out == nil {
		return apperrors.NewValidation(fmt.Sprintf("--%s 必须是 JSON 数组", flag))
	}
	params[key] = out
	return nil
}
