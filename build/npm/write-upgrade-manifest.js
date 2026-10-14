// Copyright 2026 Alibaba Group
// Licensed under the Apache License, Version 2.0

const fs = require("fs");
const path = require("path");
const crypto = require("crypto");

// 校验实际打包字节，再把平台资产描述写入精确版本的 npm 清单。
function writeUpgradeManifest(packageRoot) {
  const manifestPath = path.join(packageRoot, "package.json");
  const manifest = JSON.parse(fs.readFileSync(manifestPath, "utf8"));
  const assets = path.join(packageRoot, "assets");
  const sums = new Map();
  for (const line of fs.readFileSync(path.join(assets, "checksums.txt"), "utf8").split(/\r?\n/)) {
    if (!line.trim()) continue;
    const match = /^([a-fA-F0-9]{64})\s{2}(\S+)$/.exec(line);
    if (!match || sums.has(match[2])) throw new Error("invalid or duplicate release checksum");
    sums.set(match[2], match[1].toLowerCase());
  }
  const files = {};
  for (const name of fs.readdirSync(assets).sort()) {
    if (!/^dws-(?:(?:darwin|linux)-(?:amd64|arm64)\.tar\.gz|windows-(?:amd64|arm64)\.zip|openharmony-arm64\.tar\.gz|skills\.zip)$/.test(name)) continue;
    const data = fs.readFileSync(path.join(assets, name));
    if (data.length === 0 || data.length > 128 * 1024 * 1024) throw new Error(`invalid release asset size: ${name}`);
    const sha256 = crypto.createHash("sha256").update(data).digest("hex");
    if (sums.get(name) !== sha256) throw new Error(`release checksum mismatch: ${name}`);
    files[name] = {
      urls: [`https://github.com/DingTalk-Real-AI/dingtalk-workspace-cli/releases/download/v${manifest.version}/${name}`],
      sha256,
      size: data.length,
    };
  }
  if (!files["dws-skills.zip"] || Object.keys(files).length < 2) throw new Error("missing upgrade assets");
  manifest.dwsUpgrade = { version: 1, files };
  fs.writeFileSync(manifestPath, JSON.stringify(manifest, null, 2) + "\n");
}

if (require.main === module) writeUpgradeManifest(process.argv[2]);
module.exports = { writeUpgradeManifest };
