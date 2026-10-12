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
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/DingTalk-Real-AI/dingtalk-workspace-cli/internal/corecmd/contract"
	"github.com/DingTalk-Real-AI/dingtalk-workspace-cli/internal/output"
)

// MethodDefinition is the service-method authoring declaration compiled by
// ServiceBuilder into the existing corecmd.Spec + ContractDecl (RFC §5.12).
// It owns the complete reviewed surface declaration plus constrained local
// validation/mount decoration; the execution owner is installed by pairing it
// with a MethodBinding whose exactly-one rule is enforced at compile time.
type MethodDefinition struct {
	Identity      MethodIdentity
	Input         InputSpec
	Contract      ContractSpec
	Safety        contract.SafetySpec
	OutputRollout output.RolloutState

	// Validate is a provider-specific local check. It runs in the existing
	// corecmd preflight boundary; it is not a second CLI surface declaration.
	Validate func(cmd *cobra.Command, args []string) error

	// PostMount is limited to framework/compatibility decoration after declared
	// flags are registered (for example, reviewed backend-source annotations).
	// It cannot add business flags; corecmd.New owns the mount order.
	PostMount func(cmd *cobra.Command)
}

// MethodIdentity declares the method's CLI location, backend routing names,
// and help prose. CLIPath is the user command path used to build the Cobra
// tree; ProductID/Name form the stable canonical identity (ProductID.Name).
type MethodIdentity struct {
	CLIPath     string
	ProductID   string
	Name        string
	Short       string
	Long        string
	Description string
	Example     string
}

// InputSpec declares the method's parameters and cross-flag constraints.
type InputSpec struct {
	Parameters  []ParameterSpec
	Constraints []Constraint
}

// ParameterSpec is the service authoring layer for one flag; the compiler
// maps it to the existing FlagSpec (which stays the binding/Schema authority).
// Fields mirror FlagSpec one-for-one so a service author does not maintain a
// second flag declaration for help, Schema, and runtime binding parity.
type ParameterSpec struct {
	Name      string
	Type      FlagKind
	Shorthand string
	Bind      string
	Usage     string
	Default   string
	Hidden    bool

	Required       bool
	RequiredHint   string
	RequiredError  string
	MarkRequired   bool
	ArgDefault     string
	ValidationMode FlagValidationMode

	Aliases []string
	EnvVar  string
	Input   []string

	Transform func(raw string) (any, error)
	OmitEmpty bool
	Trim      bool

	Enum              []string
	Format            string
	Example           string
	RequiredWhen      string
	SchemaDescription string
}

// ContractSpec declares the leaf Contract facts (selection / interface /
// dry-run / result / pagination). The compiler maps them onto ContractDecl;
// Identity is derived from MethodIdentity (CLIPath consistency is enforced,
// not inferred: a mismatch is a compile error).
type ContractSpec struct {
	Selection  contract.SelectionSpec
	Interface  *contract.InterfaceSpec
	DryRun     *contract.DryRunSpec
	Result     *contract.ResultSpec
	Pagination *contract.PaginationSpec
}

// MethodBinding installs exactly one execution owner onto a MethodDefinition
// (RFC §5.12.1): Tool (standard single call — paired with the provider dispatch
// executor via ToolBinding), Invoke (custom single call under the §5.12.3
// boundary), or Orchestrate (multi-step flow, exclusive). Exactly one owner
// must be declared; the compiler rejects zero or conflicting owners
// fail-closed. RunE is deliberately absent — it is legacy-migration only, not
// a MethodDef authoring surface.
type MethodBinding struct {
	Tool string
	// ToolExecutor is supplied by the provider runtime for a standard Tool
	// binding; it is not a second execution owner.
	ToolExecutor func(c *Ctx, toolArgs map[string]any) error
	Invoke       func(c *Ctx, toolArgs map[string]any) error
	Orchestrate  func(c *Ctx) error
}

