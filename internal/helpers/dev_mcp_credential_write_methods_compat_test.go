package helpers

import (
	"github.com/spf13/cobra"

	"github.com/DingTalk-Real-AI/dingtalk-workspace-cli/internal/executor"
)

func buildDevMCPCredentialWriteTestCommand(runner executor.Runner, name string) *cobra.Command {
	root := &cobra.Command{Use: "dws"}
	if err := addDevMCPCredentialWriteMethods(runner, root); err != nil {
		panic("add credential write methods: " + err.Error())
	}
	dev := childCommand(root, "dev")
	mcp := childCommand(dev, "mcp")
	credential := childCommand(mcp, "credential")
	return childCommand(credential, name)
}
