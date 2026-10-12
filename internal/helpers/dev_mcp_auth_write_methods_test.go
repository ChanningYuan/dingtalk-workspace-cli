package helpers

import (
	"testing"

	"github.com/DingTalk-Real-AI/dingtalk-workspace-cli/internal/corecmd/contract"
	"github.com/DingTalk-Real-AI/dingtalk-workspace-cli/internal/corecmd/contractfinal"
	"github.com/DingTalk-Real-AI/dingtalk-workspace-cli/internal/corecmd/runtimeannotate"
)

func TestDevMCPAuthConfigSaveMethodContractParity(t *testing.T) {
	cmd := buildDevMCPAuthWriteTestCommand(&captureRunner{})
	final, ok := contractfinal.RuntimeContractFinal(cmd)
	if !ok || final.Identity == nil || final.Safety == nil || final.DryRun == nil {
		t.Fatalf("ContractFinal = %#v, registered = %t", final, ok)
	}
	if final.Identity.ProductID != "dev" || final.Identity.Name != devMCPAuthConfigSaveTool || final.Identity.CanonicalPath != "dev."+devMCPAuthConfigSaveTool || final.Identity.CLIPath != "dev mcp auth save" {
		t.Fatalf("identity = %#v", *final.Identity)
	}
	wantSafety := contract.SafetySpec{Effect: "write", EffectSource: "corecmd.contract", Risk: "medium", Confirmation: "user_required", Idempotency: "unknown"}
	if *final.Safety != wantSafety || final.DryRun.PreviewKind != contract.DryRunPreviewInvocation || final.DryRun.RemoteReads {
		t.Fatalf("contract = safety:%#v dry_run:%#v", final.Safety, final.DryRun)
	}
	for name, property := range map[string]string{
		"mcp-id":                "mcpId",
		"auth-type":             "authType",
		"token-auth-config":     "tokenAuthConfig",
		"signature-auth-config": "signatureAuthConfig",
	} {
		flag := cmd.Flags().Lookup(name)
		if flag == nil || firstFlagAnnotation(flag, runtimeannotate.AnnotationFlagProperty) != property {
			t.Fatalf("--%s metadata = %#v", name, flag)
		}
	}
	if flag := cmd.Flags().Lookup("api-secret-auth-config"); flag == nil || !flag.Hidden {
		t.Fatalf("--api-secret-auth-config metadata = %#v", flag)
	}
}
