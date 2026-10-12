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

func TestDevMCPCredentialReadMethodsContractParity(t *testing.T) {
	for _, tc := range []struct {
		name string
		tool string
		path string
	}{
		{name: "list", tool: devMCPCredentialListTool, path: "dev mcp credential list"},
		{name: "get", tool: devMCPCredentialGetTool, path: "dev mcp credential get"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cmd := buildDevMCPCredentialReadTestCommand(&captureRunner{}, tc.name)
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
			for _, name := range []string{"mcp-id"} {
				flag := cmd.Flags().Lookup(name)
				if flag == nil || firstFlagAnnotation(flag, runtimeannotate.AnnotationFlagProperty) != "mcpId" || firstFlagAnnotation(flag, runtimeannotate.AnnotationFlagType) != "integer" {
					t.Fatalf("--%s metadata = %#v", name, flag)
				}
			}
			if tc.name == "list" {
				for name, property := range map[string]string{"cursor": "cursor", "page-size": "pageSize"} {
					flag := cmd.Flags().Lookup(name)
					if flag == nil || firstFlagAnnotation(flag, runtimeannotate.AnnotationFlagProperty) != property {
						t.Fatalf("--%s metadata = %#v", name, flag)
					}
				}
			} else {
				flag := cmd.Flags().Lookup("credential-id")
				if flag == nil || firstFlagAnnotation(flag, runtimeannotate.AnnotationFlagProperty) != "credentialId" {
					t.Fatalf("--credential-id metadata = %#v", flag)
				}
			}
		})
	}
}

func TestDevMCPCredentialReadMethodsBackendParity(t *testing.T) {
	cases := []struct {
		name   string
		args   []string
		tool   string
		params map[string]any
	}{
		{
			name:   "list",
			args:   []string{"dev", "mcp", "credential", "list", "--mcp-id", "10520", "--cursor", "2", "--page-size", "20"},
			tool:   devMCPCredentialListTool,
			params: map[string]any{"mcpId": 10520, "cursor": 2, "pageSize": 20},
		},
		{
			name:   "list omits empty paging",
			args:   []string{"dev", "mcp", "credential", "list", "--mcp-id", "10520"},
			tool:   devMCPCredentialListTool,
			params: map[string]any{"mcpId": 10520},
		},
		{
			name:   "get",
			args:   []string{"dev", "mcp", "credential", "get", "--mcp-id", "10520", "--credential-id", "10001", "--dry-run"},
			tool:   devMCPCredentialGetTool,
			params: map[string]any{"mcpId": 10520, "credentialId": 10001},
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
			if len(runner.calls) != 1 || runner.calls[0].Tool != tc.tool || !reflect.DeepEqual(runner.calls[0].Params, tc.params) {
				t.Fatalf("calls = %#v", runner.calls)
			}
		})
	}
}

func buildDevMCPCredentialReadTestCommand(runner executor.Runner, name string) *cobra.Command {
	root := &cobra.Command{Use: "dws"}
	credential := newDevMCPTestGroup(root, "credential")
	if err := addDevMCPCredentialReadMethods(runner, credential); err != nil {
		panic("add credential read methods: " + err.Error())
	}
	return childCommand(credential, name)
}
