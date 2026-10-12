# RFC：为 DWS 增加版本锁定的 Skill 内容读取器

- **状态**：draft，征求设计评审
- **范围**：`dws skill` 内容发现与只读读取、内嵌 Skill 内容、Help/Schema 关联
- **不包含**：Pi 原生 runtime、Skill 执行引擎、脚本执行、任意本地目录浏览、Skill 安装行为改造

## 1. 决策摘要

DWS 增加一组面向 Agent、调试工具和文档导航的只读命令：

```text
dws skill list
dws skill list dingtalk-calendar/references
dws skill read dingtalk-calendar
dws skill read dingtalk-calendar references/calendar.md
```

命令默认读取**当前运行的 DWS 二进制内嵌的 Skill 内容**，而不是 GitHub `main`、当前工作目录或用户已经安装的 Skill。这样，CLI 返回的说明、Help 中的关联 Skill 和实际执行命令来自同一个版本。

设计采用以下原则：

1. `list` 用于渐进式发现，`read` 用于按需读取一个 Skill 或一个 reference。
2. 由内容组装层白名单决定哪些文件进入 reader；生产 reader 只接收 `SKILL.md` 和文档型 reference，不接收脚本、二进制资源和资产。
3. `list` 永远输出 JSON；`read` 默认输出原始 Markdown，`--json` 输出 JSON 信封。
4. 路径只允许在 multi Skill 树内解析，拒绝绝对路径、`..`、目录读取和不存在的资源。
5. 读取器只读，不安装、不解压到用户目录、不启动 Agent、不执行 Skill 中的命令。
6. 复用已有的 `EmbeddedSkills` 和 mono/multi 内容合同，不建立第二套 Skill 元数据系统。

## 2. 背景：Lark 为什么提供 `skills list/read`

Lark 的 `skills list/read` 表面上只是两个文件读取命令，实际解决的是 Agent CLI 中的四个框架问题。

### 2.1 让二进制与 Skill 内容保持同版本

如果命令只输出 GitHub 链接，链接指向的是不断变化的分支，而不是当前正在执行的 CLI 版本。一个旧版 CLI 可能把 Agent 引导到新版参数、路径和行为；一个新版 CLI 也可能继续引用已被删除的文档。

把 Skill 内容编译进二进制后，CLI 可以直接返回与自身匹配的合同：

```text
当前 CLI 版本 -> 当前命令 Schema -> 当前 Help -> 当前 Skill 内容
```

这比在输出中拼接一个移动的远程 URL 更适合作为 Agent 的事实来源。远程链接仍可作为人类阅读入口，但不应是运行时唯一的内容来源。

### 2.2 支持渐进式披露，而不是一次加载整个 Skill 树

Skill 通常包含入口协议、产品规则、工作流和 reference。启动时把所有 Markdown 注入 Agent 会增加上下文成本，也会让不相关的产品规则影响路由。

`list/read` 将加载过程拆成三个阶段：

```text
list        -> 找到候选 Skill 或 reference
read        -> 读取入口合同或一个具体 reference
执行        -> 由 Agent 按 Skill 合同决定后续动作
```

这使 Skill 树既能被人浏览，也能被 Agent 通过稳定接口按需导航。

### 2.3 提供离线、无认证、无网络依赖的文档查询

Skill 内容读取不应因为网络、登录状态、代理或仓库权限失败。它是本地 CLI 的描述能力，不是外部内容抓取能力。

将内容放入 `fs.FS` 并通过只读 reader 暴露，能够让：

- Agent 在没有网络时继续获得操作规则；
- 测试使用固定内容验证输出；
- 插件或宿主替换内容源而不改变命令合同；
- 命令不需要给文档读取器额外的 DingTalk 权限。

### 2.4 把 Skill 引用变成框架能力

Skill 不仅是安装到某个 Agent 目录的文件，也是命令发现和帮助系统的一部分。命令的 Help、Schema affordance 和 Agent metadata 已经知道某条命令关联哪些 Skill；`read` 使这些引用可以在当前二进制内闭环解析，而不是跳出到一个可能漂移的外部页面。

