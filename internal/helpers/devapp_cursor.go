// Copyright 2026 Alibaba Group
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package helpers

import (
	"github.com/spf13/cobra"

	"github.com/DingTalk-Real-AI/dingtalk-workspace-cli/internal/executor"
)

// Cursor pagination is declared directly on each devapp list LeafSpec and is
// applied to backend params by devAppCallCursor. PostMount only carries leaf
// metadata.
//
// Semantics stay PASS-THROUGH: the CLI forwards cursor/pageSize into upstream
// params and the upstream returns nextCursor/hasMore. No synthesis, no
// offset/page translation — see docs/cursor-pagination-design.md.

// devAppApplyCursorParams forwards cursor/pageSize into params as-is. cursor is
// only set when non-empty (first page omits it). size < 1 floors to 20.
func devAppApplyCursorParams(cmd *cobra.Command, params map[string]any) {
	if cur := devAppStringFlag(cmd, "cursor"); cur != "" {
		params["cursor"] = cur
	}
	size := devAppIntFlag(cmd, "page-size")
	if size < 1 {
		size = 20
	}
	params["pageSize"] = size
}

// devAppCallCursor is the list execution body: apply cursor params, then run.
func devAppCallCursor(runner executor.Runner) func(*cobra.Command, string, map[string]any) error {
	return func(cmd *cobra.Command, tool string, params map[string]any) error {
		devAppApplyCursorParams(cmd, params)
		return runDevAppTool(runner, cmd, tool, params)
	}
}

// devAppMetaCursor is PostMount for list leaves: surface meta + cursor flags.
func devAppMetaCursor(tool string) func(*cobra.Command) {
	return func(cmd *cobra.Command) {
		devAppLeafMeta(cmd, tool)
	}
}
