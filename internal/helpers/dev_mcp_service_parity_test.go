package helpers

import (
	"bytes"
	"context"
	stderrors "errors"
	"reflect"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/DingTalk-Real-AI/dingtalk-workspace-cli/internal/corecmd"
	"github.com/DingTalk-Real-AI/dingtalk-workspace-cli/internal/corecmd/contract"
	"github.com/DingTalk-Real-AI/dingtalk-workspace-cli/internal/corecmd/contractfinal"
	"github.com/DingTalk-Real-AI/dingtalk-workspace-cli/internal/corecmd/runtimeannotate"
	apperrors "github.com/DingTalk-Real-AI/dingtalk-workspace-cli/internal/errors"
	"github.com/DingTalk-Real-AI/dingtalk-workspace-cli/internal/executor"
)

type devMCPServiceParityCase struct {
	name       string
	tool       string
	cliPath    string
	dryRun     bool
	safety     contract.SafetySpec
	parameters []devMCPServiceParameterCase
}

type devMCPServiceParameterCase struct {
	name     string
	property string
	typeName string
	required bool
}

func TestDevMCPServiceMethodsContractParity(t *testing.T) {
	for _, tc := range devMCPServiceParityCases() {
		t.Run(tc.name, func(t *testing.T) {
			cmd := buildDevMCPServiceTestCommand(&captureRunner{}, tc.name)
			final, ok := contractfinal.RuntimeContractFinal(cmd)
			if !ok {
				t.Fatal("missing ContractFinal")
			}
			if final.Identity == nil {
				t.Fatal("missing ContractFinal identity")
			}
			identity := final.Identity
			if identity.ProductID != "dev" || identity.Name != tc.tool ||
				identity.CanonicalPath != "dev."+tc.tool || identity.CLIPath != tc.cliPath ||
				identity.PrimaryCLIPath != tc.cliPath {
				t.Fatalf("identity = %#v", *identity)
			}
			if final.Interface == nil || final.Interface.Mode != contract.InterfaceModeComposite ||
				final.Interface.Availability != contract.InterfaceAvailable || final.Interface.Reason == "" {
				t.Fatalf("interface = %#v", final.Interface)
			}
			wantSafety := tc.safety
			wantSafety.EffectSource = "corecmd.contract"
			if final.Safety == nil || *final.Safety != wantSafety {
				t.Fatalf("safety = %#v, want %#v", final.Safety, wantSafety)
			}
			if tc.dryRun {
				if final.DryRun == nil || final.DryRun.PreviewKind != contract.DryRunPreviewInvocation || final.DryRun.RemoteReads {
					t.Fatalf("dry-run = %#v", final.DryRun)
				}
			} else if final.DryRun != nil {
				t.Fatalf("unexpected dry-run declaration: %#v", final.DryRun)
			}
		})
	}
}

func TestDevMCPServiceMethodsSchemaParameterParity(t *testing.T) {
	for _, tc := range devMCPServiceParityCases() {
		t.Run(tc.name, func(t *testing.T) {
			cmd := buildDevMCPServiceTestCommand(&captureRunner{}, tc.name)
			if got := visibleFlagCount(cmd); got != len(tc.parameters) {
				t.Fatalf("visible flag count = %d, want %d", got, len(tc.parameters))
			}
			for _, want := range tc.parameters {
				flag := cmd.Flags().Lookup(want.name)
				if flag == nil {
					t.Fatalf("missing --%s", want.name)
				}
				if got := firstFlagAnnotation(flag, runtimeannotate.AnnotationFlagProperty); got != want.property {
					t.Fatalf("--%s property = %q, want %q", want.name, got, want.property)
				}
				if got := firstFlagAnnotation(flag, runtimeannotate.AnnotationFlagType); got != want.typeName {
					t.Fatalf("--%s type = %q, want %q", want.name, got, want.typeName)
				}
				if got := firstFlagAnnotation(flag, runtimeannotate.AnnotationFlagRequired); got != boolString(want.required) {
					t.Fatalf("--%s required = %q, want %t", want.name, got, want.required)
				}
			}
		})
	}
}