因此，Lark 的这项能力不是在实现一个新的 Agent runtime。它提供的是一个版本一致的、只读的 Skill 内容 API；真正的路由、确认、后端调用和脚本生命周期仍由既有框架负责。

## 3. DWS 当前状态

### 3.1 已有内嵌内容源

DWS 根包通过 `embed.FS` 内嵌两棵内容树：

```text
skills/mono/
skills/multi/
```

`EmbeddedSkills` 已经是发布二进制的内容源，`dws skill setup` 在没有显式 `--source` 或环境变量时，会从当前二进制提取所选模式，再复用已有的目录安装逻辑。因此，DWS 已具备“内容随二进制锁定”的基础设施。

读取器不应再次从工作目录加载相同内容，也不需要先调用 `skill setup`。它可以直接在 `EmbeddedSkills` 上实现 `fs.FS` reader，避免临时目录和文件系统副作用。

### 3.2 已有 mono/multi 内容合同

- `multi` 是面向公共 Agent Skills 发现和默认安装的产品 Skill 树。
- `mono` 是兼容旧面和全局协议的单 Skill 树，带有 `metadata.internal: true`，不作为公共多 Skill 列表展示。
- 每个 multi Skill 以 `<name>/SKILL.md` 为入口，可包含 `references/` 和可选 `scripts/`。
- reference 是渐进式读取的文档依赖；脚本是安装后由 Skill 合同引用的机器资源。

读取器必须尊重这棵内容合同，而不是把所有嵌入文件都当成同一种可供 Agent 直接消费的文档。

### 3.3 当前缺口

当前 `dws skill` 已有安装、下载、搜索和 setup 子命令，但没有：

- 列出当前二进制包含哪些 Skill；
- 读取与当前 CLI 版本匹配的 `SKILL.md`；
- 读取一个具体 reference 而无需访问仓库或安装目录；
- 用机器可解析的格式返回 Skill 名称、描述、版本约束和元数据；
- 将 Help/Schema 中的 Skill source reference 解析为本地内容。

当前产品 Help 中的 `SkillDocumentation` 生成 GitHub `main` 分支链接。这对人类继续阅读仍有价值，但不能替代版本锁定的本地内容读取。

## 4. 目标与非目标

### 4.1 目标

1. 为当前二进制内嵌的 multi Skill 提供稳定的 list/read CLI 合同。
2. 支持 Agent 的渐进式发现和读取，不要求网络、登录或预先安装。
3. 保证 raw Markdown 输出没有额外日志、标题或包装文本污染。
4. 提供机器可解析的 JSON 输出，包含内容来源和版本信息。
5. 复用现有 mono/multi 目录合同、frontmatter 和 `EmbeddedSkills`。
6. 将命令关联的 Skill source reference 与本地内容读取对齐。
7. 对路径穿越、任意文件读取和脚本资源暴露建立明确边界。

### 4.2 非目标

- 不实现 `--channel pi` 或任何 Pi 原生 Agent runtime。
- 不实现 Skill 的解释器、路由器、工作流引擎或执行沙箱。
- 不通过 `read` 自动安装、升级、解压或修改用户的 Agent Skill 目录。
- 不扫描 `~/.agents/skills`、`~/.pi/agent/skills` 或其他本地安装目录。
- 不把 `metadata.requires.skills` 自动展开为多次读取。
- 不暴露 mono 兼容树；reader v1 只支持 multi 公共 Skill 面。
- 不将脚本、二进制、图片或其他资产纳入文档读取 API。
- 不在本 RFC 中修改 marketplace 安装协议和 Agent 路径注册表。

## 5. 命令合同

### 5.1 命令命名

沿用现有的单数命名空间 `dws skill`，不新增平行的 `dws skills`：

```text
dws skill list [path] [flags]
dws skill read <skill> [relative-path] [flags]
```

v1 只支持 multi 公共 Skill 面，不提供 `--mode` 开关。mono 是带 `metadata.internal: true` 的兼容树，不属于公共 Agent 内容 API；未来若确有调试需求，再作为显式 mode 评审加入。

