# RFC：Go MethodDef + corecmd 统一命令构建框架

- **状态**：已合并 —— 内容并入 [`rfc-command-framework-convergence.md`](rfc-command-framework-convergence.md) **§5.12 Service Method Builder 扩展**
- **本文件保留为指针**，避免两份并行框架 RFC 漂移

## 合并去向

| 本文档原章节 | 归宿 |
|---|---|
| MethodDef 结构（Identity/Input/Contract/Safety/Execution） | §5.12.1（拆为 MethodDefinition + MethodBinding，对齐 §5.1 Definition/Executable） |
| ServiceBuilder 责任/校验/编译结果 | §5.12.2 |
| 执行口子（Tool/Invoke/Orchestrate/RunE 边界） | §5.12.1、§5.12.3 |
| dingtalk-dev 首个 Provider | §5.12.4 |
| 迁移与兼容 | §5.12.4（兼容策略沿用主 RFC §6/§9） |
| 测试与验收（parity 三层） | §5.12.4 parity 门禁 |
| 非目标/风险控制 | §5.12.2、§5.12.3 及主 RFC §10 |
| 参考实现（lark/WeCom/gh） | 主 RFC §1 决策摘要与本文档下文 |

## 关键设计决定（合并时固化）

1. **`corecmd.Spec` 是编译产物/IR，不是 service authoring source**：MethodDefinition → CompileMethod → `corecmd.Spec` → `corecmd.New`。
2. **不新增 `CommandDescriptor` 或第二套 Registry**：ContractFinal 仍从真实 Cobra leaf 收集；现有 Effective/Bound/Schema Registry 管线不变。
3. **执行三选一，基数恰好为 1**：Tool（标准单次，框架装默认 Invoke）/ Invoke（受 §5.12.3 硬边界约束）/ Orchestrate（多步流程）；`RunE` 仅 legacy 迁移。
4. **Builder 不自行 `PrepareCommandTree`**：app root 工厂仍是唯一准备点。
5. **Group 一律经现有 `ApplyGroupPolicy`**，不重复实现 group 行为。
6. **Catalog 是消费者不是权威**：未来 MethodDefinition 可导出 normalized Catalog Snapshot 供 lint/diff，但运行时 Schema 仍由 live Cobra tree 管线生成。

## 参考实现（保留备查）

- [larksuite/cli](https://github.com/larksuite/cli)：service method builder、`extension/command.Definition`、Hooks、统一 output/runtime。
- [WecomTeam/wecom-cli](https://github.com/WecomTeam/wecom-cli)：ServiceSchema builder、CustomCommand、Helper 注入和统一 Client 组装。
- [cli/cli](https://github.com/cli/cli)：Factory/Options/Run 和 raw API 逃生舱的传统 CLI 边界（DWS 不作为 service 主模式）。

**新工作请直接修改 [`rfc-command-framework-convergence.md`](rfc-command-framework-convergence.md) §5.12；本文件不再承接增量。**