func TestDevMCPServiceMethodsBackendArgsAndDryRunParity(t *testing.T) {
	cases := []struct {
		name   string
		args   []string
		tool   string
		params map[string]any
		dryRun bool
	}{
		{
			name: "list",
			args: []string{"dev", "mcp", "service", "list", "--cursor", "2", "--page-size", "20", "--keyword", "客户", "--creator-user-id", "staff-1"},
			tool: devMCPServiceListTool,
			params: map[string]any{
				"cursor": 2, "pageSize": 20, "keyword": "客户", "creatorUserId": "staff-1",
			},
		},
		{
			name:   "get",
			args:   []string{"dev", "mcp", "service", "get", "--mcp-id", "10487"},
			tool:   devMCPServiceGetTool,
			params: map[string]any{"mcpId": 10487},
		},
		{
			name: "create",
			args: []string{"dev", "mcp", "service", "create", "--name", "客户服务", "--description", "查询客户", "--icon-url", "https://example.com/icon.png", "--introduction", "介绍", "--server-name", "customer-info", "--dry-run"},
			tool: devMCPServiceCreateTool,
			params: map[string]any{
				"name": "客户服务", "description": "查询客户", "icon_url": "https://example.com/icon.png", "introduction": "介绍", "serverName": "customer-info",
			},
			dryRun: true,
		},
		{
			name: "update",
			args: []string{"dev", "mcp", "service", "update", "--mcp-id", "10487", "--name", "新名称", "--description", "新描述", "--icon-url", "https://example.com/new.png", "--introduction", "新介绍", "--server-name", "new-name", "--dry-run"},
			tool: devMCPServiceUpdateTool,
			params: map[string]any{
				"mcpId": 10487, "name": "新名称", "description": "新描述", "icon_url": "https://example.com/new.png", "introduction": "新介绍", "serverName": "new-name",
			},
			dryRun: true,
		},
		{
			name:   "delete",
			args:   []string{"dev", "mcp", "service", "delete", "--mcp-id", "10487", "--yes"},
			tool:   devMCPServiceDeleteTool,
			params: map[string]any{"mcpId": 10487},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			runner := &devMCPServiceRecordingRunner{}
			root := newDevAppTestRoot(runner)
			root.SetArgs(tc.args)
			if err := corecmd.ExecuteForTest(root); err != nil {
				t.Fatalf("Execute() error = %v", err)
			}
			if len(runner.calls) != 1 {
				t.Fatalf("backend calls = %d, want 1", len(runner.calls))
			}
			call := runner.calls[0]
			if call.Tool != tc.tool {
				t.Fatalf("tool = %q, want %q", call.Tool, tc.tool)
			}
			if !reflect.DeepEqual(call.Params, tc.params) {
				t.Fatalf("params = %#v, want %#v", call.Params, tc.params)
			}
			if call.DryRun != tc.dryRun {
				t.Fatalf("dry_run = %t, want %t", call.DryRun, tc.dryRun)
			}
		})
	}
}

