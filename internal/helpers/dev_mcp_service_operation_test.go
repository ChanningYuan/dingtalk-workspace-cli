package helpers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/DingTalk-Real-AI/dingtalk-workspace-cli/internal/corecmd"
	"github.com/DingTalk-Real-AI/dingtalk-workspace-cli/internal/corecmd/contract"
	"github.com/DingTalk-Real-AI/dingtalk-workspace-cli/internal/corecmd/contractfinal"
	"github.com/DingTalk-Real-AI/dingtalk-workspace-cli/internal/executor"
)

func TestDevMCPServiceGetOperationCarriesResultContract(t *testing.T) {
	cmd := buildDevMCPServiceTestCommand(&captureRunner{}, "get")
	final, ok := contractfinal.RuntimeContractFinal(cmd)
	if !ok || final.Result == nil {
		t.Fatalf("result contract = %#v, registered = %t", final.Result, ok)
	}
	if len(final.Result.Outcomes) != 2 {
		t.Fatalf("outcomes = %#v", final.Result.Outcomes)
	}
	var schema map[string]any
	if err := json.Unmarshal(final.Result.DataSchema, &schema); err != nil {
		t.Fatalf("data schema: %v", err)
	}
	if schema["type"] != "object" || schema["description"] == "" {
		t.Fatalf("data schema = %#v", schema)
	}
	if schema["additionalProperties"] != true {
		t.Fatalf("root additionalProperties = %#v", schema["additionalProperties"])
	}
	required, ok := schema["required"].([]any)
	if !ok || len(required) != 1 || required[0] != "mcpId" {
		t.Fatalf("required = %#v", schema["required"])
	}
	if final.DryRun == nil || final.DryRun.PreviewKind != contract.DryRunPreviewInvocation || final.DryRun.RemoteReads {
		t.Fatalf("dry-run contract = %#v", final.DryRun)
	}
	properties, ok := schema["properties"].(map[string]any)
	if !ok || properties["mcpId"] == nil || properties["categories"] == nil || properties["tools"] == nil {
		t.Fatalf("properties = %#v", schema["properties"])
	}
	categories, ok := properties["categories"].(map[string]any)
	if !ok {
		t.Fatalf("categories schema = %#v", properties["categories"])
	}
	assertResultSchemaObjectProperties(t, categories["items"], map[string]string{
		"code":        "string",
		"displayName": "string",
	})
	tools, ok := properties["tools"].(map[string]any)
	if !ok {
		t.Fatalf("tools schema = %#v", properties["tools"])
	}
	assertResultSchemaObjectProperties(t, tools["items"], map[string]string{
		"description":  "string",
		"displayName":  "string",
		"inputSchema":  "string",
		"name":         "string",
		"outputSchema": "string",
	})
}

func assertResultSchemaObjectProperties(t *testing.T, raw any, expected map[string]string) {
	t.Helper()
	object, ok := raw.(map[string]any)
	if !ok || object["type"] != "object" || object["additionalProperties"] != true {
		t.Fatalf("object schema = %#v", raw)
	}
	properties, ok := object["properties"].(map[string]any)
	if !ok {
		t.Fatalf("object properties = %#v", object["properties"])
	}
	for name, wantType := range expected {
		property, ok := properties[name].(map[string]any)
		if !ok || property["type"] != wantType || property["description"] == "" {
			t.Fatalf("property %q = %#v", name, properties[name])
		}
	}
}

