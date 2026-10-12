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

func TestDevMCPMemberListMethodParity(t *testing.T) {
	cmd := buildDevMCPMemberListTestCommand(&captureRunner{})
	final, ok := contractfinal.RuntimeContractFinal(cmd)
	if !ok || final.Identity == nil || final.Safety == nil {
		t.Fatalf("ContractFinal = %#v, registered = %t", final, ok)
	}
	if final.Identity.ProductID != "dev" || final.Identity.Name != devMCPMemberListTool || final.Identity.CanonicalPath != "dev."+devMCPMemberListTool || final.Identity.CLIPath != "dev mcp member list" {
		t.Fatalf("identity = %#v", *final.Identity)
	}
	wantSafety := contract.SafetySpec{Effect: "read", EffectSource: "corecmd.contract", Risk: "medium", Confirmation: "not_required", Idempotency: "idempotent"}
	if *final.Safety != wantSafety || final.DryRun != nil {
		t.Fatalf("contract = safety:%#v dry_run:%#v", final.Safety, final.DryRun)
	}
	flag := cmd.Flags().Lookup("mcp-id")
	if flag == nil || firstFlagAnnotation(flag, runtimeannotate.AnnotationFlagProperty) != "mcpId" || firstFlagAnnotation(flag, runtimeannotate.AnnotationFlagType) != "integer" {
		t.Fatalf("--mcp-id metadata = %#v", flag)
	}

	runner := &devMCPToolReadRecordingRunner{}
	root := newDevAppTestRoot(runner)
	root.SetArgs([]string{"dev", "mcp", "member", "list", "--mcp-id", "10520", "--dry-run"})
	if err := corecmd.ExecuteForTest(root); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if len(runner.calls) != 1 || runner.calls[0].Tool != devMCPMemberListTool || !reflect.DeepEqual(runner.calls[0].Params, map[string]any{"mcpId": 10520}) || !runner.calls[0].DryRun {
		t.Fatalf("calls = %#v", runner.calls)
	}
}

func buildDevMCPMemberListTestCommand(runner executor.Runner) *cobra.Command {
	root := &cobra.Command{Use: "dws"}
	member := newDevMCPTestGroup(root, "member")
	if err := addDevMCPMemberListMethod(runner, member); err != nil {
		panic("add member list method: " + err.Error())
	}
	return childCommand(member, "list")
}
