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

package corecmd

import (
	"strings"
	"testing"

	"github.com/DingTalk-Real-AI/dingtalk-workspace-cli/internal/corecmd/contract"
	"github.com/spf13/cobra"
)

func serviceMethodDef(cliPath string) MethodDefinition {
	return MethodDefinition{
		Identity: MethodIdentity{
			CLIPath:     cliPath,
			ProductID:   "dev",
			Name:        "mcp_service_list",
			Description: "查询当前用户有开发权限的 MCP 服务",
			Example:     "  dws dev mcp service list --page-size 20 --format json",
		},
		Input: InputSpec{
			Parameters: []ParameterSpec{
				{Name: "keyword", Type: KindString, Usage: "按服务名关键词过滤", Bind: "keyword"},
				{Name: "page-size", Type: KindInt, Usage: "每页条数，最大 100", Bind: "pageSize"},
			},
		},
		Contract: ContractSpec{
			Selection: contract.SelectionSpec{
				AgentSummary: "查询当前用户有开发权限的 MCP 服务",
				UseWhen:      []string{"查询当前用户有开发权限的 MCP 服务"},
				AvoidWhen:    []string{"只需调用已发布 MCP 工具时使用 dws mcp published"},
				Examples:     []string{"dws dev mcp service list"},
			},
			Interface: &contract.InterfaceSpec{
				Mode:         contract.InterfaceModeComposite,
				Availability: contract.InterfaceAvailable,
				Reason:       "Reviewed unpinned adapter.",
			},
		},
		Safety: contract.SafetySpec{
			Effect: "read", Risk: "medium",
			Confirmation: "not_required", Idempotency: "idempotent",
		},
	}
}

// TestServiceBuilderCompilesMethodToCorecmdSpec verifies the compile mapping:
// identity → ContractDecl identity, parameters → FlagSpec bindings, and the
// ToolBinding default executor path.
func TestServiceBuilderCompilesMethodToCorecmdSpec(t *testing.T) {
	var invokedToolArgs map[string]any
	binding := ToolBinding("mcp_service_list", func(c *Ctx, toolArgs map[string]any) error {
		invokedToolArgs = toolArgs
		return nil
	})
	spec, err := compileMethod(serviceMethodDef("dev mcp service list"), binding)
	if err != nil {
		t.Fatalf("compileMethod: %v", err)
	}
	if spec.Use != "list" {
		t.Fatalf("Use = %q, want list", spec.Use)
	}
	if len(spec.Flags) != 2 {
		t.Fatalf("flags = %d, want 2", len(spec.Flags))
	}
	if spec.Flags[1].Bind != "pageSize" {
		t.Fatalf("page-size Bind = %q, want pageSize", spec.Flags[1].Bind)
	}
	if spec.Contract.Identity.CanonicalPath != "dev.mcp_service_list" {
		t.Fatalf("CanonicalPath = %q", spec.Contract.Identity.CanonicalPath)
	}
	if spec.Contract.Identity.CLIPath != "dev mcp service list" {
		t.Fatalf("CLIPath = %q", spec.Contract.Identity.CLIPath)
	}
	if spec.Invoke == nil {
		t.Fatal("ToolBinding must install Invoke")
	}
	if spec.RunE != nil || spec.Orchestrate != nil || spec.ResultInvoke != nil {
		t.Fatal("ToolBinding must not set RunE/Orchestrate/ResultInvoke")
	}
	_ = invokedToolArgs
}

// TestServiceBuilderBindingExactlyOneOwner covers the fail-closed binding
// cardinality: zero owners, two owners, and a bare Tool without a provider
// executor are all rejected.
func TestServiceBuilderBindingExactlyOneOwner(t *testing.T) {
	def := serviceMethodDef("dev mcp service list")
	invoker := func(c *Ctx, toolArgs map[string]any) error { return nil }
	orchestrator := func(c *Ctx) error { return nil }

	cases := []struct {
		name    string
		binding MethodBinding
		wantErr string
	}{
		{"zero owners", MethodBinding{}, "exactly one of Tool/Invoke/Orchestrate"},
		{"orchestrate plus invoke", MethodBinding{Invoke: invoker, Orchestrate: orchestrator}, "cannot be combined"},
		{"orchestrate plus tool", MethodBinding{Tool: "t", Orchestrate: orchestrator}, "Orchestrate cannot be combined"},
		{"bare tool", MethodBinding{Tool: "t"}, "requires ToolExecutor"},
	}
	for _, tc := range cases {
		_, err := compileMethod(def, tc.binding)
		if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
			t.Fatalf("%s: err = %v, want %q", tc.name, err, tc.wantErr)
		}
	}
	if _, err := compileMethod(def, MethodBinding{Invoke: invoker}); err != nil {
		t.Fatalf("Invoke-only binding rejected: %v", err)
	}
	if _, err := compileMethod(def, MethodBinding{Orchestrate: orchestrator}); err != nil {
		t.Fatalf("Orchestrate-only binding rejected: %v", err)
	}
}

