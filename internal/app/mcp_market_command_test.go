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

package app

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"testing"

	"github.com/DingTalk-Real-AI/dingtalk-workspace-cli/internal/corecmd"
	apperrors "github.com/DingTalk-Real-AI/dingtalk-workspace-cli/internal/errors"
	"github.com/DingTalk-Real-AI/dingtalk-workspace-cli/internal/output"
	"github.com/DingTalk-Real-AI/dingtalk-workspace-cli/pkg/edition"
	"github.com/spf13/cobra"
)

type mcpMarketSearchTestCaller struct {
	responses []string
	productID string
	toolName  string
	calls     []map[string]any
	err       error
	dryRun    bool
}

func (c *mcpMarketSearchTestCaller) CallTool(_ context.Context, productID, toolName string, args map[string]any) (*edition.ToolResult, error) {
	c.productID = productID
	c.toolName = toolName
	c.calls = append(c.calls, args)
	if c.err != nil {
		return nil, c.err
	}
	i := len(c.calls) - 1
	if i >= len(c.responses) {
		i = len(c.responses) - 1
	}
	return &edition.ToolResult{Content: []edition.ContentBlock{{Type: "text", Text: c.responses[i]}}}, nil
}

func (*mcpMarketSearchTestCaller) Format() string { return "json" }
func (c *mcpMarketSearchTestCaller) DryRun() bool { return c.dryRun }
func (*mcpMarketSearchTestCaller) Fields() string { return "" }
func (*mcpMarketSearchTestCaller) JQ() string     { return "" }

// mcpMarketSearchPage builds one search_market_mcps response. totalCount is
// kept deliberately inconsistent with the items, as observed on the live
// endpoint.
func mcpMarketSearchPage(ids ...int) string {
	values := make([]map[string]any, 0, len(ids))
	for _, id := range ids {
		values = append(values, map[string]any{"mcpId": id, "name": fmt.Sprintf("MCP %d", id)})
	}
	raw, _ := json.Marshal(map[string]any{
		"success": true,
		"result":  map[string]any{"values": values, "totalCount": 1},
	})
	return string(raw)
}

func executeMCPMarketSearchCommand(t *testing.T, caller edition.ToolCaller, args ...string) (map[string]any, string, error) {
	t.Helper()
	ctx, _ := output.WithResultStore(t.Context())
	root := &cobra.Command{Use: "mcp", SilenceErrors: true, SilenceUsage: true}
	root.PersistentFlags().String("format", "json", "")
	root.PersistentPostRunE = func(executed *cobra.Command, _ []string) error {
		_, _, err := output.EmitStoredResult(executed)
		return err
	}
	root.AddCommand(newMCPMarketGroup(caller))
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs(append([]string{"market", "search"}, args...))
	err := corecmd.ExecuteContextForTest(root, ctx)
	var envelope map[string]any
	_ = json.Unmarshal(out.Bytes(), &envelope)
	return envelope, out.String(), err
}

func mcpMarketSearchPagination(t *testing.T, envelope map[string]any, raw string) map[string]any {
	t.Helper()
	meta, _ := envelope["meta"].(map[string]any)
	pagination, ok := meta["pagination"].(map[string]any)
	if !ok {
		t.Fatalf("meta.pagination missing:\n%s", raw)
	}
	return pagination
}

func mcpMarketSearchItems(t *testing.T, envelope map[string]any, raw string) []any {
	t.Helper()
	data, _ := envelope["data"].(map[string]any)
	items, ok := data["mcps"].([]any)
	if !ok {
		t.Fatalf("data.mcps missing:\n%s", raw)
	}
	return items
}

