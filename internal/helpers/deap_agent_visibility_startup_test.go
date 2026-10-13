// Copyright 2026 Alibaba Group
// Licensed under the Apache License, Version 2.0 (the "License");

package helpers

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/DingTalk-Real-AI/dingtalk-workspace-cli/internal/auth"
	"github.com/DingTalk-Real-AI/dingtalk-workspace-cli/internal/testseam"
)

type visibilityStartupWriter func([]byte) (int, error)

func (f visibilityStartupWriter) Write(data []byte) (int, error) { return f(data) }

func TestCrossPlatformCoverageVisibilityLeaseMigratesBeforeReady(t *testing.T) {
	f := newVisibilityUpgradeFixture(t)
	previousProfile := auth.RuntimeProfile()
	auth.SetRuntimeProfile(f.binding.DWSProfile)
	t.Cleanup(func() { auth.SetRuntimeProfile(previousProfile) })
	cmd := newDeapConnectCommand()
	cmd.SetContext(context.Background())
	for name, value := range map[string]string{"agent-uuid": f.binding.AgentUUID, "channel": "dsh", "binding-revision": "9", "runtime-instance-id": "00000000-0000-4000-8000-000000000001"} {
		if err := cmd.Flags().Set(name, value); err != nil {
			t.Fatal(err)
		}
	}
	cmd.SetIn(strings.NewReader("released\n"))
	announced := false
	cmd.SetErr(visibilityStartupWriter(func(data []byte) (int, error) {
		if string(data) != "[employee] leased\n" {
			return 0, fmt.Errorf("unexpected announcement %q", data)
		}
		persisted, err := loadDigitalEmployeeBinding(deapConnectConfigDir(), f.binding.DWSProfile)
		if err != nil || persisted.SupervisorProfile != "manager-corp:original-manager" {
			return 0, errors.New("ready announced before supervisor persisted")
		}
		if _, err = os.Stat(employeeLeaseGuardPath(f.binding.DWSProfile)); err != nil {
			return 0, err
		}
		announced = true
		return len(data), nil
	}))
	if err := runEmployeeLease(cmd); err != nil {
		t.Fatal(err)
	}
	if !announced {
		t.Fatal("lease did not announce readiness")
	}
	if _, err := os.Stat(employeeLeaseGuardPath(f.binding.DWSProfile)); !os.IsNotExist(err) {
		t.Fatalf("released lease guard remains: %v", err)
	}
}

func TestCrossPlatformCoverageVisibilityLeaseFailureNeverReady(t *testing.T) {
	f := newVisibilityUpgradeFixture(t)
	f.profiles = nil
	previousProfile := auth.RuntimeProfile()
	auth.SetRuntimeProfile(f.binding.DWSProfile)
	t.Cleanup(func() { auth.SetRuntimeProfile(previousProfile) })
	cmd := newDeapConnectCommand()
	cmd.SetContext(context.Background())
	for name, value := range map[string]string{"agent-uuid": f.binding.AgentUUID, "channel": "dsh", "binding-revision": "9", "runtime-instance-id": "00000000-0000-4000-8000-000000000001"} {
		if err := cmd.Flags().Set(name, value); err != nil {
			t.Fatal(err)
		}
	}
	cmd.SetIn(strings.NewReader("released\n"))
	var announcement bytes.Buffer
	cmd.SetErr(&announcement)
	if err := runEmployeeLease(cmd); err == nil {
		t.Fatal("unmigrated lease was accepted")
	}
	if announcement.Len() != 0 {
		t.Fatalf("failed migration announced readiness: %q", announcement.String())
	}
	if _, err := os.Stat(employeeLeaseGuardPath(f.binding.DWSProfile)); !os.IsNotExist(err) {
		t.Fatalf("failed migration installed guard: %v", err)
	}
}

func visibilityStartupNativeFixture(t *testing.T) (*visibilityUpgradeFixture, digitalEmployeeAdapterConfig) {
	t.Helper()
	f := newVisibilityUpgradeFixture(t)
	f.binding.Channel = "codex"
	if err := updateEmployeeBinding(f.binding); err != nil {
		t.Fatal(err)
	}
	return f, digitalEmployeeAdapterConfig{Binding: f.binding, SelfOpenDingTalkID: "employee-self"}
}