建议 flags：

```text
--json               read 输出 JSON；list 接受但无语义变化
```

`list` 始终输出 JSON，接受 `--json` 只是为了与 `read` 保持参数对称；`read` 不带 `--json` 时输出原始 Markdown。

### 5.2 `skill list`

无参数时列出所选内容树下的 Skill：

```text
dws skill list
```

指定 Skill 时列出该 Skill 的一层内容：

```text
dws skill list dingtalk-calendar
dws skill list dingtalk-calendar/references
```

该行为与目录发现保持一致：

- Skill 根目录返回入口文件和一级目录；
- 目录路径只返回直接子项，不递归倾倒整个 Skill 树；
- list 不返回文件正文；
- 默认不列出 `scripts/`、二进制和资产。

`list` 的 JSON 合同分为两种：

```json
{
  "ok": true,
  "skills": [
    {
      "name": "dingtalk-calendar",
      "description": "...",
      "version": "...",
      "metadata": {}
    }
  ],
  "count": 1
}
```

指定目录时返回一层目录项，不递归读取正文：

```json
{
  "ok": true,
  "path": "dingtalk-calendar/references",
  "entries": [
    {"path": "dingtalk-calendar/references/calendar.md", "is_dir": false}
  ],
  "count": 1
}
```

目录项必须带 Skill 前缀，结果按路径稳定排序，因此可以直接作为后续 `read` 的输入。

### 5.3 `skill read`

读取 Skill 入口：

```text
dws skill read dingtalk-calendar
```

等价于读取：

```text
skills/multi/dingtalk-calendar/SKILL.md
```

读取一个文档型 reference：

```text
dws skill read dingtalk-calendar references/calendar.md
dws skill read dingtalk-calendar/references/calendar.md
```

两种写法必须解析为同一个目标；单参数形式在第一个 `/` 处分割，双参数形式分别传入 Skill 名称和相对路径。`relative-path` 相对于 Skill 根目录解析，不允许把 `skills/multi` 或操作系统路径传给命令。读取目录或未进入 reader 内容源的资源时返回明确错误，而不是把目录内容或二进制字节写到 stdout。

### 5.4 Raw Markdown 输出

`read` 默认输出文件的原始字节内容：

- 不添加标题、代码围栏、路径前缀或诊断信息；
- 正文写入 stdout；
- 错误写入 stderr；
- 保留 Skill 原有换行和 frontmatter；
- 成功退出码为 0，读取失败使用现有 CLI 错误/退出码合同。

raw 模式是给 Agent 和管道使用的主要模式。调用方可以把结果直接作为下一轮上下文，而不需要剥离 CLI 包装。

读取入口 `SKILL.md` 时，可以在 stderr 提供一条由 CLI 生成的 guidance，说明：

- 当前 Skill 的其他文件应通过 `dws skill read <skill> <relative-path>` 读取；
- 引用其他 Skill 时应改为以目标 Skill 为根的 `dws skill read <other-skill> <path>`；
- guidance 不属于文件正文，不得写入 raw stdout。

读取 reference 时不额外输出 guidance。`read --json` 可以把同一 guidance 放入可选的 `guidance` 字段，保持正文 `content` 不被污染。这是 Lark 设计中很重要的一点：它把 Agent 的导航提示与版本锁定的文件内容分开。

### 5.5 JSON 输出

`list --json` 的单项至少包含：

```json
{
  "name": "dingtalk-calendar",
  "description": "...",
  "path": "dingtalk-calendar",
  "metadata": {
    "category": "product",
    "requires": {
      "bins": ["dws"]
    },
    "cli_version": ">=0.2.14"
  },
  "bundle_version": "..."
}
```

`read --json` 返回：

```json
{
  "skill": "dingtalk-calendar",
  "path": "SKILL.md",
  "content": "---\nname: dingtalk-calendar\n...",
  "metadata": {
    "category": "product",
    "cli_version": ">=0.2.14"
  },
  "bundle_version": "..."
}
```

字段规则：

