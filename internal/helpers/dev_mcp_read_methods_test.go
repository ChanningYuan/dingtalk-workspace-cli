package helpers

import (
	"reflect"
	"testing"

	"github.com/DingTalk-Real-AI/dingtalk-workspace-cli/internal/corecmd"
	"github.com/DingTalk-Real-AI/dingtalk-workspace-cli/internal/corecmd/contract"
	"github.com/DingTalk-Real-AI/dingtalk-workspace-cli/internal/corecmd/contractfinal"
	"github.com/DingTalk-Real-AI/dingtalk-workspace-cli/internal/corecmd/runtimeannotate"
	"github.com/DingTalk-Real-AI/dingtalk-workspace-cli/internal/executor"
	"github.com/spf13/cobra"
)

func TestDevMCPURLAndAuthReadMethodsParity(t *testing.T) {
	cases := []struct {
		name string
		tool string
		path string
	}{
		{name: "url", tool: devMCPServerURLGetTool, path: "dev mcp url get"},
		{name: "auth", tool: devMCPAuthConfigGetTool, path: "dev mcp auth get"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var cmd *cobra.Command
			if tc.name == "url" {
				cmd = buildDevMCPURLGetTestCommand(&captureRunner{})
			} else {
				cmd = buildDevMCPAuthGetTestCommand(&captureRunner{})
			}
			final, ok := contractfinal.RuntimeContractFinal(cmd)
			if !ok || final.Identity == nil || final.Safety == nil {
				t.Fatalf("ContractFinal = %#v, registered = %t", final, ok)
			}
			if final.Identity.ProductID != "dev" || final.Identity.Name != tc.tool || final.Identity.CanonicalPath != "dev."+tc.tool || final.Identity.CLIPath != tc.path {
				t.Fatalf("identity = %#v", *final.Identity)
			}
			wantSafety := contract.SafetySpec{Effect: "read", EffectSource: "corecmd.contract", Risk: "medium", Confirmation: "not_required", Idempotency: "idempotent"}
			if *final.Safety != wantSafety || final.DryRun != nil {
				t.Fatalf("contract = safety:%#v dry_run:%#v", final.Safety, final.DryRun)
			}
			mcpID := cmd.Flags().Lookup("mcp-id")
			if mcpID == nil || firstFlagAnnotation(mcpID, runtimeannotate.AnnotationFlagProperty) != "mcpId" || firstFlagAnnotation(mcpID, runtimeannotate.AnnotationFlagType) != "integer" {
				t.Fatalf("--mcp-id metadata = %#v", mcpID)
			}
			if tc.name == "url" {
				source := cmd.Flags().Lookup("source")
				if source == nil || source.DefValue != "MARKET" || firstFlagAnnotation(source, runtimeannotate.AnnotationFlagProperty) != "source" {
					t.Fatalf("--source metadata = %#v", source)
				}
			}
		})
	}
}

func TestDevMCPURLAndAuthReadMethodsBackendParity(t *testing.T) {
	t.Run("url uppercases source", func(t *testing.T) {
		runner := &devMCPToolReadRecordingRunner{}
		root := newDevAppTestRoot(runner)
		root.SetArgs([]string{"dev", "mcp", "url", "get", "--mcp-id", "10487", "--source", "published", "--dry-run"})
		if err := corecmd.ExecuteForTest(root); err != nil {
			t.Fatalf("Execute() error = %v", err)
		}
		if len(runner.calls) != 1 {
			t.Fatalf("backend calls = %d, want 1", len(runner.calls))
		}
		want := map[string]any{"mcpId": 10487, "source": "PUBLISHED"}
		if runner.calls[0].Tool != devMCPServerURLGetTool || !reflect.DeepEqual(runner.calls[0].Params, want) || !runner.calls[0].DryRun {
			t.Fatalf("invocation = %#v, want tool=%q params=%#v dry_run=true", runner.calls[0], devMCPServerURLGetTool, want)
		}
	})

	t.Run("auth get sends mcp id", func(t *testing.T) {
		runner := &devMCPToolReadRecordingRunner{}
		root := newDevAppTestRoot(runner)
		root.SetArgs([]string{"dev", "mcp", "auth", "get", "--mcp-id", "10520"})
		if err := corecmd.ExecuteForTest(root); err != nil {
			t.Fatalf("Execute() error = %v", err)
		}
		if len(runner.calls) != 1 || runner.calls[0].Tool != devMCPAuthConfigGetTool || !reflect.DeepEqual(runner.calls[0].Params, map[string]any{"mcpId": 10520}) {
			t.Fatalf("calls = %#v", runner.calls)
		}
	})
}

func buildDevMCPURLGetTestCommand(runner executor.Runner) *cobra.Command {
	root := &cobra.Command{Use: "dws"}
	url := newDevMCPTestGroup(root, "url")
	if err := addDevMCPURLGetMethod(runner, url); err != nil {
		panic("add url get method: " + err.Error())
	}
	return childCommand(url, "get")
}

func buildDevMCPAuthGetTestCommand(runner executor.Runner) *cobra.Command {
	root := &cobra.Command{Use: "dws"}
	auth := newDevMCPTestGroup(root, "auth")
	if err := addDevMCPAuthGetMethod(runner, auth); err != nil {
		panic("add auth get method: " + err.Error())
	}
	return childCommand(auth, "get")
}