// ServiceBuilder compiles MethodDefinitions into a Cobra subtree through the
// existing corecmd pipeline. It creates or reuses intermediate groups along
// each method's CLIPath (applying the existing ApplyGroupPolicy to groups it
// creates), compiles every definition into a corecmd.Spec, and mounts the
// runnable leaf via corecmd.New. It holds no registry, never calls
// PrepareCommandTree, and never generates commands from Schema (RFC §5.12.2).
type ServiceBuilder struct {
	parent *cobra.Command
	paths  map[string]bool
}

// NewServiceBuilder returns a builder that mounts methods under parent.
func NewServiceBuilder(parent *cobra.Command) (*ServiceBuilder, error) {
	if parent == nil {
		return nil, fmt.Errorf("corecmd.NewServiceBuilder: parent is nil")
	}
	return &ServiceBuilder{
		parent: parent,
		paths:  map[string]bool{},
	}, nil
}

// Build returns the parent tree after all added methods have been mounted.
// It does not prepare or execute the tree; the application root owns that
// lifecycle step after every provider has mounted its commands.
func (b *ServiceBuilder) Build() *cobra.Command {
	if b == nil {
		return nil
	}
	return b.parent
}

// Add validates one MethodDefinition + MethodBinding pair and mounts its
// compiled leaf. The pair is the legacy authoring unit: definitions without an
// owner, or with two owners, are rejected before any Cobra mutation.
func (b *ServiceBuilder) Add(def MethodDefinition, binding MethodBinding) error {
	if b == nil {
		return fmt.Errorf("corecmd.ServiceBuilder.Add: builder is nil")
	}
	spec, err := compileMethod(def, binding)
	if err != nil {
		return err
	}
	return b.mount(def.Identity.CLIPath, spec)
}

func (b *ServiceBuilder) mount(cliPath string, spec Spec) error {
	segments := pathSegments(cliPath)
	if len(segments) < 2 {
		return fmt.Errorf("corecmd.ServiceBuilder: CLIPath %q must include at least a service group and a leaf segment", cliPath)
	}
	full := "/" + strings.Join(segments, "/")
	if b.paths[full] {
		return fmt.Errorf("corecmd.ServiceBuilder: duplicate CLIPath %q", cliPath)
	}
	relativeGroups, err := b.relativeGroups(segments[:len(segments)-1], cliPath)
	if err != nil {
		return err
	}
	if err := b.preflightMount(relativeGroups, spec.Use, cliPath); err != nil {
		return err
	}
	group, err := b.groupFor(relativeGroups)
	if err != nil {
		return err
	}
	for _, child := range group.Commands() {
		if child.Name() == spec.Use {
			return fmt.Errorf("corecmd.ServiceBuilder: duplicate CLIPath %q", cliPath)
		}
	}
	leaf := New(spec)
	leaf.Args = cobra.NoArgs
	leaf.DisableAutoGenTag = true
	group.AddCommand(leaf)
	b.paths[full] = true
	return nil
}

// relativeGroups removes the path prefix represented by an attached builder
// parent. Root builders retain the complete CLIPath group path so they can
// create the provider subtree below the executable root.
func (b *ServiceBuilder) relativeGroups(segs []string, cliPath string) ([]string, error) {
	if b.parent.Parent() == nil {
		return segs, nil
	}
	parentSegments := pathSegments(b.parent.CommandPath())
	if len(parentSegments) == 0 {
		return nil, fmt.Errorf("corecmd.ServiceBuilder: parent %q has no attached command path for CLIPath %q", b.parent.CommandPath(), cliPath)
	}
	candidates := [][]string{parentSegments}
	if len(parentSegments) > 1 {
		candidates = append(candidates, parentSegments[1:])
	}
	for _, candidate := range candidates {
		if len(candidate) <= len(segs) && hasPathPrefix(segs, candidate) {
			return segs[len(candidate):], nil
		}
	}
	return nil, fmt.Errorf("corecmd.ServiceBuilder: parent %q is not an ancestor of CLIPath %q", b.parent.CommandPath(), cliPath)
}

