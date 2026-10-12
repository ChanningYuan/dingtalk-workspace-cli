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
	"bytes"
	"context"
	"encoding/json"
	"fmt"

	"github.com/DingTalk-Real-AI/dingtalk-workspace-cli/internal/corecmd"
	"github.com/DingTalk-Real-AI/dingtalk-workspace-cli/internal/corecmd/contract"
	"github.com/DingTalk-Real-AI/dingtalk-workspace-cli/internal/executor"
	"github.com/DingTalk-Real-AI/dingtalk-workspace-cli/internal/output"
)

type DevMCPServiceGetRequest struct {
	MCPID int
}

type DevMCPServiceGetCategory struct {
	Code        string `json:"code,omitempty"`
	DisplayName string `json:"displayName,omitempty"`
}

type DevMCPServiceGetTool struct {
	Description  string `json:"description,omitempty"`
	DisplayName  string `json:"displayName,omitempty"`
	InputSchema  string `json:"inputSchema,omitempty"`
	Name         string `json:"name,omitempty"`
	OutputSchema string `json:"outputSchema,omitempty"`
}

type DevMCPServiceGetResponse struct {
	MCPID        string                     `json:"mcpId,omitempty"`
	Name         string                     `json:"name,omitempty"`
	Description  string                     `json:"description,omitempty"`
	IconURL      string                     `json:"iconUrl,omitempty"`
	Introduction string                     `json:"introduction,omitempty"`
	ServerName   string                     `json:"serverName,omitempty"`
	Categories   []DevMCPServiceGetCategory `json:"categories,omitempty"`
	Tools        []DevMCPServiceGetTool     `json:"tools,omitempty"`

	preview json.RawMessage
}

func devMCPServiceGetOperation(runner executor.Runner) corecmd.Operation[DevMCPServiceGetRequest, DevMCPServiceGetResponse] {
	return corecmd.Operation[DevMCPServiceGetRequest, DevMCPServiceGetResponse]{
		MethodDefinition: corecmd.MethodDefinition{
			Identity: corecmd.MethodIdentity{
				CLIPath:     "dev mcp service get",
				ProductID:   "dev",
				Name:        devMCPServiceGetTool,
				Short:       "查询 MCP 服务详情",
				Description: "查询指定 MCP 服务的开发配置",
				Example:     "  dws dev mcp service get --mcp-id 10487 --format json",
			},
			Input: corecmd.InputSpec{Parameters: []corecmd.ParameterSpec{
				{Name: "mcp-id", Type: corecmd.KindInt, Usage: "MCP 服务 ID", Bind: "mcpId", Required: true, RequiredHint: "--mcp-id 为必填"},
			}},
			Contract: corecmd.ContractSpec{
				Selection: contract.SelectionSpec{
					AgentSummary: "查询指定 MCP 服务的开发配置",
					UseWhen:      []string{"查询指定 MCP 服务的开发配置"},
					AvoidWhen:    []string{"只需调用已发布 MCP 工具时使用 dws mcp published，而不是开发配置命令"},
					Examples:     []string{"dws dev mcp service get"},
				},
				Interface: &contract.InterfaceSpec{
					Mode:         contract.InterfaceModeComposite,
					Availability: contract.InterfaceAvailable,
					Reason:       "Reviewed unpinned adapter: this static developer command calls the helper-only mcpdev/mcp_service_get management endpoint, which is intentionally absent from pinned public MCP metadata.",
				},
				DryRun: &contract.DryRunSpec{
					PreviewKind: contract.DryRunPreviewInvocation,
					RemoteReads: false,
				},
				Result: &contract.ResultSpec{
					Outcomes: []contract.ResultOutcome{contract.ResultOutcomeSuccess, contract.ResultOutcomeFailure},
					DataSchema: json.RawMessage(`{
						"type":"object",
						"description":"MCP 服务详情",
						"required":["mcpId"],
						"properties":{
							"mcpId":{"type":"string","description":"MCP 服务 ID"},
							"name":{"type":"string","description":"服务名称"},
							"description":{"type":"string","description":"服务用途描述"},
							"iconUrl":{"type":"string","description":"服务图标 URL"},
							"introduction":{"type":"string","description":"服务详情介绍"},
							"serverName":{"type":"string","description":"服务稳定语义标识"},
							"categories":{"type":"array","description":"服务分类","items":{"type":"object","description":"服务分类","properties":{"code":{"type":"string","description":"分类编码"},"displayName":{"type":"string","description":"分类展示名称"}},"additionalProperties":true}},
							"tools":{"type":"array","description":"服务包含的工具","items":{"type":"object","description":"MCP 工具","properties":{"description":{"type":"string","description":"工具描述"},"displayName":{"type":"string","description":"工具展示名称"},"inputSchema":{"type":"string","description":"工具输入 Schema"},"name":{"type":"string","description":"工具名称"},"outputSchema":{"type":"string","description":"工具输出 Schema"}},"additionalProperties":true}}
						},
						"additionalProperties":true
					}`),
				},
			},
			Safety:        devMCPReadSafety(),
			OutputRollout: output.RolloutUnifiedActive,
			PostMount:     devMCPServicePostMount(devMCPServiceGetTool),
		},
		Decode: func(c *corecmd.Ctx) (DevMCPServiceGetRequest, error) {
			return DevMCPServiceGetRequest{MCPID: c.Int("mcp-id")}, nil
		},
		Execute: func(ctx context.Context, request DevMCPServiceGetRequest, options corecmd.OperationOptions) (DevMCPServiceGetResponse, error) {
			return executeDevMCPServiceGet(ctx, runner, request, options)
		},
		Project: func(response DevMCPServiceGetResponse, options corecmd.OperationOptions) output.CommandResult {
			if options.DryRun && len(response.preview) > 0 {
				return output.Success(json.RawMessage(response.preview), output.WithDryRun())
			}
			return output.Success(response)
		},
	}
}

