package corecmd

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/DingTalk-Real-AI/dingtalk-workspace-cli/internal/corecmd/contract"
	apperrors "github.com/DingTalk-Real-AI/dingtalk-workspace-cli/internal/errors"
	"github.com/DingTalk-Real-AI/dingtalk-workspace-cli/internal/output"
)

func TestOperationMountExecutesTypedRequestWithoutBuildArgs(t *testing.T) {
	var gotRequest typedTestRequest
	var gotOptions OperationOptions
	transformCalled := false
	operation := Operation[typedTestRequest, typedTestResponse]{
		MethodDefinition: MethodDefinition{
			Identity: MethodIdentity{
				CLIPath:     "dev mcp service get",
				ProductID:   "dev",
				Name:        "mcp_service_get",
				Description: "查询 MCP 服务详情",
			},
			Input: InputSpec{Parameters: []ParameterSpec{
				{
					Name:     "mcp-id",
					Type:     KindInt,
					Bind:     "mcpId",
					Usage:    "MCP 服务 ID",
					Required: true,
				},
				{
					Name: "label", Type: KindString, Usage: "label",
					Transform: func(string) (any, error) {
						transformCalled = true
						return nil, errors.New("BuildArgs should not run")
					},
				},
			}},
			Contract: ContractSpec{
				Selection: contract.SelectionSpec{
					AgentSummary: "查询 MCP 服务详情",
					UseWhen:      []string{"查询 MCP 服务详情"},
					AvoidWhen:    []string{"不需要查询开发配置时不要使用"},
					Examples:     []string{"dev mcp service get"},
				},
				Interface: &contract.InterfaceSpec{
					Mode:         contract.InterfaceModeComposite,
					Availability: contract.InterfaceAvailable,
					Reason:       "test",
				},
			},
			Safety:        contract.SafetySpec{Effect: "read", Risk: "low", Confirmation: "not_required", Idempotency: "idempotent"},
			OutputRollout: output.RolloutUnifiedActive,
		},
		Decode: func(c *Ctx) (typedTestRequest, error) {
			return typedTestRequest{MCPID: c.Int("mcp-id")}, nil
		},
		Execute: func(_ context.Context, request typedTestRequest, options OperationOptions) (typedTestResponse, error) {
			gotRequest = request
			gotOptions = options
			return typedTestResponse{MCPID: request.MCPID}, nil
		},
		Project: func(response typedTestResponse, options OperationOptions) output.CommandResult {
			if options.DryRun {
				return output.Success(response, output.WithDryRun())
			}
			return output.Success(response)
		},
	}

	root := newTestRoot("dws")
	root.PersistentFlags().Bool("dry-run", false, "")
	builder, err := NewServiceBuilder(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := operation.Mount(builder); err != nil {
		t.Fatalf("Mount: %v", err)
	}
	cmd := findChild(findChild(findChild(root, "dev"), "mcp"), "service")
	if cmd == nil {
		t.Fatal("service group missing")
	}
	leaf := findChild(cmd, "get")
	if leaf == nil {
		t.Fatal("get leaf missing")
	}
	root.SetArgs([]string{"dev", "mcp", "service", "get", "--mcp-id", "10487"})
	if err := ExecuteForTest(root); err != nil {
		t.Fatalf("execute: %v", err)
	}
	if transformCalled {
		t.Fatal("typed operation unexpectedly called BuildArgs transform")
	}
	if gotRequest != (typedTestRequest{MCPID: 10487}) {
		t.Fatalf("request = %#v", gotRequest)
	}
	if gotOptions.DryRun {
		t.Fatal("DryRun = true, want false")
	}
}

func TestOperationMountCarriesDryRunAndStoresTypedResult(t *testing.T) {
	var gotOptions OperationOptions
	operation := Operation[int, typedTestResponse]{
		MethodDefinition: MethodDefinition{
			Identity: MethodIdentity{CLIPath: "dev mcp service get", ProductID: "dev", Name: "mcp_service_get", Description: "get"},
			Input:    InputSpec{Parameters: []ParameterSpec{{Name: "mcp-id", Type: KindInt, Usage: "id", Required: true}}},
			Contract: ContractSpec{
				Selection: contract.SelectionSpec{
					AgentSummary: "get",
					UseWhen:      []string{"get"},
					AvoidWhen:    []string{"not get"},
					Examples:     []string{"dev mcp service get"},
				},
				Interface: &contract.InterfaceSpec{
					Mode:         contract.InterfaceModeComposite,
					Availability: contract.InterfaceAvailable,
					Reason:       "test",
				},
			},
			Safety:        contract.SafetySpec{Effect: "read", Risk: "low", Confirmation: "not_required", Idempotency: "idempotent"},
			OutputRollout: output.RolloutUnifiedActive,
		},
		Decode: func(c *Ctx) (int, error) { return c.Int("mcp-id"), nil },
		Execute: func(_ context.Context, request int, options OperationOptions) (typedTestResponse, error) {
			gotOptions = options
			return typedTestResponse{MCPID: request}, nil
		},
		Project: func(response typedTestResponse, options OperationOptions) output.CommandResult {
			if !options.DryRun {
				return output.Success(response)
			}
			return output.Success(response, output.WithDryRun())
		},
	}

	root := newTestRoot("dws")
	root.PersistentFlags().Bool("dry-run", false, "")
	builder, err := NewServiceBuilder(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := operation.Mount(builder); err != nil {
		t.Fatal(err)
	}
	leaf := findChild(findChild(findChild(findChild(root, "dev"), "mcp"), "service"), "get")
	if leaf == nil {
		t.Fatal("get leaf missing")
	}
	root.SetArgs([]string{"dev", "mcp", "service", "get", "--mcp-id", "10487", "--dry-run"})
	if err := ExecuteForTest(root); err != nil {
		t.Fatalf("execute: %v", err)
	}
	if !gotOptions.DryRun {
		t.Fatal("DryRun = false, want true")
	}
}

func TestOperationMountRequiresSafetyConfirmation(t *testing.T) {
	executed := false
	operation := Operation[int, int]{
		MethodDefinition: func() MethodDefinition {
			def := serviceMethodDef("dev mcp service update")
			def.Identity.Name = "mcp_service_update"
			def.Safety = contract.SafetySpec{Effect: "write", Risk: "medium", Confirmation: "user_required", Idempotency: "unknown"}
			def.OutputRollout = output.RolloutUnifiedActive
			return def
		}(),
		Decode: func(c *Ctx) (int, error) { return c.Int("page-size"), nil },
		Execute: func(context.Context, int, OperationOptions) (int, error) {
			executed = true
			return 1, nil
		},
		Project: func(int, OperationOptions) output.CommandResult { return output.Success(nil) },
	}
	root := newTestRoot("dws")
	builder, err := NewServiceBuilder(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := operation.Mount(builder); err != nil {
		t.Fatal(err)
	}
	root.SetIn(strings.NewReader(""))
	root.SetArgs([]string{"dev", "mcp", "service", "update"})
	err = ExecuteForTest(root)
	if !apperrors.IsConfirmationRequired(err) {
		t.Fatalf("error = %v, want confirmation_required", err)
	}
	if executed {
		t.Fatal("typed operation executed before confirmation")
	}
}

func TestOperationValidationRejectsIncompleteDeclaration(t *testing.T) {
	base := Operation[int, int]{
		MethodDefinition: serviceMethodDef("dev mcp service get"),
		Decode:           func(*Ctx) (int, error) { return 0, nil },
		Execute:          func(context.Context, int, OperationOptions) (int, error) { return 0, nil },
		Project:          func(int, OperationOptions) output.CommandResult { return output.Success(nil) },
	}
	for _, tc := range []struct {
		name   string
		mutate func(*Operation[int, int])
		want   string
	}{
		{"missing decode", func(op *Operation[int, int]) { op.Decode = nil }, "Operation.Decode is nil"},
		{"missing execute", func(op *Operation[int, int]) { op.Execute = nil }, "Operation.Execute is nil"},
		{"missing project", func(op *Operation[int, int]) { op.Project = nil }, "Operation.Project is nil"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			op := base
			tc.mutate(&op)
			if _, err := compileOperation(op); err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("compile error = %v, want %q", err, tc.want)
			}
		})
	}
}

type typedTestRequest struct {
	MCPID int
}

type typedTestResponse struct {
	MCPID int
}