func hasPathPrefix(path, prefix []string) bool {
	if len(prefix) > len(path) {
		return false
	}
	for i, segment := range prefix {
		if path[i] != segment {
			return false
		}
	}
	return true
}

func (b *ServiceBuilder) preflightMount(groups []string, leaf, cliPath string) error {
	if err := validateMountGroup(b.parent, "parent"); err != nil {
		return fmt.Errorf("corecmd.ServiceBuilder: CLIPath %q: %w", cliPath, err)
	}
	current := b.parent
	for _, seg := range groups {
		if strings.TrimSpace(seg) == "" {
			return fmt.Errorf("corecmd.ServiceBuilder: empty path segment in %v", groups)
		}
		next := findCommandChild(current, seg)
		if next == nil {
			return nil
		}
		if err := validateMountGroup(next, "intermediate group"); err != nil {
			return fmt.Errorf("corecmd.ServiceBuilder: CLIPath %q: %w", cliPath, err)
		}
		current = next
	}
	if findCommandChild(current, leaf) != nil {
		return fmt.Errorf("corecmd.ServiceBuilder: duplicate CLIPath %q", cliPath)
	}
	return nil
}

func validateMountGroup(cmd *cobra.Command, role string) error {
	if cmd == nil {
		return fmt.Errorf("%s command is nil", role)
	}
	if _, ok, err := GroupPolicyFor(cmd); err != nil {
		return fmt.Errorf("%s %q has invalid policy: %w", role, cmd.CommandPath(), err)
	} else if !ok && commandHasExecution(cmd) {
		return fmt.Errorf("%s %q is runnable and cannot be used as a group", role, cmd.CommandPath())
	}
	return nil
}

func commandHasExecution(cmd *cobra.Command) bool {
	return cmd.Run != nil || cmd.RunE != nil
}

func findCommandChild(parent *cobra.Command, name string) *cobra.Command {
	for _, child := range parent.Commands() {
		if child.Name() == name {
			return child
		}
	}
	return nil
}

// groupFor walks/creates intermediate groups for segs under the builder
// parent. Created groups are navigation-only containers configured with the
// default group policy (navigation_only / reject / sibling), matching the
// hand-written helper containers being migrated. Pre-existing groups are
// reused as-is — their owning declaration stays authoritative.
func (b *ServiceBuilder) groupFor(segs []string) (*cobra.Command, error) {
	current := b.parent
	for _, seg := range segs {
		if strings.TrimSpace(seg) == "" {
			return nil, fmt.Errorf("corecmd.ServiceBuilder: empty path segment in %v", segs)
		}
		next := findCommandChild(current, seg)
		if next == nil {
			next = &cobra.Command{
				Use:              seg,
				TraverseChildren: true,
			}
			current.AddCommand(next)
		}
		if _, ok, err := GroupPolicyFor(next); err != nil {
			return nil, fmt.Errorf("corecmd.ServiceBuilder: group %q has invalid policy: %w", next.CommandPath(), err)
		} else if !ok {
			if commandHasExecution(next) {
				return nil, fmt.Errorf("corecmd.ServiceBuilder: group %q is runnable and cannot be used as a group", next.CommandPath())
			}
			ApplyGroupPolicy(next, GroupPolicy{
				Mode:        GroupNavigationOnly,
				Positionals: PositionalsReject,
				Recovery:    RecoverySibling,
			})
		}
		current = next
	}
	return current, nil
}

// compileMethod maps a MethodDefinition + legacy MethodBinding pair to the
// existing corecmd.Spec. Declaration compilation is shared with typed
// Operation.Mount so flags, Contract and Schema facts have one authority.
func compileMethod(def MethodDefinition, binding MethodBinding) (Spec, error) {
	if err := validateMethodBinding(def, binding); err != nil {
		return Spec{}, err
	}
	spec, err := compileMethodDefinition(def)
	if err != nil {
		return Spec{}, err
	}
	switch {
	case binding.Orchestrate != nil:
		spec.Orchestrate = binding.Orchestrate
	case binding.Tool != "":
		spec.Invoke = binding.ToolExecutor
	case binding.Invoke != nil:
		spec.Invoke = binding.Invoke
	default:
		return Spec{}, fmt.Errorf("method %s: unreachable binding state", canonicalMethod(def))
	}
	return spec, nil
}