func TestDevMCPServiceMethodsValidationConfirmationAndBackendErrors(t *testing.T) {
	t.Run("write requires confirmation before backend", func(t *testing.T) {
		runner := &devMCPServiceRecordingRunner{}
		root := newDevAppTestRoot(runner)
		root.SetIn(strings.NewReader(""))
		root.SetArgs([]string{"dev", "mcp", "service", "update", "--mcp-id", "10487", "--description", "新描述"})
		err := corecmd.ExecuteForTest(root)
		if !apperrors.IsConfirmationRequired(err) {
			t.Fatalf("error = %v, want confirmation_required", err)
		}
		if len(runner.calls) != 0 {
			t.Fatalf("backend calls = %d, want 0", len(runner.calls))
		}
	})

	t.Run("validation precedes confirmation", func(t *testing.T) {
		runner := &devMCPServiceRecordingRunner{}
		root := newDevAppTestRoot(runner)
		root.SetIn(strings.NewReader(""))
		root.SetArgs([]string{"dev", "mcp", "service", "create", "--name", "名称"})
		err := corecmd.ExecuteForTest(root)
		if err == nil || apperrors.IsConfirmationRequired(err) {
			t.Fatalf("error = %v, want validation error before confirmation", err)
		}
		if len(runner.calls) != 0 {
			t.Fatalf("backend calls = %d, want 0", len(runner.calls))
		}
	})

	t.Run("backend errors are returned without retry", func(t *testing.T) {
		sentinel := stderrors.New("backend failed")
		for _, tc := range devMCPServiceExecutionCases() {
			t.Run(tc.name, func(t *testing.T) {
				runner := &devMCPServiceRecordingRunner{err: sentinel}
				root := newDevAppTestRoot(runner)
				root.SetArgs(tc.args)
				err := corecmd.ExecuteForTest(root)
				if !stderrors.Is(err, sentinel) {
					t.Fatalf("error = %v, want backend error", err)
				}
				if len(runner.calls) != 1 {
					t.Fatalf("backend calls = %d, want 1", len(runner.calls))
				}
			})
		}
	})
}

func TestDevMCPServiceCreateWarningOnlyAfterSuccessfulBackend(t *testing.T) {
	args := []string{"dev", "mcp", "service", "create", "--name", "名称", "--description", "描述", "--dry-run"}

	t.Run("success emits warning", func(t *testing.T) {
		runner := &devMCPServiceRecordingRunner{}
		root := newDevAppTestRoot(runner)
		var output bytes.Buffer
		root.SetOut(&output)
		root.SetErr(&output)
		root.SetArgs(args)
		if err := corecmd.ExecuteForTest(root); err != nil {
			t.Fatalf("Execute() error = %v", err)
		}
		if len(runner.calls) != 1 {
			t.Fatalf("backend calls = %d, want 1", len(runner.calls))
		}
		if !strings.Contains(output.String(), "未设置 --server-name") {
			t.Fatalf("warning missing from output: %s", output.String())
		}
	})

	t.Run("backend error does not emit warning", func(t *testing.T) {
		sentinel := stderrors.New("backend failed")
		runner := &devMCPServiceRecordingRunner{err: sentinel}
		root := newDevAppTestRoot(runner)
		var output bytes.Buffer
		root.SetOut(&output)
		root.SetErr(&output)
		root.SetArgs(args)
		if err := corecmd.ExecuteForTest(root); !stderrors.Is(err, sentinel) {
			t.Fatalf("error = %v, want backend error", err)
		}
		if len(runner.calls) != 1 {
			t.Fatalf("backend calls = %d, want 1", len(runner.calls))
		}
		if strings.Contains(output.String(), "未设置 --server-name") {
			t.Fatalf("unexpected warning after backend error: %s", output.String())
		}
	})
}