func TestMCPMarketSearchCallsMetaServerWithPageArgs(t *testing.T) {
	caller := &mcpMarketSearchTestCaller{responses: []string{mcpMarketSearchPage(9595, 1024)}}

	envelope, raw, err := executeMCPMarketSearchCommand(t, caller, "--keyword", " 机器人 ", "--limit", "2")
	if err != nil {
		t.Fatalf("execute mcp market search: %v\n%s", err, raw)
	}
	if caller.productID != mcpMetaServerID || caller.toolName != mcpMetaMarketSearchTool {
		t.Fatalf("call routed to %s/%s, want %s/%s", caller.productID, caller.toolName, mcpMetaServerID, mcpMetaMarketSearchTool)
	}
	if len(caller.calls) != 1 {
		t.Fatalf("calls = %d, want 1", len(caller.calls))
	}
	call := caller.calls[0]
	if call["search"] != "机器人" || call["page"] != 1 || call["pageSize"] != 2 {
		t.Fatalf("call args = %#v", call)
	}
	items := mcpMarketSearchItems(t, envelope, raw)
	if got := len(items); got != 2 {
		t.Fatalf("items = %d, want 2", got)
	}
	first, _ := items[0].(map[string]any)
	if value, present := first["serverName"]; !present || value != nil {
		t.Fatalf("serverName must be present as null when the endpoint omits it: %#v", first)
	}
	data, _ := envelope["data"].(map[string]any)
	if data["keyword"] != "机器人" {
		t.Fatalf("data.keyword = %#v", data["keyword"])
	}
	pagination := mcpMarketSearchPagination(t, envelope, raw)
	if pagination["next_token"] != "2" || pagination["endpoint_exhausted"] == true {
		t.Fatalf("pagination = %#v, want next_token 2 and not exhausted", pagination)
	}
	for _, key := range []string{"pages", "items"} {
		if _, present := pagination[key]; present {
			t.Fatalf("pagination.%s must not be published for a single-page command: %#v", key, pagination)
		}
	}
}

func TestMCPMarketSearchProjectsReviewedFields(t *testing.T) {
	response := `{"success":true,"result":{"values":[{"mcpId":9595,"name":"机器人消息","description":"机器人消息","serverName":"bot","official":true,"corpName":"钉钉（中国）信息技术有限公司","charged":false,"installCount":283898,"highQuality":true,"sourceType":"DINGTALK","icon":"https://example.test/icon.png"}]}}`
	caller := &mcpMarketSearchTestCaller{responses: []string{response}}

	envelope, raw, err := executeMCPMarketSearchCommand(t, caller, "--keyword", "机器人")
	if err != nil {
		t.Fatalf("execute mcp market search: %v\n%s", err, raw)
	}
	items := mcpMarketSearchItems(t, envelope, raw)
	item, _ := items[0].(map[string]any)
	got := make([]string, 0, len(item))
	for key := range item {
		got = append(got, key)
	}
	sort.Strings(got)
	want := append([]string(nil), mcpMarketSearchFields...)
	sort.Strings(want)
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("item fields = %v, want %v", got, want)
	}
	if item["serverName"] != "bot" || item["official"] != true {
		t.Fatalf("projected values changed: %#v", item)
	}
}

func TestMCPMarketSearchShortPageIsExhausted(t *testing.T) {
	caller := &mcpMarketSearchTestCaller{responses: []string{mcpMarketSearchPage(2013)}}

	envelope, raw, err := executeMCPMarketSearchCommand(t, caller, "--keyword", "Teambition", "--limit", "2")
	if err != nil {
		t.Fatalf("execute mcp market search: %v\n%s", err, raw)
	}
	pagination := mcpMarketSearchPagination(t, envelope, raw)
	if pagination["endpoint_exhausted"] != true {
		t.Fatalf("pagination = %#v, want exhausted", pagination)
	}
	if token, _ := pagination["next_token"].(string); token != "" {
		t.Fatalf("next_token = %q, want empty", token)
	}
}

func TestMCPMarketSearchCursorSelectsPage(t *testing.T) {
	caller := &mcpMarketSearchTestCaller{responses: []string{mcpMarketSearchPage()}}

	_, raw, err := executeMCPMarketSearchCommand(t, caller, "--keyword", "机器人", "--cursor", "3")
	if err != nil {
		t.Fatalf("execute mcp market search: %v\n%s", err, raw)
	}
	if got := caller.calls[0]["page"]; got != 3 {
		t.Fatalf("page = %#v, want 3", got)
	}
	if got := caller.calls[0]["pageSize"]; got != mcpMarketSearchDefaultLimit {
		t.Fatalf("pageSize = %#v, want default %d", got, mcpMarketSearchDefaultLimit)
	}
}