- `name` 是 frontmatter 中的 Skill 名称，必须与目录名一致；
- `description` 来自入口 frontmatter；
- `metadata` 保留已解析的 frontmatter metadata，不另建平行字段表；
- `bundle_version` 是当前运行 DWS 二进制的版本；
- `skill_version` 只有在 Skill 明确声明独立版本时才出现，不为当前内容伪造版本号；
- `path` 使用 `/`，与 `embed.FS` 和公开 Skill 路径合同一致；
- JSON 字段名、错误类型和成功/失败语义纳入测试 golden。

如果现有 Skill 只有 `cli_version` 约束而没有独立 `version`，这不阻碍读取。reader 应保留现有 frontmatter 的兼容写法，包括 mono 使用的顶层 `cli_version` 和 multi 常用的 `metadata.cli_version`；引入独立 Skill 版本属于后续内容合同变更，不作为本 RFC 的隐式副作用。

## 6. 内容边界与安全规则

### 6.1 只读嵌入式源

生产命令只绑定由 `EmbeddedSkills` 的 `skills/multi` 子树构建出的文档白名单视图。读取器不接受用户提供的绝对路径，不根据当前工作目录猜测 Skill 源，也不跟随已安装 Agent 目录。

内部实现可以把 reader 抽象为 `fs.FS`，以便单元测试注入固定树；这不等于对外开放任意 `--source`。开发者若要测试未构建的内容，应使用现有 source/setup 开发流程，而不是让生产读取命令变成文件浏览器。

### 6.2 路径规范化

输入路径在解析前统一为 `/` 分隔，并执行以下拒绝规则：

- 空路径只能表示 Skill 根目录的 list，不能表示 read；
- 拒绝绝对路径；
- 拒绝任何 `..` 路径段；
- 拒绝规范化后逃出当前 Skill 的路径；
- 拒绝 `.` 以外的隐式路径穿越变体；
- 拒绝目录作为 read 目标；
- 拒绝不存在的 Skill、文件或目录；
- 不把输入直接交给操作系统路径拼接后读取。

由于源是 `embed.FS`，实现还应拒绝超出允许内容树的路径，而不是仅依赖宿主操作系统的路径行为。

### 6.3 文档与机器资源分离

DWS Skill 中存在供安装后运行的 `scripts/`。这些脚本可能涉及认证、写操作、分页恢复或数据完整性规则；它们不是文档 reader 的执行对象，也不应因 `list/read` 而被解释。

Lark 的实现把这条边界放在 `go:embed` 白名单，而不是放在通用 reader：生产 embed 只包含 `SKILL.md`、`references/` 以及明确列出的文档目录；`assets/`、`scripts/` 根本不进入 reader 的 `fs.FS`。reader 本身只负责安全地列目录和读文件。

DWS 的安装用 `EmbeddedSkills` 目前包含完整 mono/multi 树，可能包含脚本。因此不能把安装用 FS 原样传给新的 reader。建议增加一个面向读取的文档视图：

```text
安装源：EmbeddedSkills
  ├── skills/mono/**
  └── skills/multi/**

读取源：EmbeddedSkillDocs（白名单视图）
  ├── <skill>/SKILL.md
  └── <skill>/references/**（以及未来明确登记的文档目录）
```

| 内容 | list | read | 说明 |
|---|---:|---:|---|
| `SKILL.md` | 是 | 是 | Skill 入口合同 |
| `references/**` | 是 | 是 | 渐进式文档，由读取源白名单保证 |
| 明确登记的文档目录 | 是 | 是 | 例如未来的 `routes/`、`scenes/` |
| `scripts/` | 否 | 否 | 不进入读取源 |
| 二进制、图片、资产 | 否 | 否 | 不进入读取源 |

这样 reader 不需要知道什么是脚本，也不需要额外的 `resource_not_documentation` 分支；内容安全边界由构建期/组装期白名单和 embed 完整性测试保证。若未来 reader 支持插件或 overlay，插件提供的也是已经经过同一内容政策验证的 `fs.FS`。

当 Markdown 引用一个脚本时，`read` 返回的正文仍保留原引用。调用方若需要使用该脚本，应遵循现有 Skill setup/安装合同；reader 不替它下载、展开或启动脚本。

