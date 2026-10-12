package contract

import (
	"strings"
	"testing"
)

func TestValidateToolIdentity(t *testing.T) {
	for _, value := range []string{"dev", "aitable-helper", "mcp_service_get", "v2"} {
		if err := ValidateToolIdentityPart("identity", value); err != nil {
			t.Fatalf("valid identity %q: %v", value, err)
		}
	}
	for _, value := range []string{"", "  ", "1dev", ".dev", "dev.name", "dev name", "dev!"} {
		if err := ValidateToolIdentityPart("identity", value); err == nil {
			t.Fatalf("invalid identity accepted: %q", value)
		}
	}
	if err := ValidateToolIdentity("dev", "mcp_service_get"); err != nil {
		t.Fatal(err)
	}
	if err := ValidateToolIdentity("dev.name", "get"); err == nil || !strings.Contains(err.Error(), "ProductID") {
		t.Fatalf("invalid product identity error = %v", err)
	}
}
