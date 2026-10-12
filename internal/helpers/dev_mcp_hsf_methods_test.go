package helpers

import (
	"reflect"
	"testing"

	"github.com/DingTalk-Real-AI/dingtalk-workspace-cli/internal/corecmd"
	"github.com/DingTalk-Real-AI/dingtalk-workspace-cli/internal/corecmd/contract"
	"github.com/DingTalk-Real-AI/dingtalk-workspace-cli/internal/corecmd/contractfinal"
	"github.com/DingTalk-Real-AI/dingtalk-workspace-cli/internal/corecmd/runtimeannotate"
	"github.com/spf13/cobra"
)

func TestDevMCPHsfMethodsContractParity(t *testing.T) {
	for _, tc := range []struct {
		name   string
		tool   string
		effect string
		risk   string
	}{
		{"method-list", devMCPHsfMethodListTool, "read", "medium"},
		{"create-hsf", devMCPToolCreateHsfTool, "write", "medium"},
		{"update-hsf", devMCPToolUpdateHsfTool, "write", "medium"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var cmdPath string
			var cmd *cobra.Command
			switch tc.name {
			case "method-list":
				cmd = buildDevMCPHsfMethodTestCommand(&captureRunner{}, tc.name)
			default:
				cmd = buildDevMCPToolHsfMethodTestCommand(&captureRunner{}, tc.name)
			}
			switch tc.name {
			case "method-list":
				cmdPath = "dev mcp hsf method-list"
			default:
				cmdPath = "dev mcp tool " + tc.name
			}
			final, ok := contractfinal.RuntimeContractFinal(cmd)
			if !ok || final.Identity == nil || final.Safety == nil {
				t.Fatalf("ContractFinal = %#v, registered = %t", final, ok)
			}
			if final.Identity.ProductID != "dev" || final.Identity.Name != tc.tool || final.Identity.CanonicalPath != "dev."+tc.tool || final.Identity.CLIPath != cmdPath {
				t.Fatalf("identity = %#v", *final.Identity)
			}
			idempotency := "unknown"
			if tc.effect == "read" {
				idempotency = "idempotent"
			}
			wantSafety := contract.SafetySpec{Effect: tc.effect, EffectSource: "corecmd.contract", Risk: tc.risk, Confirmation: "not_required", Idempotency: idempotency}
			if tc.effect != "read" {
				wantSafety.Confirmation = "user_required"
			}
			if *final.Safety != wantSafety {
				t.Fatalf("safety = %#v, want %#v", *final.Safety, wantSafety)
			}
			if tc.effect == "read" {
				if final.DryRun != nil {
					t.Fatalf("read dry_run = %#v", final.DryRun)
				}
			} else if final.DryRun == nil || final.DryRun.PreviewKind != contract.DryRunPreviewInvocation || final.DryRun.RemoteReads {
				t.Fatalf("write dry_run = %#v", final.DryRun)
			}
			if flag := cmd.Flags().Lookup("mcp-id"); tc.name != "method-list" && (flag == nil || firstFlagAnnotation(flag, runtimeannotate.AnnotationFlagProperty) != "mcpId") {
				t.Fatalf("mcp-id metadata = %#v", flag)
			}
		})
	}
}

func TestDevMCPHsfMethodListPayloadParity(t *testing.T) {
	runner := &devMCPToolReadRecordingRunner{}
	root := newDevAppTestRoot(runner)
	root.SetArgs([]string{"dev", "mcp", "hsf", "method-list", "--interface-name", "a.b.C", "--version", "1.0.0", "--dry-run"})
	if err := corecmd.ExecuteForTest(root); err != nil {
		t.Fatal(err)
	}
	want := map[string]any{"interfaceName": "a.b.C", "version": "1.0.0"}
	if len(runner.calls) != 1 || runner.calls[0].Tool != devMCPHsfMethodListTool || !reflect.DeepEqual(runner.calls[0].Params, want) || !runner.calls[0].DryRun {
		t.Fatalf("calls = %#v, want %#v", runner.calls, want)
	}
}