func compileMethodDefinition(def MethodDefinition) (Spec, error) {
	if err := validateMethodDefinition(def); err != nil {
		return Spec{}, err
	}
	segments := pathSegments(def.Identity.CLIPath)
	use := segments[len(segments)-1]

	flags := make([]FlagSpec, 0, len(def.Input.Parameters))
	seenFlagNames := map[string]bool{}
	seenBindKeys := map[string]bool{}
	for _, p := range def.Input.Parameters {
		if strings.TrimSpace(p.Name) == "" {
			return Spec{}, fmt.Errorf("method %s: parameter name is empty", canonicalMethod(def))
		}
		if seenFlagNames[p.Name] {
			return Spec{}, fmt.Errorf("method %s: duplicate parameter %q", canonicalMethod(def), p.Name)
		}
		if p.Type < KindString || p.Type > KindStringSlice {
			return Spec{}, fmt.Errorf("method %s: parameter %q has invalid type %d", canonicalMethod(def), p.Name, p.Type)
		}
		seenFlagNames[p.Name] = true
		bind := bindKey(FlagSpec{Name: p.Name, Bind: p.Bind})
		if seenBindKeys[bind] {
			return Spec{}, fmt.Errorf("method %s: duplicate parameter binding %q", canonicalMethod(def), bind)
		}
		seenBindKeys[bind] = true
		flags = append(flags, FlagSpec{
			Name:              p.Name,
			Shorthand:         p.Shorthand,
			Usage:             p.Usage,
			Kind:              p.Type,
			Default:           p.Default,
			Required:          p.Required,
			ValidationMode:    p.ValidationMode,
			RequiredError:     p.RequiredError,
			RequiredHint:      p.RequiredHint,
			MarkRequired:      p.MarkRequired,
			ArgDefault:        p.ArgDefault,
			Aliases:           p.Aliases,
			EnvVar:            p.EnvVar,
			Input:             p.Input,
			Transform:         p.Transform,
			OmitEmpty:         p.OmitEmpty,
			Trim:              p.Trim,
			Enum:              p.Enum,
			Format:            p.Format,
			Example:           p.Example,
			RequiredWhen:      p.RequiredWhen,
			SchemaDescription: firstNonEmpty(p.SchemaDescription, p.Usage),
			Hidden:            p.Hidden,
			Bind:              p.Bind,
		})
	}

	decl := ContractDecl{
		Description: def.Identity.Description,
		Selection:   def.Contract.Selection,
		Interface:   def.Contract.Interface,
		DryRun:      def.Contract.DryRun,
		Result:      def.Contract.Result,
		Pagination:  def.Contract.Pagination,
		Identity: contract.ToolIdentitySpec{
			ProductID:      def.Identity.ProductID,
			Name:           def.Identity.Name,
			CanonicalPath:  def.Identity.ProductID + "." + def.Identity.Name,
			CLIPath:        def.Identity.CLIPath,
			PrimaryCLIPath: def.Identity.CLIPath,
		},
	}

	return Spec{
		Use:           use,
		Short:         firstNonEmpty(def.Identity.Short, def.Identity.Description),
		Long:          def.Identity.Long,
		Example:       def.Identity.Example,
		Flags:         flags,
		Constraints:   def.Input.Constraints,
		Safety:        def.Safety,
		Contract:      decl,
		Validate:      def.Validate,
		PostMount:     def.PostMount,
		OutputRollout: def.OutputRollout,
	}, nil
}

