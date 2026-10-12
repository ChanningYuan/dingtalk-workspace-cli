package helpers

import (
	"github.com/spf13/cobra"

	"github.com/DingTalk-Real-AI/dingtalk-workspace-cli/internal/executor"
)

func buildDevMCPToolLifecycleTestCommand(runner executor.Runner, name string) *cobra.Command {
	root := &cobra.Command{Use: "dws"}
	if err := addDevMCPToolLifecycleMethods(runner, root); err != nil {
		panic("add tool lifecycle methods: " + err.Error())
	}
	dev := childCommand(root, "dev")
	mcp := childCommand(dev, "mcp")
	tool := childCommand(mcp, "tool")
	return childCommand(tool, name)
}
