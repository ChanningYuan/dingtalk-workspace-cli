# RFC：Skill 状态感知与安装完整性（漂移提示 + 依赖闭环）

- **状态**：draft，征求设计评审
- **范围**：`internal/skillstate`、`internal/skillprovenance`、`internal/output` 信封 `_notice`、`dws skill setup` 闭环校验
- **不包含**：启动时自动同步/下载、marketplace 版本比较、Skill overlay、运行时 Skill 内容读取（见 [rfc-skill-content-reader.md](rfc-skill-content-reader.md)）

## 1. 决策摘要

DWS 已经拥有 Skill 状态基础设施：`~/.dws/skills-state.json` 记录版本、官方 Skill 清单和每个托管 Skill 的 sha256 指纹；升级流程会原子重发布内嵌 Skill。缺的是**消费这些事实的两个闭环**：

1. **漂移提示**：命令执行时零网络比对「当前二进制内嵌 Skill」与「已安装 Skill」，不一致就往 JSON 信封的 `_notice.skills` 注入一行机器可读提示，引导用户跑 `dws skill setup`。
2. **依赖闭环**：把产品 Skill 对 `dingtalk-shared` 的依赖从隐含约定升级为 frontmatter 显式声明，setup 完成后 fail-closed 校验闭环，运行时发现半截安装也通过 `_notice.skills` 报告。

原则：

1. 零网络、零阻塞：漂移检查只读本地文件，失败静默降级为不提示，绝不让命令因此失败。
2. 提示走 `_notice` 信封字段，不污染 stdout 业务载荷，不污染 raw 输出。
3. 所有权判断只信 `skills-state.json` 的托管记录，绝不根据目录名前缀猜测用户文件归属。
4. 依赖声明以 Skill frontmatter 为权威，不在 Go 代码里写死产品→shared 的名单。
5. 复用 `skillprovenance` 的 digest 合同（`skill-directory-v1`），不发明第二套指纹。

## 2. 背景：Lark 的做法

Lark 的 `internal/skillscheck` 子系统在每次启动前同步执行（零网络）：

- 读本地 `skills-state.json`，比较记录的 CLI 版本与当前版本；
- 不匹配就挂 StaleNotice，注入 JSON 信封的 `_notice.skills` 字段，统一指向 `lark-cli update`；
- CI、dev 构建、非 release semver、`LARKSUITE_CLI_NO_SKILLS_NOTIFIER` 均跳过；
- `metadata.requires.skills` 声明的硬依赖在 Skill 树组合后校验，缺失即失败（fail-closed，绝不自动放宽）。

它解决的问题是：**用户升级了 CLI 二进制，但 Agent 目录里的 Skill 还是旧版**。此时 Skill 文档里的参数、路由、安全规则与正在执行的命令漂移，Agent 按旧规则调用新命令，产生难以排查的失败。这类漂移的用户可见症状是「命令行为变了」，根因却是「文档没跟上」，没有提示几乎无法自查。

## 3. DWS 当前状态

### 3.1 已有的状态基础设施

`internal/skillstate/state.go` 持久化 `~/.dws/skills-state.json`：

```json
{
  "version": "1.0.61",
  "official_skills": ["dingtalk-calendar", "..."],
  "updated_skills": ["..."],
  "managed_skills": [
    {"name": "dingtalk-calendar", "version": "...", "source": "dws-skill-setup",
     "digest": "sha256:...", "digest_scope": "skill-directory-v1"}
  ],
  "updated_at": "..."
}
```

- `skillprovenance.Record` 为每个托管 Skill 记录版本、来源（`dws-skill-setup` / `dws-upgrade`）和目录内容 sha256；
- `DigestDir` 按路径排序哈希全部常规文件，合同稳定（`skill-directory-v1`）；
- setup 与 upgrade 都会写状态；mono 安装时清理 multi 状态；
- `legacyOfficialSkillNames` 冻结历史官方名，用于安全迁移。

**但消费者只有两个**：`skill setup`（迁移决策）和 `upgrade`（同步决策）。没有任何代码在普通命令执行时读它做漂移判断。

