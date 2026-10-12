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
	apperrors "github.com/DingTalk-Real-AI/dingtalk-workspace-cli/internal/errors"
	"github.com/spf13/cobra"
)

// V5 契约（2026-07-21）：HSF 型工具参数解析保持与既有接口一致。
// hsf 与 http 的关键差异：apiInputs/apiOutputs 无需提供（服务端按 HSF 方法
// schema 自动生成）；映射 target 为 $.<DTO简名>.<字段>（非 Pascal 位置名）、
// 出参 source 无 .Body. 前缀——因此不走 http 侧的映射静态互验（lint）。

func devMCPToolHsfCreateParams(cmd *cobra.Command) (map[string]any, error) {
	mcpID, err := devMCPRequiredInt(cmd, "mcp-id")
	if err != nil {
		return nil, err
	}
	name, err := devMCPRequiredString(cmd, "name")
	if err != nil {
		return nil, err
	}
	params := map[string]any{"mcpId": mcpID, "name": name}
	if err := devMCPValidateToolName(name); err != nil {
		return nil, err
	}
	hsfInfo, err := devMCPRequiredJSONObjectFlag(cmd, "hsf-info")
	if err != nil {
		return nil, err
	}
	params["hsfInfo"] = hsfInfo
	devMCPPutString(params, "title", devMCPStringFlag(cmd, "title"))
	devMCPPutString(params, "description", devMCPStringFlag(cmd, "description"))
	for _, f := range []struct{ flag, key string }{
		{"tool-inputs", "toolInputs"},
		{"input-mappings", "inputMappings"},
		{"tool-outputs", "toolOutputs"},
		{"output-mappings", "outputMappings"},
	} {
		if err := devMCPPutJSONArrayFlag(cmd, params, f.flag, f.key); err != nil {
			return nil, err
		}
		if _, ok := params[f.key]; !ok {
			return nil, apperrors.NewValidation("--" + f.flag + " 为必填（hsf 建造四件套；--tool-outputs 不精修显式传 '[]'；apiInputs/apiOutputs 无需提供，服务端按 HSF 方法 schema 自动生成）")
		}
	}
	if fields, ok := params["toolInputs"].([]any); ok {
		if err := devMCPValidateFieldsFlag("tool-inputs", fields, true); err != nil {
			return nil, err
		}
	}
	if fields, ok := params["toolOutputs"].([]any); ok {
		if err := devMCPValidateFieldsFlag("tool-outputs", fields, true); err != nil {
			return nil, err
		}
	}
	if mappings, ok := params["inputMappings"].([]any); ok {
		if err := devMCPValidateMappingsFlag("input-mappings", mappings); err != nil {
			return nil, err
		}
	}
	if mappings, ok := params["outputMappings"].([]any); ok {
		if err := devMCPValidateMappingsFlag("output-mappings", mappings); err != nil {
			return nil, err
		}
	}
	if err := devMCPPutTimeoutOOK(cmd, params); err != nil {
		return nil, err
	}
	return params, nil
}

func devMCPToolHsfUpdateParams(cmd *cobra.Command) (map[string]any, error) {
	params, err := devMCPToolLocatorParams(cmd)
	if err != nil {
		return nil, err
	}
	if name := devMCPStringFlag(cmd, "name"); name != "" {
		if err := devMCPValidateToolName(name); err != nil {
			return nil, err
		}
		params["name"] = name
	}
	devMCPPutString(params, "title", devMCPStringFlag(cmd, "title"))
	devMCPPutString(params, "description", devMCPStringFlag(cmd, "description"))
	if err := devMCPPutJSONObjectFlag(cmd, params, "hsf-info", "hsfInfo"); err != nil {
		return nil, err
	}
	for _, f := range []struct{ flag, key string }{
		{"tool-inputs", "toolInputs"},
		{"input-mappings", "inputMappings"},
		{"tool-outputs", "toolOutputs"},
		{"output-mappings", "outputMappings"},
	} {
		if err := devMCPPutJSONArrayFlag(cmd, params, f.flag, f.key); err != nil {
			return nil, err
		}
	}
	if fields, ok := params["toolInputs"].([]any); ok {
		if err := devMCPValidateFieldsFlag("tool-inputs", fields, true); err != nil {
			return nil, err
		}
	}
	if fields, ok := params["toolOutputs"].([]any); ok {
		if err := devMCPValidateFieldsFlag("tool-outputs", fields, true); err != nil {
			return nil, err
		}
	}
	if mappings, ok := params["inputMappings"].([]any); ok {
		if err := devMCPValidateMappingsFlag("input-mappings", mappings); err != nil {
			return nil, err
		}
	}
	if mappings, ok := params["outputMappings"].([]any); ok {
		if err := devMCPValidateMappingsFlag("output-mappings", mappings); err != nil {
			return nil, err
		}
	}
	if err := devMCPPutTimeoutOOK(cmd, params); err != nil {
		return nil, err
	}
	if len(params) == 2 {
		return nil, apperrors.NewValidation("至少提供一项待更新字段（部分更新语义：只传要改的，未传保持原值）")
	}
	return params, nil
}