func TestDevMCPServiceGetTypedResponseDrivesOutput(t *testing.T) {
	runner := &devMCPServiceGetOperationRunner{result: executor.Result{
		Invocation: executor.Invocation{Implemented: true},
		Response: map[string]any{
			"content": map[string]any{
				"success": true,
				"result":  map[string]any{"mcpId": "10487", "name": "客户服务"},
			},
		},
	}}
	root := newDevAppTestRoot(runner)
	var outputBuffer bytes.Buffer
	root.SetOut(&outputBuffer)
	root.SetErr(&outputBuffer)
	root.SetArgs([]string{"dev", "mcp", "service", "get", "--mcp-id", "10487"})
	if err := corecmd.ExecuteForTest(root); err != nil {
		t.Fatal(err)
	}
	var envelope struct {
		Data map[string]any `json:"data"`
	}
	if err := json.Unmarshal(outputBuffer.Bytes(), &envelope); err != nil {
		t.Fatalf("output = %q: %v", outputBuffer.String(), err)
	}
	if envelope.Data["mcpId"] != "10487" || envelope.Data["name"] != "客户服务" {
		t.Fatalf("data = %#v", envelope.Data)
	}
}

func TestDevMCPServiceGetDryRunProjectsInvocationPreview(t *testing.T) {
	runner := &devMCPServiceGetOperationRunner{result: executor.Result{
		Invocation: executor.Invocation{Implemented: false},
		Response:   map[string]any{"dry_run": true, "request": map[string]any{"mcpId": 10487}},
	}}
	root := newDevAppTestRoot(runner)
	var outputBuffer bytes.Buffer
	root.SetOut(&outputBuffer)
	root.SetErr(&outputBuffer)
	root.SetArgs([]string{"dev", "mcp", "service", "get", "--mcp-id", "10487", "--dry-run"})
	if err := corecmd.ExecuteForTest(root); err != nil {
		t.Fatal(err)
	}
	var envelope struct {
		Data map[string]any `json:"data"`
	}
	if err := json.Unmarshal(outputBuffer.Bytes(), &envelope); err != nil {
		t.Fatalf("output = %q: %v", outputBuffer.String(), err)
	}
	invocation, ok := envelope.Data["invocation"].(map[string]any)
	if !ok || invocation["legacy_path"] != "dev mcp service get" || invocation["canonical_path"] != devMCPProduct+"."+devMCPServiceGetTool {
		t.Fatalf("dry-run invocation preview = %#v", envelope.Data)
	}
	if _, ok := envelope.Data["response"].(map[string]any); !ok {
		t.Fatalf("dry-run response preview = %#v", envelope.Data)
	}
}

func TestDevMCPServiceGetOperationDecodesTypedResponse(t *testing.T) {
	runner := &devMCPServiceGetOperationRunner{result: executor.Result{
		Invocation: executor.Invocation{Implemented: true},
		Response: map[string]any{
			"content": map[string]any{
				"success": true,
				"result": map[string]any{
					"mcpId":      "10487",
					"name":       "客户服务",
					"serverName": "customer-info",
					"categories": []any{map[string]any{"code": "sales", "displayName": "销售"}},
					"tools":      []any{map[string]any{"name": "search", "description": "搜索"}},
				},
			},
		},
	}}
	response, err := executeDevMCPServiceGet(context.Background(), runner, DevMCPServiceGetRequest{MCPID: 10487}, corecmd.OperationOptions{CLIPath: "dev mcp service get"})
	if err != nil {
		t.Fatal(err)
	}
	if len(runner.invocations) != 1 {
		t.Fatalf("invocations = %d, want 1", len(runner.invocations))
	}
	if !reflect.DeepEqual(runner.invocations[0].Params, map[string]any{"mcpId": 10487}) {
		t.Fatalf("params = %#v", runner.invocations[0].Params)
	}
	if runner.invocations[0].LegacyPath != "dev mcp service get" || runner.invocations[0].CanonicalPath != devMCPProduct+"."+devMCPServiceGetTool {
		t.Fatalf("invocation identity = %#v", runner.invocations[0])
	}
	if response.MCPID != "10487" || response.Name != "客户服务" || response.ServerName != "customer-info" {
		t.Fatalf("response = %#v", response)
	}
	if len(response.Categories) != 1 || response.Categories[0].Code != "sales" {
		t.Fatalf("categories = %#v", response.Categories)
	}
	if len(response.Tools) != 1 || response.Tools[0].Name != "search" {
		t.Fatalf("tools = %#v", response.Tools)
	}
}

