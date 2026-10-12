package helpers

import (
	"reflect"
	"strings"
	"testing"

	"github.com/DingTalk-Real-AI/dingtalk-workspace-cli/internal/corecmd"
	"github.com/DingTalk-Real-AI/dingtalk-workspace-cli/internal/corecmd/contract"
	"github.com/DingTalk-Real-AI/dingtalk-workspace-cli/internal/corecmd/contractfinal"
	"github.com/DingTalk-Real-AI/dingtalk-workspace-cli/internal/corecmd/runtimeannotate"
)

func TestDevMCPCredentialWriteMethodsContractParity(t *testing.T) {
	for _, tc := range []struct {
		name   string
		tool   string
		effect string
		risk   string
	}{
		{name: "save", tool: devMCPCredentialSaveTool, effect: "write", risk: "medium"},
		{name: "debug", tool: devMCPCredentialDebugTool, effect: "write", risk: "medium"},
		{name: "bind", tool: devMCPCredentialBindTool, effect: "write", risk: "medium"},
		{name: "unbind", tool: devMCPCredentialUnbindTool, effect: "write", risk: "medium"},
		{name: "delete", tool: devMCPCredentialDeleteTool, effect: "destructive", risk: "high"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cmd := buildDevMCPCredentialWriteTestCommand(&captureRunner{}, tc.name)
			final, ok := contractfinal.RuntimeContractFinal(cmd)
			if !ok || final.Identity == nil || final.Safety == nil || final.DryRun == nil {
				t.Fatalf("ContractFinal = %#v, registered = %t", final, ok)
			}
			if final.Identity.ProductID != "dev" || final.Identity.Name != tc.tool || final.Identity.CanonicalPath != "dev."+tc.tool || final.Identity.CLIPath != "dev mcp credential "+tc.name {
				t.Fatalf("identity = %#v", *final.Identity)
			}
			wantSafety := contract.SafetySpec{Effect: tc.effect, EffectSource: "corecmd.contract", Risk: tc.risk, Confirmation: "user_required", Idempotency: "unknown"}
			if *final.Safety != wantSafety || final.DryRun.PreviewKind != contract.DryRunPreviewInvocation || final.DryRun.RemoteReads {
				t.Fatalf("contract = safety:%#v dry_run:%#v", final.Safety, final.DryRun)
			}
			for name, property := range map[string]string{"mcp-id": "mcpId"} {
				flag := cmd.Flags().Lookup(name)
				if flag == nil || firstFlagAnnotation(flag, runtimeannotate.AnnotationFlagProperty) != property {
					t.Fatalf("--%s metadata = %#v", name, flag)
				}
			}
			if tc.name != "unbind" {
				flag := cmd.Flags().Lookup("credential-id")
				if flag == nil || firstFlagAnnotation(flag, runtimeannotate.AnnotationFlagProperty) != "credentialId" {
					t.Fatalf("--credential-id metadata = %#v", flag)
				}
			}
		})
	}
}

func TestDevMCPCredentialWriteMethodsBackendParity(t *testing.T) {
	for _, tc := range []struct {
		name   string
		tool   string
		params map[string]any
		args   []string
	}{
		{
			name:   "save",
			tool:   devMCPCredentialSaveTool,
			params: map[string]any{"mcpId": 10520, "name": "account", "content": map[string]any{"redacted": true}},
			args:   []string{"--mcp-id", "10520", "--name", "account", "--content", `{"apiKey":"secret"}`, "--dry-run"},
		},
		{
			name:   "debug",
			tool:   devMCPCredentialDebugTool,
			params: map[string]any{"mcpId": 10520, "credentialId": 10001},
			args:   []string{"--mcp-id", "10520", "--credential-id", "10001", "--dry-run"},
		},
		{
			name:   "bind",
			tool:   devMCPCredentialBindTool,
			params: map[string]any{"mcpId": 10520, "credentialId": 10001},
			args:   []string{"--mcp-id", "10520", "--credential-id", "10001", "--dry-run"},
		},
		{
			name:   "unbind",
			tool:   devMCPCredentialUnbindTool,
			params: map[string]any{"mcpId": 10520},
			args:   []string{"--mcp-id", "10520", "--dry-run"},
		},
		{
			name:   "delete",
			tool:   devMCPCredentialDeleteTool,
			params: map[string]any{"mcpId": 10520, "credentialId": 10001},
			args:   []string{"--mcp-id", "10520", "--credential-id", "10001", "--dry-run"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			runner := &devMCPToolReadRecordingRunner{}
			root := newDevAppTestRoot(runner)
			root.SetArgs(append([]string{"dev", "mcp", "credential", tc.name}, tc.args...))
			if err := corecmd.ExecuteForTest(root); err != nil {
				t.Fatalf("Execute() error = %v", err)
			}
			if len(runner.calls) != 1 || runner.calls[0].Tool != tc.tool || !reflect.DeepEqual(runner.calls[0].Params, tc.params) || !runner.calls[0].DryRun {
				t.Fatalf("calls = %#v, want tool=%q params=%#v dry_run=true", runner.calls, tc.tool, tc.params)
			}
		})
	}
}

func TestDevMCPCredentialWriteMethodsValidationParity(t *testing.T) {
	runner := &devMCPToolReadRecordingRunner{}
	root := newDevAppTestRoot(runner)
	root.SetArgs([]string{"dev", "mcp", "credential", "save", "--mcp-id", "10520", "--name", "account", "--content", "{}", "--content-file", "also.json"})
	if err := corecmd.ExecuteForTest(root); err == nil || !strings.Contains(err.Error(), "只能使用一个") || len(runner.calls) != 0 {
		t.Fatalf("error = %v, calls = %d", err, len(runner.calls))
	}
}
