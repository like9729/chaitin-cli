# SafeLine-3 实体模型

本文档定义 `safeline-3` CLI 当前阶段采用的实体边界和实体关系。目标是让 CLI 命令尽量对应 SafeLine-3 产品中的真实对象，同时让 AI Agent 能通过实体关系组合命令，而不是依赖一组固定工作流。

## 设计结论

- 当前阶段不新增短命令。
- 实体命令是主入口。
- `raw request` 是 API 逃生口。
- `log` 保持简单命名，不改成 `log-event`。
- `site` 保持现有命名，不强行改成 `protected-object`。在 SafeLine-3 语义中，`site` 表示防护对象。
- 实体关系可以通过 skill 描述给 AI，但关系本身不应只存在于 skill。长期应由 SafeLine-3 产品仓库导出的 manifest/schema 承担事实来源。

## 实体列表

### node-group

节点组是防护对象和 listener 的工作模式边界。

关系：

- `node-group` 包含一个或多个节点。
- `node-group` 决定可创建的 `site` 类型。
- `node-group` 决定可创建的 `listener` 类型。
- `site create` 和 `listener create` 必须先读取节点组模式并做兼容性校验。

关键字段：

- `id`
- `name`
- `mode`
- `nodes`
- `is_default`
- `standalone`

### site

`site` 是 SafeLine-3 防护对象的 CLI 名称。

保留 `site` 而不是改成 `protected-object` 的原因：

- 命令更短。
- 延续 SafeLine-2 CLI 的用户习惯。
- 对 AI 和人类都更容易输入。
- 内部 API path 仍可映射到 `/api/v3/protected-object/...`。

关系：

- `site` 属于一个 `node-group`。
- 代理类 `site` 引用已有 `listener`。
- 非代理类 `site` 使用远端 listener 描述。
- `site` 包含一个或多个 application。
- `site` 或 application 可引用 `policy-group`。
- TLS listener 场景下，`site` 可能需要引用普通证书。

当前支持类型：

- `reverse-proxy`
- `route-proxy`
- `transparent-proxy`
- `transparent`
- `mirror`
- `sdk`

首版不暴露：

- 国密证书参数。
- 防护组参数。
- 标签参数。

### listener

listener 是防护对象流量入口。

关系：

- `listener` 属于一个 `node-group`。
- 代理类 `site` 引用已有 `listener`。
- listener 类型受 `node-group.mode` 约束。

当前 CLI 已实现：

- `listener list`
- `listener create reverse-proxy`
- `listener update reverse-proxy`
- `listener create route-proxy`
- `listener update route-proxy`
- `listener create transparent-proxy`
- `listener update transparent-proxy`
- `listener delete`

三类 listener 都必须按节点组模式校验：`reverse-proxy` listener 只能用于 `ReverseProxy` 节点组，`route-proxy` listener 只能用于 `RouteProxy` 节点组，`transparent-proxy` listener 只能用于 `Transparent` 节点组。

### ip-group

IP 组是检测策略和访问控制中可复用的 IP/CIDR 集合。

关系：

- 可被策略、ACL 或其它检测配置引用。
- 当前 CLI 将其作为独立实体管理。

### policy-group

策略组是检测策略集合。

关系：

- 可被 `site` 或 application 引用。
- 内部包含模块级检测配置。

当前 CLI 只提供常见语义化操作和文件兜底，不试图完整展开所有复杂策略结构。

### policy-rule

策略规则是自定义检测规则。

关系：

- 可作为全局规则。
- 可绑定到 application。
- 可参与 ACL 相关配置。

### acl template

ACL 模板是访问控制规则的生成模板。

关系：

- 作用于一个或多个 `node-group`。
- 可引用 `policy-rule`。
- 可生成 `acl rule`。

当前命令保持为：

```bash
safeline-3 acl template ...
```

暂不强行拆成根级 `acl-template`，以保持命令简洁。

### acl rule

ACL 规则是 ACL 模板生成或维护的具体规则。

当前命令保持为：

```bash
safeline-3 acl rule ...
```

### log

日志是查询域，不改名。

关系：

- attack/access/bot 日志可关联 `site`、application、node-group、node。
- 日志查询需要时间范围；CLI 默认补最近 24 小时。

当前命令：

```bash
safeline-3 log attack list
safeline-3 log access list
safeline-3 log bot list
```

### system

系统域用于授权、机器码、时间等系统级查询和维护。

当前明确不提供：

- `system reboot`
- `system shutdown`

### network

网络域用于查询网口、链路、网络服务、VRRP。

当前明确不提供：

- `network soft-bypass`
- `network hard-bypass`

### monitor

监控域用于节点状态和历史状态查询。

`monitor node-groups` 只作为查询/兼容别名，主入口仍是 `node-group`。

### raw

raw 是 API 逃生口。

关系：

- 不承诺业务语义。
- 不作为正常工作流首选。
- 用于未封装接口、排障、临时验证。

## 节点组模式兼容矩阵

| node-group mode | 可创建 site 类型 | listener 形态 |
| --- | --- | --- |
| `reverse-proxy` | `reverse-proxy` | 已存在 reverse-proxy listener |
| `route-proxy` | `route-proxy` | 已存在 route-proxy listener |
| `transparent` | `transparent-proxy`, `transparent` | transparent-proxy 使用已有 listener；transparent 使用远端 listener |
| `mirror` | `transparent`, `mirror` | 远端 listener |
| `sdk` | `sdk` | 远端 listener；仅后端已存在 SDK/software 节点组时可用 |

CLI 必须执行该矩阵的硬校验，不能只依赖 skill 提醒。

## Skill 的职责

AI Agent 的当前操作 skill 见 [`agent-skill.md`](agent-skill.md)。

skill 应该描述：

- 实体之间的关系。
- AI 在执行任务前应该查询哪些实体。
- 创建不同类型 `site` 的推荐步骤。
- 失败后的排障路径。
- 什么时候使用实体命令，什么时候使用 raw。

skill 不应该成为以下内容的唯一来源：

- API method/path。
- required / condition-required。
- 枚举完整列表。
- 高风险操作名单。
- 节点组模式兼容矩阵。

这些内容长期应来自 SafeLine-3 产品仓库导出的 manifest/schema，并由 CLI 执行层校验。