## 7. 内容范围与安装行为

### 7.1 只读 multi 树

reader 的唯一内容根是 `skills/multi`：

```text
dws skill list
```

`dingtalk-shared` 作为共享 Skill 正常列出，因为它是 multi 内容树的一部分。

### 7.2 mono 不在读取范围

mono 是兼容旧 Agent 面的单 Skill 树，frontmatter 明确标记 `metadata.internal: true`，公共 Agent Skills 发现本就将其隐藏。reader 不为其提供任何入口：

- 没有 `--mode mono`；
- mono 的 `dws` 入口名不出现在 `dws skill list` 结果中，`dws skill read dws` 返回 `unknown_skill`；
- mono 内容继续只通过 `dws skill setup --mode mono` 的安装路径触达。

这样可以避免公共 Agent 把 mono mega-skill 与产品 Skill 重复加载，也让 reader 合同只有一个内容根，不需要 mode 字段参与寻址。

### 7.3 不改变 setup

`skill list/read` 与 `skill setup` 是不同能力：

- setup 把内容复制或链接到 Agent 可发现目录；
- list/read 查询当前二进制携带的内容；
- read 不保证用户目录已经安装该 Skill；
- setup 不需要改造成通过 CLI read 逐文件输出。

两者共享同一 `EmbeddedSkills` 内容源和相同的 mono/multi 目录合同，以避免内容分叉。

## 8. 与 Help、Schema 和 Agent metadata 的关系

DWS 已经拥有三类 Skill 关联事实：

- ProductDecl 的 `HelpReferences.RelatedSkills`；
- Help 中的 `SkillDocumentation`；
- Schema/catalog 中的 `agent_source_refs`。

本 RFC 不新增一套关联注册表，而是让已有引用可以指向本地内容：

1. Help 继续可以展示 GitHub 链接，作为人类可点击的外部入口。
2. Help 同时可展示等价的本地提示，例如 `dws skill read dingtalk-calendar`。
3. `agent_source_refs` 的 Skill path 应使用可被 reader 解析的规范相对路径。
4. Schema 只记录来源引用和路径，不把整份 Markdown 嵌进每个 Schema 响应。
5. Agent metadata 生成器可以在需要时把 source reference 解析为当前版本内容，但不得在每条命令发现时无条件加载全部 references。

这样，Help/Schema 仍然由现有 Contract、ProductDecl 和 registry 驱动；reader 只是为这些来源引用提供一个版本一致的解析端点。

建议的规范引用形态为：

```text
skill://multi/dingtalk-calendar/SKILL.md
skill://multi/dingtalk-calendar/references/calendar.md
```

CLI path 与 `skill://` 引用都应经过同一个 resolver，避免一套路径规则用于 Help、另一套用于命令行。

## 9. 实现形态

本 RFC 不要求立即实现，但建议实现拆为以下小组件：

1. `internal/skillcontent`：定义注入 `fs.FS` 的 reader、Skill summary、目录项和错误类型。
2. `EmbeddedSkillDocs` adapter：从 `EmbeddedSkills` 的 `skills/multi` 子树构建 reader 使用的文档白名单视图；安装源和读取源保持分离。
3. frontmatter parser：复用项目现有解析能力；不得为了 list 创建另一套 metadata 语法。
4. embed/content QA：验证 scripts/assets 未进入读取源，且所有公开 Skill 都有入口文件。
5. `dws skill list/read` Cobra 命令：只负责参数解析、格式选择和 stdout/stderr。
6. Help/Schema resolver：把已有 source reference 映射到规范 Skill path，不在 reader 中反向扫描命令注册表。

读取器应避免调用 `materializeEmbeddedSkillSource`。setup 之所以展开到临时目录，是为了复用安装器的目录复制逻辑；read 只需要 `fs.ReadFile`、`fs.ReadDir` 和安全的相对路径校验。

## 10. 错误合同

至少定义以下可区分的错误类别：

