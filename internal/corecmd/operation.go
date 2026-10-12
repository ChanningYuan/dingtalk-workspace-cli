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
	"context"
	"fmt"

	"github.com/DingTalk-Real-AI/dingtalk-workspace-cli/internal/output"
)

// Operation is the typed command authoring unit. MethodDefinition owns the CLI
// and Contract declaration; Decode, Execute and Project own the typed runtime
// path without exposing assembled map arguments to the operation handler.
type Operation[Request any, Response any] struct {
	MethodDefinition
	Decode  func(*Ctx) (Request, error)
	Execute func(context.Context, Request, OperationOptions) (Response, error)
	Project func(Response, OperationOptions) output.CommandResult
}

// OperationOptions contains framework execution state that is meaningful to a
// typed handler without coupling it to Cobra or a provider transport.
type OperationOptions struct {
	DryRun  bool
	CLIPath string
}

// Mount compiles and mounts the typed operation through the existing command
// tree and Contract pipeline.
func (op Operation[Request, Response]) Mount(builder *ServiceBuilder) error {
	if builder == nil {
		return fmt.Errorf("corecmd.Operation.Mount: builder is nil")
	}
	spec, err := compileOperation(op)
	if err != nil {
		return err
	}
	return builder.mount(op.Identity.CLIPath, spec)
}

func compileOperation[Request any, Response any](op Operation[Request, Response]) (Spec, error) {
	if op.Decode == nil {
		return Spec{}, fmt.Errorf("method %s: Operation.Decode is nil", canonicalMethod(op.MethodDefinition))
	}
	if op.Execute == nil {
		return Spec{}, fmt.Errorf("method %s: Operation.Execute is nil", canonicalMethod(op.MethodDefinition))
	}
	if op.Project == nil {
		return Spec{}, fmt.Errorf("method %s: Operation.Project is nil", canonicalMethod(op.MethodDefinition))
	}
	spec, err := compileMethodDefinition(op.MethodDefinition)
	if err != nil {
		return Spec{}, err
	}
	spec.typedResultInvoke = func(c *Ctx) (output.CommandResult, error) {
		options := OperationOptions{DryRun: c.DryRun(), CLIPath: op.Identity.CLIPath}
		request, err := op.Decode(c)
		if err != nil {
			return nil, err
		}
		ctx := c.Command().Context()
		if ctx == nil {
			ctx = context.Background()
		}
		response, err := op.Execute(ctx, request, options)
		if err != nil {
			return nil, err
		}
		return op.Project(response, options), nil
	}
	return spec, nil
}
