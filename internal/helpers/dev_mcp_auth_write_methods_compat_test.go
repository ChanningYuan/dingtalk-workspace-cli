package helpers

import (
	"github.com/spf13/cobra"

	"github.com/DingTalk-Real-AI/dingtalk-workspace-cli/internal/executor"
)

func buildDevMCPAuthWriteTestCommand(runner executor.Runner) *cobra.Command {
	root := &cobra.Command{Use: "dws"}
	if err := addDevMCPAuthConfigSaveMethod(runner, root); err != nil {
		panic("add auth write method: " + err.Error())
	}
	dev := childCommand(root, "dev")
	mcp := childCommand(dev, "mcp")
	auth := childCommand(mcp, "auth")
	return childCommand(auth, "save")
}