| 错误 | 触发条件 |
|---|---|
| `unknown_skill` | Skill 不存在 |
| `invalid_path` | 绝对路径、`..` 或其他非法路径 |
| `not_found` | Skill 内文件不存在 |
| `directory_not_readable` | read 目标是目录 |
| `resource_not_documentation` | 目标是脚本、二进制或资产 |
| `invalid_skill_metadata` | `SKILL.md` frontmatter 无法解析或与目录名冲突 |

错误正文可以面向人类，但 `--json` 必须返回稳定的错误类型、消息和退出码映射。不得在 raw Markdown stdout 中混入错误文本。

## 11. 测试与验收标准

### 11.1 reader 单元测试

- multi 根目录列出所有公开 Skill；
- 指定 Skill 只列出一层目录；
- `SKILL.md` 返回完整原文；
- reference 返回完整原文；
- frontmatter 的 name、description、metadata 正确解析；
- bundle version 出现在 JSON 中；
- mono 树不出现在 list 中，`dws skill read dws` 返回 `unknown_skill`；
- 未知 Skill、未知文件和目录 read 返回对应错误；
- 绝对路径、`..`、混合分隔符和规范化后越界的路径全部拒绝；
- scripts、二进制和资产不会出现在文档列表中，也不能被 read；
- raw 输出不包含 CLI 包装；
- JSON 输出通过 golden 验证。

### 11.2 命令集成测试

- `dws skill list` 不访问网络、不需要登录、不依赖当前工作目录；
- `dws skill read` 不创建临时目录、不修改用户 Skill 安装目录；
- `--json` 的 stdout 是单一合法 JSON 值；
- 诊断和错误不污染 raw stdout；
- `agent_source_refs` 中的规范引用能够解析到对应内嵌文件；
- Help 展示的本地读取提示与实际命令路径一致；
- setup 使用的内容与 read 使用的内容来自同一 `EmbeddedSkills` 树。

### 11.3 内容质量门禁

现有 mono/multi 内容质量检查继续负责：

- Skill 目录形状；
- frontmatter；
- mono/multi 覆盖关系；
- reference 链接可达性；
- 脚本 orphan 检查。

reader 测试不替代这些门禁，只验证“被允许读取的内容”能以稳定合同被发现和返回。

## 12. 兼容性与发布

### 12.1 对现有命令的兼容性

这是新增子命令，不改变现有 `skill setup`、`skill install`、`skill get` 和 `skill search` 的参数或输出。现有 Agent 安装路径，包括 Pi 的 `~/.pi/agent/skills`，不需要因为 reader 增加新的注册表条目。

### 12.2 对 Skill 内容的兼容性

新增 Skill 时必须同时满足现有内容合同和 reader 的内容分类规则。一个 Skill 可以包含脚本而不暴露脚本，但它的 Markdown 必须明确说明脚本属于安装后执行资源，不能暗示 `dws skill read` 会运行它。

### 12.3 版本锁定策略

reader 返回的是运行中二进制的内容。升级 DWS 二进制后，`list/read` 看到的 Skill 内容也随之升级；这与 setup 的当前行为一致。

独立的 Skill `version` 字段、远程 marketplace 版本比较和缓存失效策略不在本 RFC 中。当前 `metadata.cli_version` 继续作为 Skill 对 CLI 的最低兼容约束。

## 13. 备选方案

### 13.1 只返回 GitHub 链接

不采用。链接适合人类跳转，但会漂移、需要网络，并且不能作为当前二进制的确定性事实来源。

### 13.2 先 setup，再读取用户安装目录

不采用。它引入文件系统副作用，读取的可能是旧版本或用户修改后的内容，无法证明与当前 CLI 一致。

### 13.3 扫描任意本地 Skill 目录

不采用。任意目录扫描扩大了权限和路径安全边界，也会把“当前 DWS 官方内容”和“用户自定义覆盖”混成一个来源。用户自定义 Skill 的生命周期应由安装/overlay 能力单独定义。

### 13.4 把所有嵌入文件都作为 read 资源

不采用。脚本和资产不是文档 API；无边界暴露会把安装资源、机器资源和 Agent 上下文混为一谈，并为未来的执行误用留下接口暗示。