func TestCrossPlatformCoverageVisibilityWorkerMigrationPrecedesAgentAndConsumer(t *testing.T) {
	for _, mode := range []string{"success", "missing"} {
		t.Run(mode, func(t *testing.T) {
			f, cfg := visibilityStartupNativeFixture(t)
			if mode == "missing" {
				f.profiles = nil
			}
			forwarderCalls, consumerCalls := 0, 0
			testseam.Swap(t, &digitalEmployeeNewForwarder, func(_ context.Context, actual digitalEmployeeAdapterConfig) (forwarder, error) {
				forwarderCalls++
				if actual.Binding.SupervisorProfile != "manager-corp:original-manager" {
					t.Fatalf("forwarder received old binding %#v", actual.Binding)
				}
				persisted, err := loadDigitalEmployeeBinding(deapConnectConfigDir(), f.binding.DWSProfile)
				if err != nil || persisted != actual.Binding {
					t.Fatalf("forwarder started before persisted binding: %#v %v", persisted, err)
				}
				return nil, errors.New("controlled-adapter-stop")
			})
			testseam.Swap(t, &employeeExecCommand, func(context.Context, string, ...string) *exec.Cmd {
				consumerCalls++
				return &exec.Cmd{Stderr: io.Discard}
			})
			err := runEmployeeWorker(context.Background(), cfg, 0)
			if mode == "missing" {
				assertVisibilityUpgradeFailure(t, err, "visibility_supervisor_missing")
				if forwarderCalls != 0 {
					t.Fatal("migration failure initialized agent")
				}
			} else {
				assertVisibilityUpgradeFailure(t, err, "agent_unavailable")
				if forwarderCalls != 1 {
					t.Fatalf("forwarder calls=%d", forwarderCalls)
				}
			}
			if consumerCalls != 0 {
				t.Fatal("test unexpectedly attempted consumer process")
			}
			state, readErr := readDigitalEmployeeState(digitalEmployeeRuntimeDir(f.binding.DWSProfile))
			if readErr != nil || state.Status != "blocked" || state.TransportReady || state.ExecutorReady {
				t.Fatalf("failed startup state=%#v err=%v", state, readErr)
			}
		})
	}
}

func TestCrossPlatformCoverageVisibilityWorkerStopDuringMigrationEndsStopped(t *testing.T) {
	f, cfg := visibilityStartupNativeFixture(t)
	entered := make(chan struct{})
	f.beforeDetail = func(ctx context.Context) { close(entered); <-ctx.Done(); f.detailErr = ctx.Err() }
	forwarderCalls, consumerCalls := 0, 0
	testseam.Swap(t, &digitalEmployeeNewForwarder, func(context.Context, digitalEmployeeAdapterConfig) (forwarder, error) {
		forwarderCalls++
		return nil, errors.New("unexpected-agent")
	})
	testseam.Swap(t, &employeeExecCommand, func(context.Context, string, ...string) *exec.Cmd {
		consumerCalls++
		return &exec.Cmd{Stderr: io.Discard}
	})
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- runEmployeeWorker(ctx, cfg, 0) }()
	select {
	case <-entered:
	case err := <-done:
		t.Fatalf("worker exited before migration: %v", err)
	case <-ctx.Done():
		t.Fatal("worker did not enter migration")
	}
	stoppedBinding := f.binding
	stoppedBinding.DesiredState = "stopped"
	if err := updateEmployeeBinding(stoppedBinding); err != nil {
		t.Fatal(err)
	}
	// 真实 stop 获取 activation、写入 stop.json；watcher 取消迁移的 RPC 上下文。
	stopCtx, stopCancel := context.WithTimeout(context.Background(), time.Second)
	defer stopCancel()
	stopErr := stopEmployeeRuntime(stopCtx, stoppedBinding)
	var workerErr error
	select {
	case workerErr = <-done:
	case <-ctx.Done():
		t.Fatal("worker did not leave migration after stop")
	}
	state, readErr := readDigitalEmployeeState(digitalEmployeeRuntimeDir(f.binding.DWSProfile))
	if readErr != nil {
		t.Fatal(readErr)
	}
	request, err := os.ReadFile(filepath.Join(digitalEmployeeRuntimeDir(f.binding.DWSProfile), "stop.json"))
	if err != nil || !bytes.Contains(request, []byte(state.RunID)) {
		t.Fatalf("stop request missing: %s %v", request, err)
	}
	if forwarderCalls != 0 || consumerCalls != 0 {
		t.Fatal("stopped startup invoked agent or consumer")
	}
	if state.Status != "stopped" || state.Code != "" || stopErr != nil || workerErr != nil {
		t.Fatalf("concurrent stop must finish cleanly: status=%s code=%s stopErr=%v workerErr=%v", state.Status, state.Code, stopErr, workerErr)
	}
}