// ToolBinding installs the standard single-call executor for a Tool binding:
// the provider passes a dispatch func that receives the assembled toolArgs and
// routes them to the backend tool. This keeps corecmd dispatch-agnostic while
// giving Tool-bound methods the default executor the RFC prescribes.
func ToolBinding(tool string, dispatch func(c *Ctx, toolArgs map[string]any) error) MethodBinding {
	return MethodBinding{Tool: tool, ToolExecutor: dispatch}
}

func validateMethodDefinition(def MethodDefinition) error {
	path := strings.TrimSpace(def.Identity.CLIPath)
	if path == "" {
		return fmt.Errorf("method definition: Identity.CLIPath is empty")
	}
	segments := pathSegments(path)
	if strings.Join(segments, " ") != path {
		return fmt.Errorf("method definition %q: CLIPath contains an empty or malformed segment", def.Identity.CLIPath)
	}
	if strings.Contains(path, "/") {
		return fmt.Errorf("method definition %q: CLIPath must use space-separated command segments", def.Identity.CLIPath)
	}
	if len(segments) < 2 {
		return fmt.Errorf("method definition %q: CLIPath must include at least a service group and a leaf segment", def.Identity.CLIPath)
	}
	if strings.TrimSpace(def.Identity.ProductID) == "" {
		return fmt.Errorf("method %s: Identity.ProductID is empty", def.Identity.CLIPath)
	}
	if strings.TrimSpace(def.Identity.Name) == "" {
		return fmt.Errorf("method %s: Identity.Name is empty", def.Identity.CLIPath)
	}
	if strings.TrimSpace(def.Identity.Description) == "" {
		return fmt.Errorf("method %s: Identity.Description is empty", def.Identity.CLIPath)
	}
	if err := contract.ValidateToolIdentity(def.Identity.ProductID, def.Identity.Name); err != nil {
		return fmt.Errorf("method %s: invalid identity: %w", def.Identity.CLIPath, err)
	}
	return nil
}

func validateMethodBinding(def MethodDefinition, binding MethodBinding) error {
	// Tool + ToolExecutor is one standard Tool owner. Invoke and Orchestrate
	// are alternative owners and cannot be combined with either Tool field.
	toolOwner := binding.Tool != ""
	switch {
	case binding.Orchestrate != nil:
		if toolOwner || binding.ToolExecutor != nil || binding.Invoke != nil {
			return fmt.Errorf(
				"method %s: MethodBinding Orchestrate cannot be combined with Tool/Invoke",
				def.Identity.CLIPath)
		}
	case toolOwner:
		if binding.ToolExecutor == nil || binding.Invoke != nil {
			return fmt.Errorf(
				"method %s: Tool binding requires ToolExecutor and cannot be combined with Invoke",
				def.Identity.CLIPath)
		}
		if binding.Tool != def.Identity.Name {
			return fmt.Errorf(
				"method %s: MethodBinding.Tool %q does not match Identity.Name %q",
				def.Identity.CLIPath, binding.Tool, def.Identity.Name)
		}
	case binding.Invoke != nil:
		if binding.ToolExecutor != nil {
			return fmt.Errorf(
				"method %s: custom Invoke cannot be combined with ToolExecutor",
				def.Identity.CLIPath)
		}
	default:
		return fmt.Errorf(
			"method %s: MethodBinding must set exactly one of Tool/Invoke/Orchestrate",
			def.Identity.CLIPath)
	}
	return nil
}

// pathSegments splits and trims a CLIPath without empty segments.
func pathSegments(cliPath string) []string {
	raw := strings.Split(strings.TrimSpace(cliPath), " ")
	segments := make([]string, 0, len(raw))
	for _, seg := range raw {
		if strings.TrimSpace(seg) != "" {
			segments = append(segments, seg)
		}
	}
	return segments
}

func canonicalMethod(def MethodDefinition) string {
	return def.Identity.ProductID + "." + def.Identity.Name
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value = strings.TrimSpace(value); value != "" {
			return value
		}
	}
	return ""
}
