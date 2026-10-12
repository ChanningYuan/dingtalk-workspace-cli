package helpers

import (
	"bytes"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/DingTalk-Real-AI/dingtalk-workspace-cli/internal/corecmd"
	"github.com/DingTalk-Real-AI/dingtalk-workspace-cli/internal/corecmd/contract"
	"github.com/DingTalk-Real-AI/dingtalk-workspace-cli/internal/corecmd/contractfinal"
	"github.com/DingTalk-Real-AI/dingtalk-workspace-cli/internal/corecmd/runtimeannotate"
)

func TestDevMCPToolLifecycleMethodsContractParity(t *testing.T) {
	for _, tc := range []struct {
		name   string
		tool   string
		effect string
		risk   string
	}{
		{name: "debug", tool: devMCPToolDebugTool, effect: "write", risk: "medium"},
		{name: "publish", tool: devMCPToolPublishTool, effect: "write", risk: "medium"},
		{name: "delete", tool: devMCPToolDeleteTool, effect: "destructive", risk: "high"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cmd := buildDevMCPToolLifecycleTestCommand(&captureRunner{}, tc.name)
			final, ok := contractfinal.RuntimeContractFinal(cmd)
			if !ok || final.Identity == nil || final.Safety == nil || final.DryRun == nil {
				t.Fatalf("ContractFinal = %#v, registered = %t", final, ok)
			}
			if final.Identity.ProductID != "dev" || final.Identity.Name != tc.tool || final.Identity.CanonicalPath != "dev."+tc.tool || final.Identity.CLIPath != "dev mcp tool "+tc.name {
				t.Fatalf("identity = %#v", *final.Identity)
			}
			wantSafety := contract.SafetySpec{Effect: tc.effect, EffectSource: "corecmd.contract", Risk: tc.risk, Confirmation: "user_required", Idempotency: "unknown"}
			if *final.Safety != wantSafety || final.DryRun.PreviewKind != contract.DryRunPreviewInvocation || final.DryRun.RemoteReads {
				t.Fatalf("contract = safety:%#v dry_run:%#v", final.Safety, final.DryRun)
			}
			for name, property := range map[string]string{"mcp-id": "mcpId", "tool-id": "toolId"} {
				flag := cmd.Flags().Lookup(name)
				if flag == nil || firstFlagAnnotation(flag, runtimeannotate.AnnotationFlagProperty) != property {
					t.Fatalf("--%s metadata = %#v", name, flag)
				}
			}
		})
	}
}

func TestDevMCPToolLifecycleMethodsDebugPayloadParity(t *testing.T) {
	runner := &devMCPToolReadRecordingRunner{}
	root := newDevAppTestRoot(runner)
	root.SetArgs([]string{
		"dev", "mcp", "tool", "debug",
		"--mcp-id", "10487",
		"--tool-id", "G-ACT-1",
		"--value", `{"city":"杭州"}`,
		"--credential-id", "10518",
		"--version-id", "v2",
		"--dry-run",
	})
	if err := corecmd.ExecuteForTest(root); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	want := map[string]any{
		"mcpId":        10487,
		"toolId":       "G-ACT-1",
		"value":        map[string]any{"city": "杭州"},
		"credentialId": 10518,
		"versionId":    "v2",
	}
	if len(runner.calls) != 1 || runner.calls[0].Tool != devMCPToolDebugTool || !reflect.DeepEqual(runner.calls[0].Params, want) || !runner.calls[0].DryRun {
		t.Fatalf("calls = %#v, want tool=%q params=%#v dry_run=true", runner.calls, devMCPToolDebugTool, want)
	}
}

func TestDevMCPToolLifecycleMethodsLocatorPayloadParity(t *testing.T) {
	for _, tc := range []struct{ name, tool string }{
		{"delete", devMCPToolDeleteTool},
		{"publish", devMCPToolPublishTool},
	} {
		t.Run(tc.name, func(t *testing.T) {
			runner := &devMCPToolReadRecordingRunner{}
			root := newDevAppTestRoot(runner)
			var stdout, stderr bytes.Buffer
			root.SetOut(&stdout)
			root.SetErr(&stderr)
			root.SetArgs([]string{"dev", "mcp", "tool", tc.name, "--mcp-id", "10487", "--tool-id", " G-ACT-1 ", "--dry-run"})
			if err := corecmd.ExecuteForTest(root); err != nil {
				t.Fatal(err)
			}
			want := map[string]any{"mcpId": 10487, "toolId": "G-ACT-1"}
			if len(runner.calls) != 1 || runner.calls[0].Tool != tc.tool || !runner.calls[0].DryRun || !reflect.DeepEqual(runner.calls[0].Params, want) {
				t.Fatalf("calls = %#v, want tool=%q params=%#v", runner.calls, tc.tool, want)
			}
			if stderr.Len() != 0 {
				t.Fatalf("dry-run stderr = %q", stderr.String())
			}
		})
	}
}

