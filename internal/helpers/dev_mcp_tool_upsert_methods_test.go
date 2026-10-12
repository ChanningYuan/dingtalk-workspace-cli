package helpers

import (
	"strings"
	"testing"

	"github.com/DingTalk-Real-AI/dingtalk-workspace-cli/internal/corecmd"
	"github.com/DingTalk-Real-AI/dingtalk-workspace-cli/internal/corecmd/contract"
	"github.com/DingTalk-Real-AI/dingtalk-workspace-cli/internal/corecmd/contractfinal"
	"github.com/DingTalk-Real-AI/dingtalk-workspace-cli/internal/corecmd/runtimeannotate"
	"github.com/DingTalk-Real-AI/dingtalk-workspace-cli/internal/executor"
	"github.com/spf13/cobra"
)

func TestDevMCPToolUpsertMethodsContractAndSchemaParity(t *testing.T) {
	cases := []struct {
		name        string
		tool        string
		path        string
		wantVisible int
		wantToolID  bool
	}{
		{name: "create", tool: devMCPToolCreateHTTPTool, path: "dev mcp tool create", wantVisible: 13},
		{name: "update", tool: devMCPToolUpdateHTTPTool, path: "dev mcp tool update", wantVisible: 14, wantToolID: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cmd := buildDevMCPToolUpsertTestCommand(&captureRunner{}, tc.name)
			final, ok := contractfinal.RuntimeContractFinal(cmd)
			if !ok || final.Identity == nil || final.Safety == nil || final.DryRun == nil {
				t.Fatalf("ContractFinal = %#v, registered = %t", final, ok)
			}
			if final.Identity.ProductID != "dev" || final.Identity.Name != tc.tool || final.Identity.CanonicalPath != "dev."+tc.tool || final.Identity.CLIPath != tc.path {
				t.Fatalf("identity = %#v", *final.Identity)
			}
			wantSafety := contract.SafetySpec{Effect: "write", EffectSource: "corecmd.contract", Risk: "medium", Confirmation: "user_required", Idempotency: "unknown"}
			if *final.Safety != wantSafety || final.DryRun.PreviewKind != contract.DryRunPreviewInvocation || final.DryRun.RemoteReads {
				t.Fatalf("contract = safety:%#v dry_run:%#v", final.Safety, final.DryRun)
			}
			if got := visibleFlagCount(cmd); got != tc.wantVisible {
				t.Fatalf("visible flags = %d, want %d", got, tc.wantVisible)
			}
			for _, want := range []struct {
				name     string
				property string
				typeName string
			}{
				{name: "mcp-id", property: "mcpId", typeName: "integer"},
				{name: "name", property: "name", typeName: "string"},
				{name: "http-info", property: "httpInfo", typeName: "string"},
				{name: "api-outputs", property: "apiOutputs", typeName: "string"},
				{name: "tool-outputs", property: "toolOutputs", typeName: "string"},
				{name: "output-mappings", property: "outputMappings", typeName: "string"},
				{name: "timeout", property: "timeout", typeName: "integer"},
				{name: "only-original-keys", property: "onlyOriginalKeys", typeName: "boolean"},
			} {
				flag := cmd.Flags().Lookup(want.name)
				if flag == nil || firstFlagAnnotation(flag, runtimeannotate.AnnotationFlagProperty) != want.property || firstFlagAnnotation(flag, runtimeannotate.AnnotationFlagType) != want.typeName {
					t.Fatalf("--%s metadata = %#v", want.name, flag)
				}
			}
			if tc.wantToolID {
				if flag := cmd.Flags().Lookup("tool-id"); flag == nil || firstFlagAnnotation(flag, runtimeannotate.AnnotationFlagProperty) != "toolId" {
					t.Fatalf("--tool-id metadata = %#v", flag)
				}
				if flag := cmd.Flags().Lookup("action-id"); flag == nil || !flag.Hidden {
					t.Fatalf("--action-id = %#v, want hidden compatibility flag", flag)
				}
			}
			if flag := cmd.Flags().Lookup("http"); flag == nil || !flag.Hidden {
				t.Fatalf("--http = %#v, want hidden compatibility flag", flag)
			}
		})
	}
}

func TestDevMCPToolUpsertMethodsValidationRunsBeforeConfirmation(t *testing.T) {
	runner := &devMCPToolReadRecordingRunner{}
	root := newDevAppTestRoot(runner)
	root.SetIn(strings.NewReader(""))
	root.SetArgs([]string{"dev", "mcp", "tool", "create", "--mcp-id", "10487", "--name", "get_weather", "--http-info", "not-json"})
	err := corecmd.ExecuteForTest(root)
	if err == nil || strings.Contains(err.Error(), "需要用户确认") || len(runner.calls) != 0 {
		t.Fatalf("error = %v, calls = %d; want validation before confirmation", err, len(runner.calls))
	}
}

func buildDevMCPToolUpsertTestCommand(runner executor.Runner, name string) *cobra.Command {
	root := &cobra.Command{Use: "dws"}
	tool := newDevMCPTestGroup(root, "tool")
	if err := addDevMCPToolUpsertMethods(runner, tool); err != nil {
		panic("add tool upsert methods: " + err.Error())
	}
	return childCommand(tool, name)
}