// TestServiceBuilderDefinitionValidation covers declaration completeness:
// empty CLIPath, single-segment path, and missing identity fields.
func TestServiceBuilderToolBindingNameMismatchRejected(t *testing.T) {
	root := newTestRoot("dws")
	builder, err := NewServiceBuilder(root)
	if err != nil {
		t.Fatal(err)
	}
	def := serviceMethodDef("dev mcp service list")
	binding := MethodBinding{
		Tool: "mcp_service_get",
		ToolExecutor: func(*Ctx, map[string]any) error {
			return nil
		},
	}
	if err := builder.Add(def, binding); err == nil ||
		!strings.Contains(err.Error(), "does not match Identity.Name") ||
		!strings.Contains(err.Error(), "mcp_service_get") ||
		!strings.Contains(err.Error(), "mcp_service_list") {
		t.Fatalf("mismatch error = %v", err)
	}
	if findChild(root, "dev") != nil {
		t.Fatal("mismatched Tool binding mounted a command tree")
	}
}

func TestServiceBuilderAddRejectsNilReceiver(t *testing.T) {
	def := serviceMethodDef("dev mcp service list")
	err := (*ServiceBuilder)(nil).Add(def, MethodBinding{Invoke: func(*Ctx, map[string]any) error { return nil }})
	if err == nil || !strings.Contains(err.Error(), "builder is nil") {
		t.Fatalf("nil builder error = %v", err)
	}
}

func TestServiceBuilderDefinitionValidation(t *testing.T) {
	invoker := func(c *Ctx, toolArgs map[string]any) error { return nil }
	cases := []struct {
		name   string
		mutate func(*MethodDefinition)
		want   string
	}{
		{"empty cli path", func(d *MethodDefinition) { d.Identity.CLIPath = "" }, "CLIPath is empty"},
		{"one segment", func(d *MethodDefinition) { d.Identity.CLIPath = "list" }, "service group and a leaf segment"},
		{"no product", func(d *MethodDefinition) { d.Identity.ProductID = "" }, "ProductID is empty"},
		{"no name", func(d *MethodDefinition) { d.Identity.Name = "" }, "Name is empty"},
		{"invalid product", func(d *MethodDefinition) { d.Identity.ProductID = "dev.product" }, "invalid identity"},
		{"invalid name", func(d *MethodDefinition) { d.Identity.Name = "mcp.service.get" }, "invalid identity"},
		{"no description", func(d *MethodDefinition) { d.Identity.Description = "" }, "Description is empty"},
	}
	for _, tc := range cases {
		def := serviceMethodDef("dev mcp service list")
		tc.mutate(&def)
		if _, err := compileMethod(def, MethodBinding{Invoke: invoker}); err == nil ||
			!strings.Contains(err.Error(), tc.want) {
			t.Fatalf("%s: err = %v, want %q", tc.name, err, tc.want)
		}
	}
}

// TestServiceBuilderDuplicatePathRejected mounts two methods with the same
// CLIPath under one builder.
func TestServiceBuilderDuplicatePathRejected(t *testing.T) {
	b, err := NewServiceBuilder(newTestRoot("dws"))
	if err != nil {
		t.Fatal(err)
	}
	invoker := func(c *Ctx, toolArgs map[string]any) error { return nil }
	if err := b.Add(serviceMethodDef("dev mcp service list"), MethodBinding{Invoke: invoker}); err != nil {
		t.Fatalf("first Add: %v", err)
	}
	err = b.Add(serviceMethodDef("dev mcp service get"), MethodBinding{Invoke: invoker})
	if err != nil {
		t.Fatalf("second Add (different path): %v", err)
	}
	err = b.Add(serviceMethodDef("dev mcp service list"), MethodBinding{Invoke: invoker})
	if err == nil || !strings.Contains(err.Error(), "duplicate CLIPath") {
		t.Fatalf("duplicate path err = %v", err)
	}
}

