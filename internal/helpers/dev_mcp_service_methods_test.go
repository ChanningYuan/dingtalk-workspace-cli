package helpers

import (
	"testing"

	"github.com/spf13/cobra"

	"github.com/DingTalk-Real-AI/dingtalk-workspace-cli/internal/executor"
)

func TestDevMCPServiceMethodsBuildFiveRunnableLeaves(t *testing.T) {
	root := &cobra.Command{Use: "dws"}
	service := newDevMCPTestGroup(root, "service")

	if err := addDevMCPServiceMethods(&captureRunner{}, service); err != nil {
		t.Fatalf("addDevMCPServiceMethods: %v", err)
	}
	want := []string{"list", "get", "create", "update", "delete"}
	if got := service.Commands(); len(got) != len(want) {
		t.Fatalf("service leaves = %d, want %d", len(got), len(want))
	}
	for _, name := range want {
		cmd := childCommand(service, name)
		if cmd.RunE == nil {
			t.Fatalf("%s has no runnable RunE", name)
		}
		if cmd.Annotations["mcp-tool"] == "" || cmd.Annotations["mcp-source"] != devMCPProduct {
			t.Fatalf("%s missing backend annotations: %#v", name, cmd.Annotations)
		}
		if cmd.Args == nil {
			t.Fatalf("%s missing Args policy", name)
		}
	}
}

func TestDevMCPServiceMethodsPreserveDeclaredFlags(t *testing.T) {
	root := &cobra.Command{Use: "dws"}
	if err := addDevMCPServiceMethods(&captureRunner{}, root); err != nil {
		t.Fatal(err)
	}
	get := childCommand(childCommand(childCommand(root, "dev"), "mcp"), "service")
	get = childCommand(get, "get")
	flag := get.Flags().Lookup("mcp-id")
	if flag == nil || flag.Value.Type() != "int" {
		t.Fatalf("mcp-id flag = %#v, want int flag", flag)
	}
	if flag.DefValue != "0" {
		t.Fatalf("mcp-id default = %q, want 0", flag.DefValue)
	}

	create := childCommand(childCommand(childCommand(root, "dev"), "mcp"), "service")
	create = childCommand(create, "create")
	for _, name := range []string{"name", "description", "icon-url", "introduction", "server-name"} {
		if create.Flags().Lookup(name) == nil {
			t.Fatalf("create missing --%s", name)
		}
	}
}

func TestDevMCPServiceMethodsExecuteProviderValidationBeforeRunner(t *testing.T) {
	runner := &captureRunner{}
	cmd := buildDevMCPServiceTestCommand(runner, "create")
	_ = cmd.Flags().Set("name", "x")
	_ = cmd.Flags().Set("description", "y")
	_ = cmd.Flags().Set("server-name", "bad_name")
	if err := cmd.RunE(cmd, nil); err == nil {
		t.Fatal("invalid server-name should fail")
	}
	if runner.last.Tool != "" {
		t.Fatalf("runner received tool %q, want no invocation", runner.last.Tool)
	}
}

var _ executor.Runner = (*captureRunner)(nil)