func TestDevMCPToolLifecycleMethodsConfirmationAndValidationParity(t *testing.T) {
	for _, tc := range []struct {
		name string
		args []string
		want string
	}{
		{"debug confirmation", []string{"debug", "--mcp-id", "1", "--tool-id", "G-ACT-1", "--value", "{}", "--no-credential"}, "--yes"},
		{"publish confirmation", []string{"publish", "--mcp-id", "1", "--tool-id", "G-ACT-1"}, "--yes"},
		{"delete confirmation", []string{"delete", "--mcp-id", "1", "--tool-id", "G-ACT-1"}, "--yes"},
		{"debug JSON", []string{"debug", "--mcp-id", "1", "--tool-id", "G-ACT-1", "--value", "[]", "--no-credential"}, "--value 必须是 JSON 对象"},
		{"debug conflicting credential", []string{"debug", "--mcp-id", "1", "--tool-id", "G-ACT-1", "--value", "{}", "--no-credential", "--credential-id", "2"}, "只能使用一个"},
		{"delete legacy flag", []string{"delete", "--mcp-id", "1", "--action-id", "G-ACT-1"}, "--action-id 已更名"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			runner := &devMCPToolReadRecordingRunner{}
			root := newDevAppTestRoot(runner)
			root.SetArgs(append([]string{"dev", "mcp", "tool"}, tc.args...))
			err := corecmd.ExecuteForTest(root)
			if err == nil || !strings.Contains(err.Error(), tc.want) || len(runner.calls) != 0 {
				t.Fatalf("error = %v, calls = %#v", err, runner.calls)
			}
		})
	}
}

func TestDevMCPToolLifecycleMethodsPublishOrderAndWarningParity(t *testing.T) {
	runner := &devMCPToolReadRecordingRunner{}
	root := newDevAppTestRoot(runner)
	var stdout, stderr bytes.Buffer
	root.SetOut(&stdout)
	root.SetErr(&stderr)
	root.SetArgs([]string{"dev", "mcp", "tool", "publish", "--mcp-id", "10487", "--tool-id", "G-ACT-1", "--yes"})
	if err := corecmd.ExecuteForTest(root); err != nil {
		t.Fatal(err)
	}
	want := map[string]any{"mcpId": 10487, "toolId": "G-ACT-1"}
	if len(runner.calls) != 2 || runner.calls[0].Tool != devMCPToolGetTool || runner.calls[1].Tool != devMCPToolPublishTool {
		t.Fatalf("calls = %#v", runner.calls)
	}
	for _, call := range runner.calls {
		if call.DryRun || !reflect.DeepEqual(call.Params, want) {
			t.Fatalf("call = %#v, want params=%#v", call, want)
		}
	}
	if !strings.Contains(stderr.String(), "跳过静态复验") || !strings.Contains(stderr.String(), "提示：发布后可通过 dws mcp published tools/invoke") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestDevMCPToolLifecycleMethodsPublishReadErrorParity(t *testing.T) {
	backendErr := errors.New("read failed")
	runner := &devMCPToolReadRecordingRunner{err: backendErr}
	root := newDevAppTestRoot(runner)
	var stderr bytes.Buffer
	root.SetErr(&stderr)
	root.SetArgs([]string{"dev", "mcp", "tool", "publish", "--mcp-id", "10487", "--tool-id", "G-ACT-1", "--yes"})
	err := corecmd.ExecuteForTest(root)
	if !errors.Is(err, backendErr) || len(runner.calls) != 1 || runner.calls[0].Tool != devMCPToolGetTool || strings.Contains(stderr.String(), "提示：发布后") {
		t.Fatalf("error = %v, calls = %#v, stderr = %q", err, runner.calls, stderr.String())
	}
}