// TestServiceBuilderCreatesGroupsWithPolicy verifies intermediate groups are
// created with the navigation-only GroupPolicy and that a pre-existing group
// is reused rather than duplicated.
func TestServiceBuilderAttachedParentUsesExactAncestry(t *testing.T) {
	root := newTestRoot("dws")
	dev := &cobra.Command{Use: "dev"}
	mcp := &cobra.Command{Use: "mcp"}
	service := &cobra.Command{Use: "service"}
	root.AddCommand(dev)
	dev.AddCommand(mcp)
	mcp.AddCommand(service)

	builder, err := NewServiceBuilder(service)
	if err != nil {
		t.Fatal(err)
	}
	if err := builder.Add(serviceMethodDef("dev mcp service list"), MethodBinding{
		Invoke: func(*Ctx, map[string]any) error { return nil },
	}); err != nil {
		t.Fatal(err)
	}
	if findChild(service, "list") == nil {
		t.Fatal("list leaf missing from attached service parent")
	}
	if findChild(service, "dev") != nil {
		t.Fatal("attached parent received a duplicated CLIPath subtree")
	}
}

func TestServiceBuilderRejectsNonAncestorWithRepeatedParentName(t *testing.T) {
	root := newTestRoot("dws")
	dev := &cobra.Command{Use: "dev"}
	firstService := &cobra.Command{Use: "service"}
	mcp := &cobra.Command{Use: "mcp"}
	parent := &cobra.Command{Use: "service"}
	root.AddCommand(dev)
	dev.AddCommand(firstService)
	firstService.AddCommand(mcp)
	mcp.AddCommand(parent)

	builder, err := NewServiceBuilder(parent)
	if err != nil {
		t.Fatal(err)
	}
	err = builder.Add(serviceMethodDef("dev mcp service list"), MethodBinding{
		Invoke: func(*Ctx, map[string]any) error { return nil },
	})
	if err == nil || !strings.Contains(err.Error(), "is not an ancestor") {
		t.Fatalf("ancestry error = %v", err)
	}
	if findChild(parent, "list") != nil || findChild(root, "mcp") != nil {
		t.Fatal("non-ancestor parent was mutated")
	}
}

func TestServiceBuilderRejectsRunnableIntermediateCommand(t *testing.T) {
	root := newTestRoot("dws")
	called := false
	dev := &cobra.Command{
		Use: "dev",
		RunE: func(*cobra.Command, []string) error {
			called = true
			return nil
		},
	}
	root.AddCommand(dev)
	builder, err := NewServiceBuilder(root)
	if err != nil {
		t.Fatal(err)
	}
	err = builder.Add(serviceMethodDef("dev mcp service list"), MethodBinding{
		Invoke: func(*Ctx, map[string]any) error { return nil },
	})
	if err == nil || !strings.Contains(err.Error(), "is runnable and cannot be used as a group") {
		t.Fatalf("runnable intermediate error = %v", err)
	}
	if findChild(dev, "mcp") != nil || dev.Annotations[groupPolicyAnnotation] != "" {
		t.Fatal("runnable intermediate was mutated")
	}
	if err := dev.RunE(dev, nil); err != nil {
		t.Fatal(err)
	}
	if !called {
		t.Fatal("original runnable command was replaced")
	}
}

func TestServiceBuilderRejectsRunnableParent(t *testing.T) {
	root := newTestRoot("dws")
	dev := &cobra.Command{Use: "dev"}
	mcp := &cobra.Command{Use: "mcp"}
	called := false
	service := &cobra.Command{
		Use: "service",
		RunE: func(*cobra.Command, []string) error {
			called = true
			return nil
		},
	}
	root.AddCommand(dev)
	dev.AddCommand(mcp)
	mcp.AddCommand(service)

	builder, err := NewServiceBuilder(service)
	if err != nil {
		t.Fatal(err)
	}
	err = builder.Add(serviceMethodDef("dev mcp service list"), MethodBinding{
		Invoke: func(*Ctx, map[string]any) error { return nil },
	})
	if err == nil || !strings.Contains(err.Error(), "is runnable and cannot be used as a group") {
		t.Fatalf("runnable parent error = %v", err)
	}
	if findChild(service, "list") != nil || service.Annotations[groupPolicyAnnotation] != "" {
		t.Fatal("runnable parent was mutated")
	}
	if err := service.RunE(service, nil); err != nil {
		t.Fatal(err)
	}
	if !called {
		t.Fatal("original runnable parent was replaced")
	}
}

func TestServiceBuilderRejectsExistingLeafBeforeMutation(t *testing.T) {
	root := newTestRoot("dws")
	first, err := NewServiceBuilder(root)
	if err != nil {
		t.Fatal(err)
	}
	def := serviceMethodDef("dev mcp service list")
	binding := MethodBinding{Invoke: func(*Ctx, map[string]any) error { return nil }}
	if err := first.Add(def, binding); err != nil {
		t.Fatal(err)
	}
	dev := findChild(root, "dev")
	mcp := findChild(dev, "mcp")
	service := findChild(mcp, "service")
	before := len(root.Commands()) + len(dev.Commands()) + len(mcp.Commands()) + len(service.Commands())

	second, err := NewServiceBuilder(root)
	if err != nil {
		t.Fatal(err)
	}
	err = second.Add(def, binding)
	if err == nil || !strings.Contains(err.Error(), "duplicate CLIPath") {
		t.Fatalf("duplicate error = %v", err)
	}
	after := len(root.Commands()) + len(dev.Commands()) + len(mcp.Commands()) + len(service.Commands())
	if after != before {
		t.Fatalf("tree changed after duplicate: before=%d after=%d", before, after)
	}
}