func executeDevMCPServiceGet(ctx context.Context, runner executor.Runner, request DevMCPServiceGetRequest, options corecmd.OperationOptions) (DevMCPServiceGetResponse, error) {
	if options.CLIPath == "" {
		return DevMCPServiceGetResponse{}, fmt.Errorf("%s response: missing operation CLI path", devMCPServiceGetTool)
	}
	invocation := executor.NewHelperInvocation(options.CLIPath, devMCPProduct, devMCPServiceGetTool, map[string]any{"mcpId": request.MCPID})
	invocation.DryRun = options.DryRun
	result, err := runner.Run(ctx, invocation)
	if err != nil {
		return DevMCPServiceGetResponse{}, err
	}
	result.Invocation.Tool = devMCPServiceGetTool
	result = normalizeDevAppServiceResult(result)
	data := devAppEnvelopeData(result)
	raw, err := json.Marshal(data)
	if err != nil {
		return DevMCPServiceGetResponse{}, fmt.Errorf("decode %s response: %w", devMCPServiceGetTool, err)
	}
	if options.DryRun {
		var preview map[string]json.RawMessage
		if err := json.Unmarshal(raw, &preview); err != nil || preview == nil {
			return DevMCPServiceGetResponse{}, fmt.Errorf("decode %s dry-run response: expected an object", devMCPServiceGetTool)
		}
		return DevMCPServiceGetResponse{preview: raw}, nil
	}
	if !result.Invocation.Implemented {
		return DevMCPServiceGetResponse{}, fmt.Errorf("decode %s response: backend did not return an implemented response", devMCPServiceGetTool)
	}
	content, ok := result.Response["content"].(map[string]any)
	if !ok || content == nil {
		return DevMCPServiceGetResponse{}, fmt.Errorf("decode %s response: missing object content", devMCPServiceGetTool)
	}
	rawContent, err := json.Marshal(content)
	if err != nil {
		return DevMCPServiceGetResponse{}, fmt.Errorf("decode %s response: %w", devMCPServiceGetTool, err)
	}
	if err := rejectDevMCPServiceGetNulls(rawContent); err != nil {
		return DevMCPServiceGetResponse{}, fmt.Errorf("decode %s response: %w", devMCPServiceGetTool, err)
	}
	var response DevMCPServiceGetResponse
	if err := json.Unmarshal(rawContent, &response); err != nil {
		return DevMCPServiceGetResponse{}, fmt.Errorf("decode %s response: %w", devMCPServiceGetTool, err)
	}
	if response.MCPID == "" {
		return DevMCPServiceGetResponse{}, fmt.Errorf("decode %s response: missing mcpId", devMCPServiceGetTool)
	}
	return response, nil
}

func rejectDevMCPServiceGetNulls(raw json.RawMessage) error {
	var object map[string]json.RawMessage
	if err := json.Unmarshal(raw, &object); err != nil || object == nil {
		return fmt.Errorf("expected object content")
	}
	if err := rejectDevMCPServiceGetNullFields(object, "", []string{
		"mcpId", "name", "description", "iconUrl", "introduction", "serverName",
	}); err != nil {
		return err
	}
	for _, field := range []struct {
		name   string
		fields []string
	}{
		{name: "categories", fields: []string{"code", "displayName"}},
		{name: "tools", fields: []string{"description", "displayName", "inputSchema", "name", "outputSchema"}},
	} {
		rawItems, exists := object[field.name]
		if !exists {
			continue
		}
		if isJSONNull(rawItems) {
			return fmt.Errorf("field %s must not be null", field.name)
		}
		var items []json.RawMessage
		if err := json.Unmarshal(rawItems, &items); err != nil {
			continue
		}
		for index, rawItem := range items {
			if isJSONNull(rawItem) {
				return fmt.Errorf("field %s[%d] must not be null", field.name, index)
			}
			var item map[string]json.RawMessage
			if err := json.Unmarshal(rawItem, &item); err != nil || item == nil {
				continue
			}
			if err := rejectDevMCPServiceGetNullFields(item, fmt.Sprintf("%s[%d]", field.name, index), field.fields); err != nil {
				return err
			}
		}
	}
	return nil
}

func rejectDevMCPServiceGetNullFields(object map[string]json.RawMessage, prefix string, fields []string) error {
	for _, field := range fields {
		raw, exists := object[field]
		if !exists || !isJSONNull(raw) {
			continue
		}
		if prefix == "" {
			return fmt.Errorf("field %s must not be null", field)
		}
		return fmt.Errorf("field %s.%s must not be null", prefix, field)
	}
	return nil
}

func isJSONNull(raw json.RawMessage) bool {
	return bytes.Equal(bytes.TrimSpace(raw), []byte("null"))
}