### 13.5 新建 `dws skills` 顶层命令

不采用。现有命令空间已经是 `dws skill`，安装、搜索和 setup 都在其中。增加复数命名空间只会让调用方猜测两个命令是否有不同的内容源。

## 14. 分阶段落地建议

### M1：只读内嵌内容

- 实现 `internal/skillcontent` reader 和安全路径校验；
- 增加 `dws skill list/read`，内容根固定为 `skills/multi`；
- 支持 raw 和 JSON；
- 完成 reader 与命令 golden 测试；
- 不改 Help/Schema。

### M2：来源引用闭环

- 统一 `agent_source_refs`、Help 本地提示和 reader 的规范路径；
- 为相关 Help 增加 `dws skill read ...` 提示；
- 验证 Schema/metadata 只按需解析内容。

### M3：内容合同强化

- 签入文档资源分类或 manifest；
- 为新增 Skill 的文档/脚本边界增加 QA；
- 再评估是否需要独立 Skill version 字段。

## 15. 与 Lark 实现的代码级校准

Lark 当前没有发现一份独立的 Skill reader RFC；它的设计事实分散在命令实现、`internal/skillcontent`、embed 声明、overlay/ref resolver、测试和 README 中。因此，本 RFC 以这些可运行合同为对照，而不是只模仿命令名称。

| Lark 的实际设计 | DWS 的采用方式 | 设计含义 |
|---|---|---|
| `skills list` 永远 JSON，带 `ok`、`count`、`skills/entries` | list 采用同样的机器合同；保留 `dws skill` 单数命名 | list 首要消费者是 Agent，不是终端人类 |
| `skills read` 默认 raw，入口 Skill 的 guidance 写 stderr | DWS 保留 raw stdout 纯净，并增加独立 guidance | 导航提示不能污染 Skill 正文 |
| reader 接收注入的 `fs.FS` | DWS reader 也只依赖抽象 FS | 测试、宿主和未来 overlay 不绑定 embed 实现 |
| 生产 embed 用白名单排除 scripts/assets | DWS 为 reader 建立 `EmbeddedSkillDocs` 视图，不能直接复用完整安装树 | 文档边界属于内容组装层 |
| frontmatter 提供 description/version/metadata | DWS 透传已有 frontmatter；独立 `version` 逐步补齐 | 不通过 CLI 代码重新定义 Skill 元数据 |
| overlay 后的最终 Skill 树供 list/read 使用 | DWS v1 先读官方内嵌树，但保留最终 FS/resolver 接缝 | 未来插件不能只改 Help 而不改 reader |
| canonical Skill reference 经 resolver 重映射 | DWS 用 `skill://` 规范引用并统一 resolver | Help、Schema、错误 guidance 使用同一版本路径 |
| reader、命令、overlay 都有独立测试 | DWS 同时测试内容源、reader、命令和 source reference | 仅测试命令输出不足以证明版本一致 |

特别需要保留的不是 Lark 的具体目录名，而是它的分层：

```text
Skill 内容合同
    ↓
内容组装/白名单/overlay
    ↓
注入式 fs.FS reader
    ↓
list/read 命令
    ↓
Help、Schema、Agent guidance
```

这条分层可以让 DWS 继续拥有自己的 mono/multi 内容树，同时获得 Lark 已验证的版本一致性和渐进式读取能力。

## 16. 结论

Lark 做 `skills list/read` 的核心原因不是“再做一个文件浏览器”，而是把 Skill 变成与 CLI 二进制同版本、可离线发现、可渐进读取的框架内容。DWS 已经拥有最关键的底座：`EmbeddedSkills`、mono/multi 内容合同、Help/Schema Skill 引用和 setup 的版本锁定行为。

因此 DWS 的最小合理实现是一个基于**文档白名单 FS**的只读内容 reader，而不是直接暴露完整安装树。它补齐的是 Skill 内容发现和版本一致性，不改变 Skill 安装能力，不实现 Pi runtime，也不把 DWS 引向一个新的 Skill 执行引擎。