### 3.2 已有的信封注入位

`internal/output/envelope.go:96-98`：

```go
// Notice 是系统级通知注入位（_notice，如升级提示），空则省略。
// 契约仅规定 object 注入位；具体通知 shape 由后续批次定型。
Notice any `json:"_notice,omitempty"`
```

注入位存在且契约明确「shape 由后续批次定型」——本 RFC 就是给 Skill 漂移通知定型。

### 3.3 漂移窗口真实存在

DWS 的升级流程会在 `dws upgrade` 时原子重发布 Skill，缩小了漂移窗口。但以下路径仍会漂移：

| 场景 | 结果 |
|---|---|
| npm/brew/脚本换新二进制，未跑 `dws skill setup` | 磁盘 Skill 是旧版，state 版本是旧版 |
| 用户手动改了 Agent 目录里的 Skill 文件 | 磁盘 digest ≠ state digest |
| setup 中途失败（部分 Skill 已发布） | state 已写但磁盘不完整 |
| 用户删了某个托管 Skill 目录 | 磁盘缺失，state 仍记录托管 |

前两者是版本/内容漂移，后两者是完整性破损。两者都该被感知，但严重级别不同。

### 3.4 依赖关系现状：隐含而非声明

审计全部 multi Skill 的 frontmatter：`metadata.requires` 只声明了 `bins: [dws]`，**没有任何 Skill 声明 `requires.skills`**。但内容合同上产品 Skill 都依赖 `dingtalk-shared`（PREREQUISITE 指向、契约块内嵌二选一，见 `docs/skill-content-framework.md` §2）。

也就是说：「每个产品 Skill 都需要 shared」是一条**文档约定**，不是机器可读事实。setup 目前总是整树安装 multi，闭环天然成立；但一旦未来支持选择性安装、或用户手动删了 `dingtalk-shared`，没有任何机制能发现产品 Skill 的前置已经缺失。

## 4. 目标与非目标

### 4.1 目标

1. 定义 `_notice.skills` 的稳定 JSON shape，覆盖版本漂移、内容漂移、托管目录缺失、依赖缺失四类。
2. 漂移检查零网络、有界耗时，失败一律降级为不提示。
3. 明确跳过规则（CI、dev 构建、环境变量、无状态文件）。
4. 把产品→shared 依赖形式化为 frontmatter `requires.skills` 声明。
5. setup 完成时 fail-closed 校验依赖闭环。
6. 不改变任何业务命令的退出码、stdout 载荷和现有行为。

### 4.2 非目标

- 不在启动时自动下载、同步或修复 Skill；提示只给建议命令。
- 不做 marketplace 远程版本比较。
- 不做 Skill overlay / 用户自定义 Skill 的漂移检测（只检测 DWS 托管记录覆盖的目录）。
- 不检测非托管目录里的同名 Skill（所有权只信 state）。
- 不在本 RFC 中实现 `dws skill list/read`（见 reader RFC）。

## 5. `_notice.skills` 合同

### 5.1 触发条件与检测层级

检测分两级，**先便宜后昂贵**：

**L1：版本快路径（每次命令都跑）**

- 读 `skills-state.json`；文件不存在 → 跳过（见 §5.3 跳过规则）；
- `state.Version != 当前二进制版本` → 产出 `reason: version_mismatch`，不再做 L2。

**L2：指纹慢路径（仅 L1 通过时按需跑）**

L1 版本一致仍可能有内容漂移（用户手改文件）。但 DigestDir 要遍历整棵 Skill 树，不能每次命令都付这个成本。规则：

- 只在 `state.Version` 一致时，对 `managed_skills` 中每个 Skill 重算磁盘 digest，与 state 记录比对；
- 为控制成本，L2 结果可以带 TTL 缓存（如 24h 内只算一次，缓存进 state 文件或独立 cache 文件）；
- digest 不一致 → `reason: content_modified`；托管目录缺失 → `reason: managed_missing`。