func TestMCPMarketSearchRejectsInvalidFlags(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{name: "missing keyword", args: []string{"--limit", "5"}, want: "keyword"},
		{name: "limit too small", args: []string{"--keyword", "x", "--limit", "0"}, want: "--limit"},
		{name: "limit too large", args: []string{"--keyword", "x", "--limit", "101"}, want: "--limit"},
		{name: "non-numeric cursor", args: []string{"--keyword", "x", "--cursor", "abc"}, want: "--cursor"},
		{name: "zero cursor", args: []string{"--keyword", "x", "--cursor", "0"}, want: "--cursor"},
		{name: "page-all is not supported", args: []string{"--keyword", "x", "--page-all"}, want: "page-all"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			caller := &mcpMarketSearchTestCaller{responses: []string{mcpMarketSearchPage()}}
			_, raw, err := executeMCPMarketSearchCommand(t, caller, tt.args...)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %v, want mention of %q\n%s", err, tt.want, raw)
			}
			if len(caller.calls) != 0 {
				t.Fatalf("invalid flags must not call the endpoint, got %d calls", len(caller.calls))
			}
		})
	}
}

func TestMCPMarketSearchClassifiesBusinessError(t *testing.T) {
	caller := &mcpMarketSearchTestCaller{responses: []string{`{"success":false,"errorMsg":"搜索失败"}`}}

	_, _, err := executeMCPMarketSearchCommand(t, caller, "--keyword", "x")
	var typed *apperrors.Error
	if !errors.As(err, &typed) || typed.Reason != "business_error" {
		t.Fatalf("error = %#v, want classified business error", err)
	}
}

func TestMCPMarketSearchRejectsMalformedResults(t *testing.T) {
	tests := []struct {
		name     string
		response string
		want     string
	}{
		{name: "missing values", response: `{"success":true,"result":{}}`, want: "result.values"},
		{name: "item without mcpId", response: `{"success":true,"result":{"values":[{"name":"x"}]}}`, want: "mcpId"},
		{name: "invalid json", response: "not-json", want: "无效 JSON"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			caller := &mcpMarketSearchTestCaller{responses: []string{tt.response}}
			_, _, err := executeMCPMarketSearchCommand(t, caller, "--keyword", "x")
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %v, want %q", err, tt.want)
			}
		})
	}
}

func TestMCPMarketSearchPropagatesCallError(t *testing.T) {
	caller := &mcpMarketSearchTestCaller{err: errors.New("permission denied")}

	_, _, err := executeMCPMarketSearchCommand(t, caller, "--keyword", "x")
	if err == nil || !strings.Contains(err.Error(), "permission denied") {
		t.Fatalf("error = %v, want call error", err)
	}
}

func TestMCPMarketSearchDryRunDoesNotCallEndpoint(t *testing.T) {
	caller := &mcpMarketSearchTestCaller{dryRun: true}

	envelope, raw, err := executeMCPMarketSearchCommand(t, caller, "--keyword", "机器人", "--cursor", "2")
	if err != nil {
		t.Fatalf("execute mcp market search dry-run: %v\n%s", err, raw)
	}
	if len(caller.calls) != 0 {
		t.Fatalf("dry-run called the endpoint %d times", len(caller.calls))
	}
	data, _ := envelope["data"].(map[string]any)
	if data["tool"] != mcpMetaMarketSearchTool {
		t.Fatalf("dry-run data = %#v", data)
	}
}

func TestMCPMarketSearchRejectsMissingCaller(t *testing.T) {
	_, _, err := executeMCPMarketSearchCommand(t, nil, "--keyword", "x")
	if err == nil || !strings.Contains(err.Error(), "caller is not configured") {
		t.Fatalf("error = %v, want missing caller error", err)
	}
}

func TestRootRegistersMCPMarketSearch(t *testing.T) {
	root := NewRootCommand(t.Context())
	cmd, _, err := root.Find([]string{"mcp", "market", "search"})
	if err != nil {
		t.Fatalf("find mcp market search: %v", err)
	}
	if got := cmd.CommandPath(); got != "dws mcp market search" {
		t.Fatalf("command path = %q, want %q", got, "dws mcp market search")
	}
}

func TestMCPMarketGroupShowsHelp(t *testing.T) {
	root := &cobra.Command{Use: "mcp", SilenceErrors: true, SilenceUsage: true}
	root.AddCommand(newMCPMarketGroup(nil))
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"market"})
	if err := corecmd.ExecuteContextForTest(root, t.Context()); err != nil {
		t.Fatalf("execute mcp market: %v", err)
	}
	if !strings.Contains(out.String(), "search") {
		t.Fatalf("help output does not list search command:\n%s", out.String())
	}
}
