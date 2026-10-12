package helpers

import (
	"context"
	stderrors "errors"
	"reflect"
	"strings"
	"testing"

	"github.com/DingTalk-Real-AI/dingtalk-workspace-cli/internal/corecmd"
	"github.com/DingTalk-Real-AI/dingtalk-workspace-cli/internal/corecmd/contract"
	"github.com/DingTalk-Real-AI/dingtalk-workspace-cli/internal/corecmd/contractfinal"
	"github.com/DingTalk-Real-AI/dingtalk-workspace-cli/internal/corecmd/runtimeannotate"
	"github.com/DingTalk-Real-AI/dingtalk-workspace-cli/internal/executor"
	"github.com/spf13/cobra"
)

func TestDevMCPToolReadMethodsContractAndSchemaParity(t *testing.T) {
	cases := []struct {
		name       string
		tool       string
		cliPath    string
		parameters []devMCPServiceParameterCase
	}{
		{
			name:    "list",
			tool:    devMCPToolListTool,
			cliPath: "dev mcp tool list",
			parameters: []devMCPServiceParameterCase{
				{name: "mcp-id", property: "mcpId", typeName: "integer"},
				{name: "cursor", property: "cursor", typeName: "integer"},
				{name: "page-size", property: "pageSize", typeName: "integer"},
				{name: "keyword", property: "keyword", typeName: "string"},
			},
		},
		{
			name:    "get",
			tool:    devMCPToolGetTool,
			cliPath: "dev mcp tool get",
			parameters: []devMCPServiceParameterCase{
				{name: "mcp-id", property: "mcpId", typeName: "integer"},
				{name: "tool-id", property: "toolId", typeName: "string"},
				{name: "version-id", property: "versionId", typeName: "string"},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cmd := buildDevMCPToolReadTestCommand(&captureRunner{}, tc.name)
			final, ok := contractfinal.RuntimeContractFinal(cmd)
			if !ok || final.Identity == nil || final.Safety == nil {
				t.Fatalf("ContractFinal = %#v, registered = %t", final, ok)
			}
			identity := final.Identity
			if identity.ProductID != "dev" || identity.Name != tc.tool || identity.CanonicalPath != "dev."+tc.tool || identity.CLIPath != tc.cliPath || identity.PrimaryCLIPath != tc.cliPath {
				t.Fatalf("identity = %#v", *identity)
			}
			wantSafety := contract.SafetySpec{Effect: "read", EffectSource: "corecmd.contract", Risk: "medium", Confirmation: "not_required", Idempotency: "idempotent"}
			if *final.Safety != wantSafety {
				t.Fatalf("safety = %#v, want %#v", *final.Safety, wantSafety)
			}
			if got := visibleFlagCount(cmd); got != len(tc.parameters) {
				t.Fatalf("visible flag count = %d", got)
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
			}
			if tc.name == "get" {
				actionID := cmd.Flags().Lookup("action-id")
				if actionID == nil || !actionID.Hidden {
					t.Fatalf("action-id = %#v, want hidden compatibility flag", actionID)
				}
			}
		})
	}
}

func TestDevMCPToolReadMethodsBackendParity(t *testing.T) {
	cases := []struct {
		name   string
		args   []string
		tool   string
		params map[string]any
		dryRun bool
	}{
		{
			name: "list",
			args: []string{"dev", "mcp", "tool", "list", "--mcp-id", "10487", "--cursor", "2", "--page-size", "20", "--keyword", "天气"},
			tool: devMCPToolListTool,
			params: map[string]any{
				"mcpId": 10487, "cursor": 2, "pageSize": 20, "keyword": "天气",
			},
		},
		{
			name:   "list omits empty optional fields",
			args:   []string{"dev", "mcp", "tool", "list", "--mcp-id", "10487"},
			tool:   devMCPToolListTool,
			params: map[string]any{"mcpId": 10487},
		},
		{
			name:   "get",
			args:   []string{"dev", "mcp", "tool", "get", "--mcp-id", "10487", "--tool-id", "G-ACT-1", "--version-id", "v2", "--dry-run"},
			tool:   devMCPToolGetTool,
			params: map[string]any{"mcpId": 10487, "toolId": "G-ACT-1", "versionId": "v2"},
			dryRun: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			runner := &devMCPToolReadRecordingRunner{}
			root := newDevAppTestRoot(runner)
			root.SetArgs(tc.args)
			if err := corecmd.ExecuteForTest(root); err != nil {
				t.Fatalf("Execute() error = %v", err)
			}
			if len(runner.calls) != 1 {
				t.Fatalf("backend calls = %d, want 1", len(runner.calls))
			}
			call := runner.calls[0]
			if call.Tool != tc.tool || !reflect.DeepEqual(call.Params, tc.params) || call.DryRun != tc.dryRun {
				t.Fatalf("invocation = %#v, want tool=%q params=%#v dry_run=%t", call, tc.tool, tc.params, tc.dryRun)
			}
		})
	}
}