v1 可以只落地 L1，把 L2 标记为后续里程碑；但 `_notice.skills` 的 shape 必须一次定型，给四类 reason 都留好位置。

### 5.2 通知 shape

成功与失败信封都可携带，注入位为 `_notice.skills`：

```json
{
  "_notice": {
    "skills": {
      "status": "stale",
      "reason": "version_mismatch",
      "installed_version": "1.0.55",
      "binary_version": "1.0.61",
      "affected": ["dingtalk-calendar", "dingtalk-doc"],
      "action": "dws skill setup",
      "message": "已安装 Skill 与当前 dws 版本不一致，运行 dws skill setup 刷新"
    }
  }
}
```

字段规则：

| 字段 | 说明 |
|---|---|
| `status` | `stale`（可修复的旧版/漂移）或 `incomplete`（托管缺失/依赖缺失） |
| `reason` | `version_mismatch` / `content_modified` / `managed_missing` / `dependency_missing` |
| `installed_version` / `binary_version` | 版本漂移时必填 |
| `affected` | 受影响的托管 Skill 名，排序稳定；无法枚举时省略 |
| `action` | 一条可直接执行的命令，恒为 `dws skill setup` |
| `message` | 面向人类的一句话，可本地化 |

同一命令出现多种问题时，按 `incomplete` 优先于 `stale` 合并为一条；`_notice.skills` 不出现数组形态，避免消费者处理多态。

### 5.3 跳过规则

以下任一条件命中，完全不检测、不提示：

- 环境变量 `DWS_NO_SKILL_NOTIFIER` 非空；
- CI 环境（按现有 CI 探测约定）；
- dev/非 release 构建（版本号不是正式发布形态）；
- `skills-state.json` 不存在（从未 setup 过——这是另一个产品问题，见 §8 备选）；
- 当前命令自身是 `dws skill setup`、`dws upgrade`（避免在修复命令上循环提示）；
- state 文件损坏：记录警告级诊断到 stderr 可选，但不阻塞命令。

### 5.4 注入点

注入发生在信封组装层（`internal/output`），不在各命令 handler 里。检测器是一个纯函数：

```go
func DetectSkillDrift(home string, binaryVersion string) (*SkillNotice, error)
```

- 输入：home、当前版本；
- 输出：nil（无问题/跳过）或一条通知；
- 不持有全局状态，测试用 testseam 替换文件系统与版本。

仅 `--format json`（及 ndjson 等机器格式）输出 `_notice`；人类可读格式可以选择在 stderr 打一行提示，但 v1 建议人类格式也保持静默，只让机器消费者感知，避免打扰交互用户。

## 6. 依赖闭环

### 6.1 声明形式化

给每个 multi 产品 Skill 的 frontmatter 增加：

```yaml
metadata:
  requires:
    bins: [dws]
    skills: [dingtalk-shared]
```

- `dingtalk-shared` 自身不声明 `requires.skills`；
- 这是把现有的文档约定变成机器可读事实，不改变任何 Skill 正文行为；
- `docs/skill-content-framework.md` §2 的 frontmatter 最小集相应更新，把产品 Skill 的 `requires.skills: [dingtalk-shared]` 列为必填。

### 6.2 setup 闭环校验（fail-closed）

`dws skill setup` 安装完成后、写 state 之前：

1. 收集本次安装的所有 Skill 名；
2. 解析每个 Skill 的 `requires.skills`；
3. 任一依赖不在安装集合内 → **setup 失败**，已发布内容按现有事务/备份机制回滚，错误信息指出缺哪个依赖、被谁依赖；
4. 校验通过后把依赖闭包信息一并写入 state（可选字段，供漂移检测复用）。

不允许的兜底：自动补装依赖、降级为警告、允许 `--force` 跳过。依赖缺失是内容发布错误，必须在发布点拦住。

### 6.3 运行时依赖缺失检测

漂移检测器复用 state 里的托管记录：若某个托管产品 Skill 存在而 `dingtalk-shared` 不在托管记录且磁盘上也不存在，产出 `reason: dependency_missing`、`status: incomplete`。这覆盖「用户手动删了 shared」的场景。

