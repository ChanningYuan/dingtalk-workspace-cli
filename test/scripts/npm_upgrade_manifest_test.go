package scripts_test

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestReleaseNpmStagingPublishesVerifiedUpgradeManifest(t *testing.T) {
	sourceRoot, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	dist := t.TempDir()
	sums := ""
	for _, name := range []string{"dws-darwin-arm64.tar.gz", "dws-skills.zip"} {
		data := []byte("fixture-" + name)
		mustWriteFile(t, filepath.Join(dist, name), data, 0600)
		sums += fmt.Sprintf("%x  %s\n", sha256.Sum256(data), name)
	}
	mustWriteFile(t, filepath.Join(dist, "checksums.txt"), []byte(sums), 0600)
	run := func() ([]byte, error) {
		cmd := exec.Command("sh", filepath.Join(sourceRoot, "scripts/release/stage-npm-package.sh"), "v1.2.3")
		cmd.Env = append(os.Environ(), "DWS_PACKAGE_SOURCE_ROOT="+sourceRoot, "DWS_PACKAGE_DIST_DIR="+dist)
		return cmd.CombinedOutput()
	}
	if out, err := run(); err != nil {
		t.Fatalf("stage: %v\n%s", err, out)
	}
	manifestPath := filepath.Join(dist, "npm/dingtalk-workspace-cli/package.json")
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	var manifest struct {
		Upgrade struct {
			Version int `json:"version"`
			Files   map[string]struct {
				URLs   []string `json:"urls"`
				SHA256 string   `json:"sha256"`
				Size   int64    `json:"size"`
			} `json:"files"`
		} `json:"dwsUpgrade"`
	}
	if err := json.Unmarshal(data, &manifest); err != nil {
		t.Fatal(err)
	}
	if manifest.Upgrade.Version != 1 || len(manifest.Upgrade.Files) != 2 {
		t.Fatalf("升级清单=%s", data)
	}
	for name, asset := range manifest.Upgrade.Files {
		want := []byte("fixture-" + name)
		if asset.Size != int64(len(want)) || asset.SHA256 != fmt.Sprintf("%x", sha256.Sum256(want)) || len(asset.URLs) != 1 || asset.URLs[0] != "https://github.com/DingTalk-Real-AI/dingtalk-workspace-cli/releases/download/v1.2.3/"+name {
			t.Fatalf("清单与资产不一致: %+v", asset)
		}
	}
	if out, err := run(); err != nil {
		t.Fatalf("restage: %v\n%s", err, out)
	}
	again, _ := os.ReadFile(manifestPath)
	if string(again) != string(data) {
		t.Fatal("重复生成结果不确定")
	}
	mustWriteFile(t, filepath.Join(dist, "dws-skills.zip"), []byte("tampered"), 0600)
	if out, err := run(); err == nil || !strings.Contains(string(out), "release checksum mismatch") {
		t.Fatalf("损坏资产未阻止发布清单生成: %v\n%s", err, out)
	}
}