func TestDevMCPToolReadMethodsValidationParity(t *testing.T) {
	cases := []struct {
		name string
		args []string
	}{
		{name: "list missing mcp-id", args: []string{"dev", "mcp", "tool", "list"}},
		{name: "get missing mcp-id", args: []string{"dev", "mcp", "tool", "get", "--tool-id", "G-ACT-1"}},
		{name: "get missing tool-id", args: []string{"dev", "mcp", "tool", "get", "--mcp-id", "10487"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			runner := &devMCPToolReadRecordingRunner{}
			root := newDevAppTestRoot(runner)
			root.SetArgs(tc.args)
			err := corecmd.ExecuteForTest(root)
			if err == nil || len(runner.calls) != 0 {
				t.Fatalf("error = %v, calls = %d", err, len(runner.calls))
			}
		})
	}

	t.Run("action-id is rejected before backend", func(t *testing.T) {
		runner := &devMCPToolReadRecordingRunner{}
		root := newDevAppTestRoot(runner)
		root.SetArgs([]string{"dev", "mcp", "tool", "get", "--mcp-id", "10487", "--action-id", "G-ACT-1"})
		err := corecmd.ExecuteForTest(root)
		if err == nil || !strings.Contains(err.Error(), "--action-id 已更名为 --tool-id") || len(runner.calls) != 0 {
			t.Fatalf("error = %v, calls = %d", err, len(runner.calls))
		}
	})

	t.Run("backend error is returned once", func(t *testing.T) {
		sentinel := stderrors.New("backend failed")
		runner := &devMCPToolReadRecordingRunner{err: sentinel}
		root := newDevAppTestRoot(runner)
		root.SetArgs([]string{"dev", "mcp", "tool", "get", "--mcp-id", "10487", "--tool-id", "G-ACT-1"})
		err := corecmd.ExecuteForTest(root)
		if !stderrors.Is(err, sentinel) || len(runner.calls) != 1 {
			t.Fatalf("error = %v, calls = %d", err, len(runner.calls))
		}
	})

}

func TestDevMCPToolVersionsMethodParity(t *testing.T) {
	cmd := buildDevMCPToolReadTestCommand(&captureRunner{}, "versions")
	final, ok := contractfinal.RuntimeContractFinal(cmd)
	if !ok || final.Identity == nil || final.Safety == nil {
		t.Fatalf("ContractFinal = %#v, registered = %t", final, ok)
	}
	if final.Identity.ProductID != "dev" || final.Identity.Name != devMCPToolVersionsTool || final.Identity.CanonicalPath != "dev."+devMCPToolVersionsTool || final.Identity.CLIPath != "dev mcp tool versions" {
		t.Fatalf("identity = %#v", *final.Identity)
	}
	wantSafety := contract.SafetySpec{Effect: "read", EffectSource: "corecmd.contract", Risk: "medium", Confirmation: "not_required", Idempotency: "idempotent"}
	if *final.Safety != wantSafety || final.DryRun != nil || final.Pagination != nil {
		t.Fatalf("contract = safety:%#v dry_run:%#v pagination:%#v", final.Safety, final.DryRun, final.Pagination)
	}
	for _, want := range []devMCPServiceParameterCase{
		{name: "mcp-id", property: "mcpId", typeName: "integer"},
		{name: "tool-id", property: "toolId", typeName: "string"},
		{name: "cursor", property: "cursor", typeName: "integer"},
		{name: "page-size", property: "pageSize", typeName: "integer"},
	} {
		flag := cmd.Flags().Lookup(want.name)
		if flag == nil || firstFlagAnnotation(flag, runtimeannotate.AnnotationFlagProperty) != want.property || firstFlagAnnotation(flag, runtimeannotate.AnnotationFlagType) != want.typeName {
			t.Fatalf("--%s metadata = %#v", want.name, flag)
		}
	}
	if actionID := cmd.Flags().Lookup("action-id"); actionID == nil || !actionID.Hidden {
		t.Fatalf("action-id = %#v, want hidden compatibility flag", actionID)
	}

	runner := &devMCPToolReadRecordingRunner{}
	root := newDevAppTestRoot(runner)
	root.SetArgs([]string{"dev", "mcp", "tool", "versions", "--mcp-id", "10487", "--tool-id", "G-ACT-1", "--cursor", "2", "--page-size", "20", "--dry-run"})
	if err := corecmd.ExecuteForTest(root); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if len(runner.calls) != 1 {
		t.Fatalf("backend calls = %d, want 1", len(runner.calls))
	}
	wantParams := map[string]any{"mcpId": 10487, "toolId": "G-ACT-1", "cursor": 2, "pageSize": 20}
	call := runner.calls[0]
	if call.Tool != devMCPToolVersionsTool || !reflect.DeepEqual(call.Params, wantParams) || !call.DryRun {
		t.Fatalf("invocation = %#v, want tool=%q params=%#v dry_run=true", call, devMCPToolVersionsTool, wantParams)
	}
}

func buildDevMCPToolReadTestCommand(runner executor.Runner, name string) *cobra.Command {
	root := &cobra.Command{Use: "dws"}
	tool := newDevMCPTestGroup(root, "tool")
	if err := addDevMCPToolReadMethods(runner, tool); err != nil {
		panic("add tool read methods: " + err.Error())
	}
	if err := addDevMCPToolVersionsMethod(runner, tool); err != nil {
		panic("add tool versions method: " + err.Error())
	}
	return childCommand(tool, name)
}

type devMCPToolReadRecordingRunner struct {
	calls []executor.Invocation
	err   error
}

func (r *devMCPToolReadRecordingRunner) Run(_ context.Context, invocation executor.Invocation) (executor.Result, error) {
	r.calls = append(r.calls, invocation)
	if r.err != nil {
		return executor.Result{}, r.err
	}
	return executor.Result{Invocation: invocation}, nil
}

var _ executor.Runner = (*devMCPToolReadRecordingRunner)(nil)
