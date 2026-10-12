package helpers

import (
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

func TestDevMCPMemberMutationMethodsContractParity(t *testing.T) {
	for _, tc := range []struct {
		name   string
		tool   string
		effect string
		risk   string
	}{
		{name: "add", tool: devMCPMemberAddTool, effect: "write", risk: "medium"},
		{name: "remove", tool: devMCPMemberRemoveTool, effect: "destructive", risk: "high"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cmd := buildDevMCPMemberMutationTestCommand(&captureRunner{}, tc.name)
			final, ok := contractfinal.RuntimeContractFinal(cmd)
			if !ok || final.Identity == nil || final.Safety == nil || final.DryRun == nil {
				t.Fatalf("ContractFinal = %#v, registered = %t", final, ok)
			}
			if final.Identity.ProductID != "dev" || final.Identity.Name != tc.tool || final.Identity.CanonicalPath != "dev."+tc.tool || final.Identity.CLIPath != "dev mcp member "+tc.name {
				t.Fatalf("identity = %#v", *final.Identity)
			}
			wantSafety := contract.SafetySpec{Effect: tc.effect, EffectSource: "corecmd.contract", Risk: tc.risk, Confirmation: "user_required", Idempotency: "unknown"}
			if *final.Safety != wantSafety || final.DryRun.PreviewKind != contract.DryRunPreviewInvocation || final.DryRun.RemoteReads {
				t.Fatalf("contract = safety:%#v dry_run:%#v", final.Safety, final.DryRun)
			}
			for name, property := range map[string]string{"mcp-id": "mcpId", "user-ids": "memberUserIds"} {
				flag := cmd.Flags().Lookup(name)
				if flag == nil || firstFlagAnnotation(flag, runtimeannotate.AnnotationFlagProperty) != property {
					t.Fatalf("--%s metadata = %#v", name, flag)
				}
			}
		})
	}
}

func TestDevMCPMemberMutationMethodsBackendParity(t *testing.T) {
	for _, tc := range []struct {
		name string
		tool string
	}{
		{name: "add", tool: devMCPMemberAddTool},
		{name: "remove", tool: devMCPMemberRemoveTool},
	} {
		t.Run(tc.name, func(t *testing.T) {
			runner := &devMCPToolReadRecordingRunner{}
			root := newDevAppTestRoot(runner)
			root.SetArgs([]string{"dev", "mcp", "member", tc.name, "--mcp-id", "10520", "--user-ids", "staff001, staff002;staff003", "--dry-run"})
			if err := corecmd.ExecuteForTest(root); err != nil {
				t.Fatalf("Execute() error = %v", err)
			}
			want := map[string]any{"mcpId": 10520, "memberUserIds": []string{"staff001", "staff002", "staff003"}}
			if len(runner.calls) != 1 || runner.calls[0].Tool != tc.tool || !reflect.DeepEqual(runner.calls[0].Params, want) || !runner.calls[0].DryRun {
				t.Fatalf("calls = %#v, want tool=%q params=%#v dry_run=true", runner.calls, tc.tool, want)
			}
		})
	}
}

func TestDevMCPMemberMutationMethodsValidationParity(t *testing.T) {
	runner := &devMCPToolReadRecordingRunner{}
	root := newDevAppTestRoot(runner)
	root.SetArgs([]string{"dev", "mcp", "member", "add", "--mcp-id", "10520", "--user-ids", " ; , "})
	err := corecmd.ExecuteForTest(root)
	if err == nil || !strings.Contains(err.Error(), "--user-ids 至少包含一个成员 staffId") || len(runner.calls) != 0 {
		t.Fatalf("error = %v, calls = %d", err, len(runner.calls))
	}
}

func buildDevMCPMemberMutationTestCommand(runner executor.Runner, name string) *cobra.Command {
	root := &cobra.Command{Use: "dws"}
	member := newDevMCPTestGroup(root, "member")
	if err := addDevMCPMemberMutationMethods(runner, member); err != nil {
		panic("add member mutation methods: " + err.Error())
	}
	return childCommand(member, name)
}
