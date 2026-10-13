// Copyright 2026 Alibaba Group
// Licensed under the Apache License, Version 2.0 (the "License");

package helpers

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/DingTalk-Real-AI/dingtalk-workspace-cli/internal/testseam"
	"github.com/DingTalk-Real-AI/dingtalk-workspace-cli/pkg/edition"
)

func visibilityUpgradeFirstLookupResponse(t *testing.T, f *visibilityUpgradeFixture, raw string, extraBlock bool) {
	t.Helper()
	InitDepsForTest(t, &visibilityUpgradeCaller{call: func(_ context.Context, token, server, tool string, args map[string]any) (*edition.ToolResult, error) {
		if token != "employee-fixture-only" || server != "contact" || tool != "search_contact_by_key_word" {
			t.Fatal("operator lookup must use the exact employee token and contact RPC")
		}
		user, _ := args["keyword"].(string)
		f.lookups = append(f.lookups, user)
		response := raw
		if user != "other-user" {
			encoded, err := json.Marshal(map[string]any{"result": []any{map[string]any{"userId": user, "openDingTalkId": f.operators[user]}}})
			if err != nil {
				t.Fatal(err)
			}
			response = string(encoded)
		}
		blocks := []edition.ContentBlock{{Type: "text", Text: response}}
		if user == "other-user" && extraBlock {
			blocks = append(blocks, edition.ContentBlock{Type: "text", Text: `{"success":false,"error":"private-service-detail"}`})
		}
		return &edition.ToolResult{Content: blocks}, nil
	}})
}

func TestCrossPlatformCoverageVisibilityUpgradeExplicitEmptyCandidate(t *testing.T) {
	f := newVisibilityUpgradeFixture(t)
	visibilityUpgradeFirstLookupResponse(t, f, `{"result":[],"success":true}`, false)
	got, err := prepareEmployeeVisibility(context.Background(), f.binding)
	want := f.binding
	want.SupervisorProfile = "manager-corp:original-manager"
	if err != nil || got != want {
		t.Fatalf("migration after explicit empty candidate = %#v, %v", got, err)
	}
	if strings.Join(f.lookups, ",") != "other-user,original-manager" || f.detailCalls != 1 {
		t.Fatalf("lookup sequence=%v published calls=%d", f.lookups, f.detailCalls)
	}
	persisted, err := loadDigitalEmployeeBinding(deapConnectConfigDir(), f.binding.DWSProfile)
	if err != nil || persisted != want {
		t.Fatalf("persisted migration = %#v, %v", persisted, err)
	}
}

func TestCrossPlatformCoverageVisibilityUpgradeUncertainEmptyCandidateRejected(t *testing.T) {
	cases := []struct {
		name, raw  string
		extraBlock bool
	}{
		{"missing success", `{"result":[]}`, false},
		{"missing result", `{"success":true}`, false},
		{"null result", `{"success":true,"result":null}`, false},
		{"object result", `{"success":true,"result":{}}`, false},
		{"nonbool success", `{"success":"true","result":[]}`, false},
		{"business rejected", `{"success":false,"result":[]}`, false},
		{"business error", `{"success":true,"result":[],"errorCode":1001}`, false},
		{"extra error", `{"success":true,"result":[],"error":"private-service-detail"}`, false},
		{"has more", `{"success":true,"result":[],"hasMore":true}`, false},
		{"unknown pagination envelope", `{"success":true,"result":[],"hasMore":false}`, false},
		{"cursor", `{"success":true,"result":[],"nextCursor":"next"}`, false},
		{"truncated", `{"success":true,"result":[],"truncated":true}`, false},
		{"malformed JSON", `{"success":true,"result":[]`, false},
		{"trailing JSON", `{"success":true,"result":[]} {"error":"private-service-detail"}`, false},
		{"duplicate success", `{"success":false,"success":true,"result":[]}`, false},
		{"duplicate result", `{"success":true,"result":[{}],"result":[]}`, false},
		{"nonempty without exact identity", `{"success":true,"result":[{"userId":"different-user","openDingTalkId":"unrelated"}]}`, false},
		{"exact user without open identity", `{"success":true,"result":[{"userId":"other-user"}]}`, false},
		{"multiple exact identities", `{"success":true,"result":[{"userId":"other-user","openDingTalkId":"one"},{"userId":"other-user","openDingTalkId":"two"}]}`, false},
		{"extra content block", `{"success":true,"result":[]}`, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := newVisibilityUpgradeFixture(t)
			visibilityUpgradeFirstLookupResponse(t, f, tc.raw, tc.extraBlock)
			before, err := os.ReadFile(f.file)
			if err != nil {
				t.Fatal(err)
			}
			_, err = prepareEmployeeVisibility(context.Background(), f.binding)
			assertVisibilityUpgradeFailure(t, err, "visibility_supervisor_unavailable")
			after, err := os.ReadFile(f.file)
			if err != nil {
				t.Fatal(err)
			}
			if string(before) != string(after) || len(f.lookups) != 1 || f.detailCalls != 0 {
				t.Fatal("uncertain lookup continued migration or changed the original binding")
			}
		})
	}
}

func TestCrossPlatformCoverageDingTalkTagConnectExplicitEmptyOperatorRejected(t *testing.T) {
	caller := newSuccessfulConnectCaller(successfulAuthResponse(), `{"result":[],"success":true}`)
	InitDepsForTest(t, caller)
	setupSuccessfulConnectSeams(t)
	testseam.Swap(t, &deapConnectSaveBinding, func(string, digitalEmployeeBinding) error {
		t.Fatal("connect must not persist a binding without an exact operator")
		return nil
	})
	leaf := newConnectTestCommand(t, false)
	if err := leaf.RunE(leaf, nil); err == nil || !strings.Contains(err.Error(), "唯一 operatorOpenDingTalkId") {
		t.Fatalf("empty operator connect error = %v", err)
	}
}