func devMCPServiceParityCases() []devMCPServiceParityCase {
	return []devMCPServiceParityCase{
		{
			name: "list", tool: devMCPServiceListTool, cliPath: "dev mcp service list",
			safety: contract.SafetySpec{Effect: "read", Risk: "medium", Confirmation: "not_required", Idempotency: "idempotent"},
			parameters: []devMCPServiceParameterCase{
				{name: "cursor", property: "cursor", typeName: "integer"},
				{name: "page-size", property: "pageSize", typeName: "integer"},
				{name: "keyword", property: "keyword", typeName: "string"},
				{name: "creator-user-id", property: "creatorUserId", typeName: "string"},
			},
		},
		{
			name: "get", tool: devMCPServiceGetTool, cliPath: "dev mcp service get", dryRun: true,
			safety:     contract.SafetySpec{Effect: "read", Risk: "medium", Confirmation: "not_required", Idempotency: "idempotent"},
			parameters: []devMCPServiceParameterCase{{name: "mcp-id", property: "mcpId", typeName: "integer", required: true}},
		},
		{
			name: "create", tool: devMCPServiceCreateTool, cliPath: "dev mcp service create", dryRun: true,
			safety: contract.SafetySpec{Effect: "write", Risk: "medium", Confirmation: "user_required", Idempotency: "unknown"},
			parameters: []devMCPServiceParameterCase{
				{name: "name", property: "name", typeName: "string", required: true},
				{name: "description", property: "description", typeName: "string", required: true},
				{name: "icon-url", property: "icon_url", typeName: "string"},
				{name: "introduction", property: "introduction", typeName: "string"},
				{name: "server-name", property: "serverName", typeName: "string"},
			},
		},
		{
			name: "update", tool: devMCPServiceUpdateTool, cliPath: "dev mcp service update", dryRun: true,
			safety: contract.SafetySpec{Effect: "write", Risk: "medium", Confirmation: "user_required", Idempotency: "unknown"},
			parameters: []devMCPServiceParameterCase{
				{name: "mcp-id", property: "mcpId", typeName: "integer", required: true},
				{name: "name", property: "name", typeName: "string"},
				{name: "description", property: "description", typeName: "string"},
				{name: "icon-url", property: "icon_url", typeName: "string"},
				{name: "introduction", property: "introduction", typeName: "string"},
				{name: "server-name", property: "serverName", typeName: "string"},
			},
		},
		{
			name: "delete", tool: devMCPServiceDeleteTool, cliPath: "dev mcp service delete", dryRun: true,
			safety:     contract.SafetySpec{Effect: "destructive", Risk: "high", Confirmation: "user_required", Idempotency: "unknown"},
			parameters: []devMCPServiceParameterCase{{name: "mcp-id", property: "mcpId", typeName: "integer", required: true}},
		},
	}
}

func devMCPServiceExecutionCases() []struct {
	name string
	args []string
} {
	return []struct {
		name string
		args []string
	}{
		{name: "list", args: []string{"dev", "mcp", "service", "list"}},
		{name: "get", args: []string{"dev", "mcp", "service", "get", "--mcp-id", "10487"}},
		{name: "create", args: []string{"dev", "mcp", "service", "create", "--name", "名称", "--description", "描述", "--dry-run"}},
		{name: "update", args: []string{"dev", "mcp", "service", "update", "--mcp-id", "10487", "--description", "描述", "--dry-run"}},
		{name: "delete", args: []string{"dev", "mcp", "service", "delete", "--mcp-id", "10487", "--dry-run"}},
	}
}

type devMCPServiceRecordingRunner struct {
	calls []executor.Invocation
	err   error
}

func (r *devMCPServiceRecordingRunner) Run(_ context.Context, invocation executor.Invocation) (executor.Result, error) {
	r.calls = append(r.calls, invocation)
	if r.err != nil {
		return executor.Result{}, r.err
	}
	if invocation.Tool == devMCPServiceGetTool {
		invocation.Implemented = true
		return executor.Result{
			Invocation: invocation,
			Response: map[string]any{
				"content": map[string]any{
					"mcpId": "10487",
					"name":  "typed-test-service",
				},
			},
		}, nil
	}
	return executor.Result{Invocation: invocation}, nil
}

func firstFlagAnnotation(flag *pflag.Flag, key string) string {
	if flag == nil || flag.Annotations == nil {
		return ""
	}
	values := flag.Annotations[key]
	if len(values) == 0 {
		return ""
	}
	return values[0]
}

func visibleFlagCount(cmd *cobra.Command) int {
	count := 0
	cmd.Flags().VisitAll(func(flag *pflag.Flag) {
		if !flag.Hidden && flag.Name != "help" {
			count++
		}
	})
	return count
}

func boolString(value bool) string {
	if value {
		return "true"
	}
	return "false"
}

var _ executor.Runner = (*devMCPServiceRecordingRunner)(nil)
