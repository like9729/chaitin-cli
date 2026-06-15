# SafeLine-3 CLI 命令分层

本文档定义 `safeline-3` CLI 的命令分层。当前阶段只实现实体命令和 `raw request`，短命令先不实现。

## 分层结论

目标分层：

```text
短命令        高频、短路径、默认参数明确的快捷入口
实体命令      当前主入口，命令对应产品实体或稳定查询域
raw request   API 逃生口
```

当前阶段：

```text
实体命令      已实现/继续完善
raw request   已实现/继续保留
短命令        暂不实现，只保留设计方向
```

## 实体命令

实体命令是当前 CLI 的主干。

要求：

- 命令应尽量对应 SafeLine-3 产品实体。
- 参数使用业务语义，而不是通用 `--query` / `--request`。
- 必填参数、条件必填参数、枚举、格式必须在 help 中说明。
- 写操作必须要求 `--yes`，并支持 `--check` 或根级 `--dry-run`。
- 影响安全和兼容性的规则必须在 CLI 执行层校验，不能只依赖 skill。

当前实体/领域命令：

```text
node-group
site
listener
ip-group
policy-group
policy-rule
acl template
acl rule
log
system
network
monitor
```

说明：

- `site` 是防护对象的 CLI 名称，保留短命名。
- `log` 是日志查询域，保持简单命名。
- `system`、`network`、`monitor` 是运维查询域，不强行拆成更长实体名。
- `acl template` 和 `acl rule` 保持二级命令，不强行拆成根级命令。

## raw request

raw 是兜底层。

适用场景：

- API 尚未封装为实体命令。
- 调试请求。
- 临时验证后端行为。
- 新版本字段尚未进入 CLI。

不适用场景：

- 日常工作流。
- 已有实体命令覆盖的操作。
- 绕过 CLI 高风险校验。

命令形式：

```bash
safeline-3 raw request GET /api/v3/license
safeline-3 raw request POST /api/v3/protected-logger/DetectLogList --body-file body.json
```

## 短命令

短命令先不实现。

短命令的定位不是复杂工作流编排，而是高频实体操作的快捷入口。

短命令应该满足：

- 比实体命令更短。
- 默认参数明确。
- 低歧义。
- 不自动执行复杂多实体编排。
- 不绕过实体层校验。
- 不隐藏高风险行为。

短命令不应该承担：

- 自动创建 listener + site + policy 的完整工作流。
- 复杂业务判断。
- 替代实体命令。
- 作为业务规则的事实来源。

候选短命令示例，仅作为后续方向，不在当前阶段实现：

```bash
safeline-3 sites
safeline-3 node-groups
safeline-3 attacks
safeline-3 access-logs
safeline-3 bot-logs
safeline-3 time
safeline-3 license
```

这些命令如果未来实现，应只是实体命令的薄包装：

```text
sites       -> site list
node-groups -> node-group list
attacks     -> log attack list --start -24h
time        -> system time
license     -> system license
```

## AI 调用策略

更完整的 AI Agent 操作规则见 [`agent-skill.md`](agent-skill.md)。

AI Agent 的推荐调用顺序：

1. 优先使用实体命令。
2. 如果未来短命令覆盖了明确高频查询，可使用短命令。
3. 只有实体命令缺失或需要调试时才使用 `raw request`。

示例：

```text
查询节点组能力：
  使用 node-group capabilities

创建防护对象：
  使用 node-group get/capabilities 判断模式
  再使用 listener/site 实体命令

查询日志：
  使用 log attack/access/bot

调试未封装接口：
  使用 raw request
```

## 测试要求

命令分层测试应覆盖：

- 根命令不意外暴露未规划短命令。
- 实体命令存在并有 help。
- raw request 只在 raw 层暴露通用 HTTP 参数。
- 禁止参数不出现在实体命令中，例如 `--gm-cert-id`、`--protected-group`、`--tag`。
- 如果未来实现短命令，短命令必须与对应实体命令生成等价请求。