## 7. 与现有机制的关系

| 现有机制 | 关系 |
|---|---|
| `skillstate` / `skillprovenance` | 直接复用，只新增「读」消费者；digest 合同不变 |
| `dws upgrade` 原子重发布 | 升级后 state 版本即当前版本，L1 自然通过，提示自动消失 |
| `_notice` 注入位 | 本 RFC 是其第一个定型 shape；后续升级提示可复用同位 |
| 备份/回滚 | setup 闭环校验失败走既有回滚，不新增事务逻辑 |
| reader RFC | 互补：reader 回答「当前二进制里有什么内容」，本 RFC 回答「磁盘上的内容是否就是这个版本」 |

## 8. 备选方案

### 8.1 每次命令都比对磁盘 digest

不采用为默认路径。遍历哈希整棵 Skill 树有真实 I/O 成本，且 99% 的命令执行没有任何漂移。版本快路径先拦，慢路径按需+TTL。

### 8.2 启动时自动重安装

不采用。Lark 也只提示不同步（同步由 `update` 命令显式触发）。自动写用户 Agent 目录超出一条业务命令的合理副作用边界；DWS 有 `dws upgrade` 作为显式刷新入口。

### 8.3 从未安装时提示「请 setup」

本 RFC 不采纳为默认行为。state 文件不存在既可能是「没装」，也可能是「用户只读使用 CLI」。对未安装用户每次提示会制造噪音。可以作为 `dws doctor` 类诊断命令的检查项，而不是 `_notice`。

### 8.4 依赖关系写进 Go 注册表

不采用。`dingtalk-*` 产品集合随内容演进，Go 侧写死名单会产生第二权威并漂移。frontmatter 声明 + QA 门禁（G2 扩展）是唯一权威。

### 8.5 对内容漂移只警告不区分 reason

不采用。`content_modified`（用户改过文件）与 `version_mismatch`（忘跑 setup）的处置不同：前者用户可能有意定制，提示文案不能粗暴建议覆盖。reason 字段一次定型，避免消费者靠解析 message。

## 9. 分阶段落地

### M1：版本漂移提示（L1）

- 实现 `DetectSkillDrift` 的 state 读取 + 版本比较；
- 定型 `_notice.skills` shape（含四类 reason 的枚举，即使 v1 只产出 `version_mismatch`）；
- 接入信封组装层，实现全部跳过规则；
- golden 测试：漂移/一致/无 state/CI/dev/ENV 跳过各分支。

### M2：完整性检测（L2 + 缺失）

- managed 目录缺失检测（`managed_missing`）；
- 带 TTL 的 digest 复算（`content_modified`）；
- 成本上限测试：大 Skill 树下检测耗时有界。

### M3：依赖闭环

- 全部产品 Skill frontmatter 增加 `requires.skills: [dingtalk-shared]`；
- QA 门禁扩展：frontmatter 声明与内容合同一致性；
- setup fail-closed 闭环校验 + 回滚验证；
- 运行时 `dependency_missing` 检测。

## 10. 测试与验收

- 漂移检测器：版本不一致、一致、state 缺失、state 损坏、托管目录缺失、digest 不一致、TTL 缓存命中/过期；
- 信封：`_notice.skills` 出现在成功与失败信封中均合法；无通知时字段缺席；raw/人类格式不携带；
- 跳过规则逐条有测试，特别是 `dws skill setup` 自身不提示；
- 闭环校验：缺 shared 的安装必然失败且回滚完整；依赖齐全时 state 记录闭包；
- 性能：L1 路径仅一次文件读取 + JSON 解析，无目录遍历。

## 11. 结论

DWS 不缺 Skill 状态的**记录**能力——state、指纹、所有权、原子发布都已在生产。缺的是**消费**：把这些事实变成用户和 Agent 可见的漂移信号，以及把文档里的隐含依赖变成可校验的发布门禁。本 RFC 的两个机制都只消费现有事实、复用现有注入位和事务机制，是低成本、高排障价值的补全。