func TestServiceBuilderCreatesGroupsWithPolicy(t *testing.T) {
	root := newTestRoot("dws")
	b, err := NewServiceBuilder(root)
	if err != nil {
		t.Fatal(err)
	}
	invoker := func(c *Ctx, toolArgs map[string]any) error { return nil }
	if err := b.Add(serviceMethodDef("dev mcp service list"), MethodBinding{Invoke: invoker}); err != nil {
		t.Fatal(err)
	}
	dev := findChild(root, "dev")
	if dev == nil {
		t.Fatal("dev group not created")
	}
	policy, ok, err := GroupPolicyFor(dev)
	if err != nil || !ok || policy.Mode != GroupNavigationOnly {
		t.Fatalf("dev GroupPolicy = %+v, %v, %v", policy, ok, err)
	}
	mcp := findChild(dev, "mcp")
	if mcp == nil {
		t.Fatal("mcp group not created")
	}
	service := findChild(mcp, "service")
	if service == nil {
		t.Fatal("service group not created")
	}
	list := findChild(service, "list")
	if list == nil {
		t.Fatal("list leaf not mounted")
	}
	// Second method under the same path reuses groups.
	if err := b.Add(serviceMethodDef("dev mcp service get"), MethodBinding{Invoke: invoker}); err != nil {
		t.Fatal(err)
	}
	if got := len(dev.Commands()); got != 1 {
		t.Fatalf("dev children = %d, want 1 (group reused)", got)
	}
	if got := len(service.Commands()); got != 2 {
		t.Fatalf("service children = %d, want 2", got)
	}
}

// TestServiceBuilderToolBindingExecutesAssembledArgs executes a Tool-bound
// method end-to-end through the corecmd pipeline and checks the assembled
// toolArgs reach the provider executor.
func TestServiceBuilderToolBindingExecutesAssembledArgs(t *testing.T) {
	var captured map[string]any
	def := serviceMethodDef("dev mcp service list")
	def.Input.Parameters = []ParameterSpec{
		{Name: "keyword", Type: KindString, Usage: "按服务名关键词过滤", Bind: "keyword"},
		{Name: "page-size", Type: KindInt, Usage: "每页条数", Bind: "pageSize", Default: "", ArgDefault: "20"},
	}
	binding := ToolBinding("mcp_service_list", func(c *Ctx, toolArgs map[string]any) error {
		captured = toolArgs
		return nil
	})
	spec, err := compileMethod(def, binding)
	if err != nil {
		t.Fatal(err)
	}
	cmd := New(spec)
	cmd.SetArgs([]string{"--keyword", "客户"})
	if err := ExecuteForTest(cmd); err != nil {
		t.Fatalf("execute: %v", err)
	}
	if captured["keyword"] != "客户" {
		t.Fatalf("keyword = %v", captured["keyword"])
	}
	if captured["pageSize"] != 20 {
		t.Fatalf("pageSize = %v (want ArgDefault floor 20)", captured["pageSize"])
	}
}

// TestServiceBuilderLeafTierAndContractIdentity checks the compiled leaf keeps
// corecmd Tier1 semantics: contract annotation installed, required validation
// enforced with the framework's own message.
func TestServiceBuilderLeafTierAndContractIdentity(t *testing.T) {
	def := serviceMethodDef("dev mcp service get")
	def.Identity.Name = "mcp_service_get"
	def.Input.Parameters = []ParameterSpec{
		{Name: "mcp-id", Type: KindInt, Usage: "MCP 服务 ID", Bind: "mcpId", Required: true, RequiredHint: "--mcp-id 为必填"},
	}
	spec, err := compileMethod(def, MethodBinding{Invoke: func(c *Ctx, toolArgs map[string]any) error { return nil }})
	if err != nil {
		t.Fatal(err)
	}
	cmd := New(spec)
	cmd.SetArgs([]string{})
	err = ExecuteForTest(cmd)
	if err == nil || !strings.Contains(err.Error(), "--mcp-id 为必填") {
		t.Fatalf("required err = %v", err)
	}
}

func newTestRoot(name string) *cobra.Command {
	return &cobra.Command{Use: name}
}

func findChild(cmd *cobra.Command, name string) *cobra.Command {
	for _, child := range cmd.Commands() {
		if child.Name() == name {
			return child
		}
	}
	return nil
}