func TestDevMCPServiceGetOperationPreservesDryRunAndRejectsMalformedResponse(t *testing.T) {
	t.Run("dry run", func(t *testing.T) {
		runner := &devMCPServiceGetOperationRunner{result: executor.Result{
			Invocation: executor.Invocation{Implemented: false},
			Response:   map[string]any{"dry_run": true, "request": map[string]any{"mcpId": 10487}},
		}}
		response, err := executeDevMCPServiceGet(context.Background(), runner, DevMCPServiceGetRequest{MCPID: 10487}, corecmd.OperationOptions{DryRun: true, CLIPath: "dev mcp service get"})
		if err != nil {
			t.Fatal(err)
		}
		if len(runner.invocations) != 1 || !runner.invocations[0].DryRun {
			t.Fatalf("invocations = %#v", runner.invocations)
		}
		if response.MCPID != "" || len(response.preview) == 0 {
			t.Fatalf("dry-run response = %#v", response)
		}
	})

	t.Run("malformed response", func(t *testing.T) {
		runner := &devMCPServiceGetOperationRunner{result: executor.Result{
			Invocation: executor.Invocation{Implemented: true},
			Response:   map[string]any{"content": "not an object"},
		}}
		_, err := executeDevMCPServiceGet(context.Background(), runner, DevMCPServiceGetRequest{MCPID: 10487}, corecmd.OperationOptions{CLIPath: "dev mcp service get"})
		if err == nil || !strings.Contains(err.Error(), devMCPServiceGetTool) {
			t.Fatalf("error = %v, want typed decode failure", err)
		}
	})

	for _, tc := range []struct {
		name    string
		content map[string]any
		want    string
	}{
		{name: "null string", content: map[string]any{"mcpId": "10487", "name": nil}, want: "field name must not be null"},
		{name: "null array", content: map[string]any{"mcpId": "10487", "categories": nil}, want: "field categories must not be null"},
		{name: "null array item", content: map[string]any{"mcpId": "10487", "categories": []any{nil}}, want: "field categories[0] must not be null"},
		{name: "null nested string", content: map[string]any{"mcpId": "10487", "tools": []any{map[string]any{"name": nil}}}, want: "field tools[0].name must not be null"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			runner := &devMCPServiceGetOperationRunner{result: executor.Result{
				Invocation: executor.Invocation{Implemented: true},
				Response:   map[string]any{"content": tc.content},
			}}
			_, err := executeDevMCPServiceGet(context.Background(), runner, DevMCPServiceGetRequest{MCPID: 10487}, corecmd.OperationOptions{CLIPath: "dev mcp service get"})
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error = %v, want %q", err, tc.want)
			}
		})
	}

	t.Run("runner error", func(t *testing.T) {
		sentinel := errors.New("runner failed")
		runner := &devMCPServiceGetOperationRunner{err: sentinel}
		_, err := executeDevMCPServiceGet(context.Background(), runner, DevMCPServiceGetRequest{MCPID: 10487}, corecmd.OperationOptions{CLIPath: "dev mcp service get"})
		if !errors.Is(err, sentinel) {
			t.Fatalf("error = %v, want runner error", err)
		}
	})
}

type devMCPServiceGetOperationRunner struct {
	invocations []executor.Invocation
	result      executor.Result
	err         error
}

func (r *devMCPServiceGetOperationRunner) Run(_ context.Context, invocation executor.Invocation) (executor.Result, error) {
	r.invocations = append(r.invocations, invocation)
	if r.err != nil {
		return executor.Result{}, r.err
	}
	result := r.result
	implemented := result.Invocation.Implemented
	result.Invocation = invocation
	result.Invocation.Implemented = implemented
	return result, nil
}
