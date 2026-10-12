package helpers

import (
	"github.com/spf13/cobra"

	"github.com/DingTalk-Real-AI/dingtalk-workspace-cli/internal/executor"
)

// The old coverage tests exercise isolated leaf factories directly. Keep those
// test seams without restoring production hand-written service constructors:
// each helper builds the same ServiceBuilder tree and selects its live leaf.
func newDevMCPTestGroup(root *cobra.Command, name string) *cobra.Command {
	dev := &cobra.Command{Use: "dev"}
	newGroupCommand(dev)
	mcp := &cobra.Command{Use: "mcp"}
	newGroupCommand(mcp)
	group := &cobra.Command{Use: name}
	newGroupCommand(group)
	root.AddCommand(dev)
	dev.AddCommand(mcp)
	mcp.AddCommand(group)
	return group
}

func buildDevMCPServiceTestCommand(runner executor.Runner, name string) *cobra.Command {
	root := &cobra.Command{Use: "dws"}
	if err := addDevMCPServiceMethods(runner, root); err != nil {
		panic("add service methods: " + err.Error())
	}
	dev := childCommand(root, "dev")
	mcp := childCommand(dev, "mcp")
	service := childCommand(mcp, "service")
	return childCommand(service, name)
}

func childCommand(parent *cobra.Command, name string) *cobra.Command {
	for _, child := range parent.Commands() {
		if child.Name() == name {
			return child
		}
	}
	panic("missing test command " + name)
}
