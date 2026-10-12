package helpers

import (
	"github.com/spf13/cobra"

	"github.com/DingTalk-Real-AI/dingtalk-workspace-cli/internal/executor"
)

func buildDevMCPHsfMethodTestCommand(runner executor.Runner, name string) *cobra.Command {
	root := &cobra.Command{Use: "dws"}
	if err := addDevMCPHsfMethodListMethod(runner, root); err != nil {
		panic("add hsf method-list: " + err.Error())
	}
	dev := childCommand(root, "dev")
	mcp := childCommand(dev, "mcp")
	hsf := childCommand(mcp, "hsf")
	return childCommand(hsf, name)
}

func buildDevMCPToolHsfMethodTestCommand(runner executor.Runner, name string) *cobra.Command {
	root := &cobra.Command{Use: "dws"}
	if err := addDevMCPToolHsfMethods(runner, root); err != nil {
		panic("add tool hsf methods: " + err.Error())
	}
	dev := childCommand(root, "dev")
	mcp := childCommand(dev, "mcp")
	tool := childCommand(mcp, "tool")
	return childCommand(tool, name)
}
