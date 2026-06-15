# SafeLine-3 CLI 设计文档

本文档定义 `chaitin-cli safeline-3` 的目标命令形态、参数格式、必填规则和 help 编写规范。目标是让人和 AI Agent 只通过 CLI help 就能准确完成 SafeLine-3 的常见运维任务，而不需要回头阅读前端代码或后端 API 结构。

关于该 CLI 与 SafeLine-3 产品自身 AI 化边界的长期定位，见 [`ai-cli-positioning.md`](ai-cli-positioning.md)。当前 CLI 应被视为过渡层和验证层，长期业务语义应沉淀回 SafeLine-3 产品自身。

实体关系和命令分层分别见 [`entity-model.md`](entity-model.md) 与 [`command-layers.md`](command-layers.md)。AI Agent 的具体调用策略见 [`agent-skill.md`](agent-skill.md)。当前阶段只实现实体命令和 `raw request`，短命令先不实现。

## 设计原则

- `safeline` 保持为 SafeLine-2 CLI，不复用或破坏现有行为。
- `safeline-3` 使用独立命令树和独立配置。
- 命令应尽量对应 SafeLine-3 产品实体或稳定查询域；`site` 保持为防护对象的 CLI 名称，`log` 保持为日志查询域名称。
- 日常命令必须使用业务参数，不暴露 `--query key=value` / `--request JSON` 作为主交互。
- `raw request` 只作为高级兜底入口，用于未封装接口、排障和临时验证。
- 写操作、删除操作和高风险系统操作必须要求 `--yes`；可用根级 `--dry-run` 预览请求。
- 复杂嵌套结构采用“常用场景显式参数 + 高级 JSON 文件”的模式。
- 实体关系可以通过 skill 描述给 AI，但影响安全、必填和兼容性的规则必须由 CLI 或产品 manifest 执行硬校验。
- 短命令属于后续阶段，只能作为高频实体命令的薄包装，当前不实现。
- 首版不支持国密证书参数。

## 配置

配置文件：

```yaml
safeline-3:
  url: https://safeline3.example.com
  api_token: YOUR_API_TOKEN
```

环境变量：

```bash
SAFELINE_3_URL=https://safeline3.example.com
SAFELINE_3_API_TOKEN=YOUR_API_TOKEN
```

全局参数：

| 参数 | 格式 | 默认值 | 说明 |
| --- | --- | --- | --- |
| `--url` | URL | 配置文件/环境变量 | SafeLine-3 地址 |
| `--api-token` | string | 配置文件/环境变量 | OpenAPI Token，发送为 `API-TOKEN` header |
| `-o, --output` | `table|json` | `table` | 输出格式 |
| `--insecure` | bool | `true` | 跳过 TLS 证书校验 |
| `-v, --verbose` | bool | `false` | 打印请求 URL、header、body |
| `--verbose-sensitive` | bool | `false` | verbose 时打印完整 token |
| 根级 `--dry-run` | bool | `false` | 打印请求但不发送 |

## 通用参数格式

| 类型 | CLI 格式 | 示例 | API 转换 |
| --- | --- | --- | --- |
| ID | 正整数 | `123` | JSON number |
| ID 列表 | 多个位置参数，或逗号分隔 | `delete 1 2 3` / `--ids 1,2,3` | `[]number` |
| 字符串列表 | 参数可重复，或逗号分隔 | `--domain a.com --domain b.com` / `--ip 1.1.1.1,2.2.2.2` | `[]string` |
| bool | flag 存在即 true，或显式 true/false | `--tls` / `--enabled=false` | JSON bool |
| 时间 | RFC3339、本地时间、Unix 秒/毫秒、相对时间 | `2026-06-13T10:00:00+08:00` / `2026-06-13 10:00:00` / `-24h` / `now` | Unix 毫秒 |
| 分页 | `--page` + `--page-size` | `--page 2 --page-size 50` | `offset=(page-1)*page_size`、`count=page_size` |
| 枚举 | 小写短值 | `--state detect` | CLI 映射为 SafeLine-3 API 枚举 |
| JSON 文件 | 文件路径或 `-` | `--condition-file rule.json` | 读取后作为 JSON object/array |

列表过滤的基础形式：

| 参数 | 语义 |
| --- | --- |
| `--name value` | 名称模糊匹配 |
| `--name-exact value` | 名称精确匹配 |
| `--id value` | ID 精确匹配 |
| `--enabled true|false` | 启用状态 |

高级过滤保留统一格式：

```bash
--filter 'FIELD:OP:VALUE'
```

`OP` 支持：`=`、`!=`、`contains`、`not-contains`、`in`、`not-in`、`>`、`<`、`>=`、`<=`。

## Help 编写规范

每一级命令都必须提供可直接指导 AI 调用的 help，不只依赖 Cobra 自动 flag 列表。

每个叶子命令的 `Long` 或 `Example` 至少包含：

- 命令用途。
- API 映射：method + path。
- 必填参数：名称、类型、格式。
- 条件必填参数：触发条件、类型、格式。
- 可选参数：默认值、枚举值。
- 参数格式说明：时间、列表、upstream、filter 等。
- 行为说明：是否写操作、是否需要 `--yes`、是否会先 get 后合并。
- 输出说明：table/json 的主要字段。
- 示例：最小可用示例和常见生产示例。

示例模板：

```text
Create a reverse-proxy protected object.

API:
  POST /api/v3/protected-object/reverse-proxy

Required:
  --name string
  --node-group ID
  --domain string      Repeatable.
  --listener ID        Repeatable.
  --upstream URL[,weight=N]  Repeatable.
  --yes

Conditionally Required:
  --cert-id ID
      Required when any selected listener has TLS enabled.

Optional:
  --app-name string       Default: default
  --url-path string       Default: /
  --url-path-op prefix|exact|regex  Default: prefix
  --state detect|dry-run|bypass|forbidden|not-apply|redirect|response|cache
  --policy-group ID
  --comment string

Formats:
  --upstream http://HOST:PORT[,weight=N]
  --upstream https://HOST:PORT[,weight=N]

Examples:
  chaitin-cli safeline-3 site create reverse-proxy --name app --node-group 1 --domain app.example.com --listener 10 --upstream http://10.0.0.1:8080 --yes
```

## 命令规划

### node-group

节点组命令。SafeLine-3 和 SafeLine-2 最大差异之一是：防护对象不是只由全局工作模式决定，而是由节点组工作模式决定。`site create`、`listener create` 必须先知道 `--node-group` 对应的 `mode`，再判断能创建哪类保护对象。

节点组模式和防护对象兼容关系来自后端 `ProtectType.CompatibleWithNodeGroupMode`：

| 节点组模式 | CLI mode | 可创建的防护对象类型 | 说明 |
| --- | --- | --- | --- |
| 反向代理 | `reverse-proxy` | `reverse-proxy` | 使用已有 reverse-proxy listener ID |
| 路由代理 | `route-proxy` | `route-proxy` | 使用已有 route-proxy listener ID |
| 透明模式 | `transparent` | `transparent-proxy`、`transparent` | `transparent-proxy` 使用已有 listener ID；`transparent` 使用远端 listener 描述 |
| 镜像模式 | `mirror` | `transparent`、`mirror` | 两类都使用远端 listener 描述 |
| SDK 模式 | `sdk` | `sdk` | 仅后端已存在 SDK/software 节点组时可用；不通过 `set-mode` 或 `node-group create` 创建 |

CLI 规则：

- `site create <type>` 收到 `--node-group` 后必须查询节点组模式并做兼容性校验，不兼容时直接本地报错。
- `listener create <type>` 也必须查询节点组模式；listener 类型必须和节点组模式兼容。
- `site capabilities --node-group ID` 应复用 `node-group capabilities ID` 的结果。
- 节点组工作模式切换会删除该节点组 listener，并重置/初始化节点网络配置；必须作为高风险操作处理。
- 默认节点组是产品内置边界，单机模式下不可通过 `node-group update/delete` 编辑或删除；切换工作模式使用 `node-group set-mode`。
- SDK 节点组只由 SDK/software 部署流程产生，CLI 不通过 `node-group create` 或 `node-group set-mode` 创建 SDK 模式。
- `monitor node-groups` 只保留为兼容/监控别名，主入口使用 `node-group`。

#### `node-group list`

API：`GET /api/v3/node_group/list`

参数：

| 参数 | 格式 | 必填 | 默认值 | 说明 |
| --- | --- | --- | --- | --- |
| `--mode` | `reverse-proxy|route-proxy|transparent|mirror|sdk` | 否 | 全部模式 | 按节点组模式过滤 |
| `--name` | string | 否 | - | 名称模糊匹配；若 API 不支持则 CLI 本地过滤 |
| `--id` | ID | 否 | - | ID 精确过滤；若 API 不支持则 CLI 本地过滤 |
| `--default` | bool | 否 | - | 过滤默认节点组 |
| `--standalone` | bool | 否 | - | 过滤单节点组 |

输出字段：

| 字段 | 说明 |
| --- | --- |
| `id` | 节点组 ID |
| `name` | 节点组名称 |
| `mode` | 节点组工作模式 |
| `supported_site_types` | 该节点组可创建的防护对象类型 |
| `standalone` | 是否单节点组 |
| `is_default` | 是否默认节点组 |
| `node_count` | 节点数量或状态摘要 |

示例：

```bash
chaitin-cli safeline-3 node-group list
chaitin-cli safeline-3 node-group list --mode transparent
```

#### `node-group get <id>`

API：`GET /api/v3/node_group/list` 后按 ID 过滤；必要时补充调用 `GET /api/v3/node_group/nodes` 和 `GET /api/v3/node_group/network/summary`。

参数：

| 参数 | 格式 | 必填 | 默认值 | 说明 |
| --- | --- | --- | --- | --- |
| `<id>` | ID | 是 | - | 节点组 ID |
| `--with-nodes` | bool | 否 | `false` | 同时输出节点列表 |
| `--with-network` | bool | 否 | `false` | 同时输出网口/IP/虚拟线摘要 |

示例：

```bash
chaitin-cli safeline-3 node-group get 1
chaitin-cli safeline-3 node-group get 1 --with-nodes --with-network
```

#### `node-group nodes <id>`

API：`GET /api/v3/node_group/nodes`

参数：

| 参数 | 格式 | 必填 | 默认值 | 说明 |
| --- | --- | --- | --- | --- |
| `<id>` | ID | 是 | - | 节点组 ID |
| `--page` | int | 否 | `1` | 页码 |
| `--page-size` | int | 否 | `20` | 每页数量 |

示例：

```bash
chaitin-cli safeline-3 node-group nodes 1
```

#### `node-group network <id>`

API：`GET /api/v3/node_group/network/summary`

用于创建 `transparent-proxy` listener、透明/镜像对象、虚拟线相关配置前查看节点组网络资源。

参数：

| 参数 | 格式 | 必填 | 默认值 | 说明 |
| --- | --- | --- | --- | --- |
| `<id>` | ID | 是 | - | 节点组 ID |

示例：

```bash
chaitin-cli safeline-3 node-group network 2
```

#### `node-group capabilities <id>`

本地能力计算命令。读取节点组 `mode` 后输出可创建的 protection object、listener 类型和 `site create` 必填参数。

参数：

| 参数 | 格式 | 必填 | 默认值 | 说明 |
| --- | --- | --- | --- | --- |
| `<id>` | ID | 是 | - | 节点组 ID |
| `--type` | `reverse-proxy|route-proxy|transparent-proxy|transparent|mirror|sdk` | 否 | 全部类型 | 只查看某类防护对象是否兼容 |

示例：

```bash
chaitin-cli safeline-3 node-group capabilities 1
chaitin-cli safeline-3 node-group capabilities 2 --type transparent-proxy
```

#### `node-group create`

API：`POST /api/v3/node_group`

参数：

| 参数 | 格式 | 必填 | 默认值 | 说明 |
| --- | --- | --- | --- | --- |
| `--name` | string | 是 | - | 节点组名称 |
| `--mode` | `reverse-proxy|route-proxy|transparent|mirror` | 是 | - | 节点组工作模式，映射 API `mode`；不支持创建 SDK 模式 |
| `--node` | node numeric ID，可重复/逗号分隔 | 否 | 空数组 | 初始加入节点，映射 `nodes` |
| `--yes` | bool | 是 | - | 确认创建 |
| `--check` | bool | 否 | `false` | 打印 payload，不写入 |
| `--explain` | bool | 否 | `false` | 解释 mode 与防护对象兼容关系 |

说明：API 有 `gm_supported` 字段，但首版不提供国密证书能力入口，CLI 创建时默认不暴露该参数。后端只允许硬件集群环境创建节点组；单机默认节点组不可手工创建、更新或删除。

示例：

```bash
chaitin-cli safeline-3 node-group create --name transparent-ng --mode transparent --node 3 --yes
```

#### `node-group update <id>`

API：`PUT /api/v3/node_group`

用于改名称、增加节点、移除节点。若节点组内已有节点，前端逻辑不允许通过普通编辑改工作模式；CLI 也不在 `update` 中提供 `--mode`，工作模式切换走 `node-group set-mode`。默认节点组不可更新。

参数：

| 参数 | 格式 | 必填 | 默认值 | 说明 |
| --- | --- | --- | --- | --- |
| `<id>` | ID | 是 | - | 节点组 ID |
| `--name` | string | 否 | 保留原值 | 新名称 |
| `--add-node` | node numeric ID，可重复/逗号分隔 | 否 | - | 增加节点，映射 `add_nodes` |
| `--remove-node` | node numeric ID，可重复/逗号分隔 | 否 | - | 移除节点，映射 `delete_nodes` |
| `--yes` | bool | 是 | - | 确认更新 |
| `--check` | bool | 否 | `false` | 打印 payload，不写入 |

示例：

```bash
chaitin-cli safeline-3 node-group update 2 --name transparent-prod --add-node 5 --yes
```

#### `node-group set-mode <id>`

API：`PUT /api/v3/workmode`

高风险命令。后端会删除该节点组 listener，重置节点网络配置，再初始化新工作模式。CLI 必须在 `--explain` 中展示影响范围，并在未传 `--yes` 时拒绝执行。

参数：

| 参数 | 格式 | 必填 | 默认值 | 说明 |
| --- | --- | --- | --- | --- |
| `<id>` | ID | 是 | - | 节点组 ID |
| `--mode` | `reverse-proxy|route-proxy|transparent|mirror` | 是 | - | 新工作模式；后端当前 workmode API 不接受 SDK |
| `--yes` | bool | 是 | - | 确认高风险变更 |
| `--check` | bool | 否 | `false` | 查询当前模式、listener/site 数量并打印变更计划 |
| `--explain` | bool | 否 | `false` | 展示将删除 listener、重置网络的说明 |

示例：

```bash
chaitin-cli safeline-3 node-group set-mode 2 --mode transparent --check
chaitin-cli safeline-3 node-group set-mode 2 --mode transparent --yes
```

#### `node-group delete <id...>`

API：`DELETE /api/v3/node_group`

默认节点组不可删除。后端只允许硬件集群节点组删除。

参数：

| 参数 | 格式 | 必填 | 默认值 | 说明 |
| --- | --- | --- | --- | --- |
| `<id...>` | ID 列表 | 是 | - | 一个或多个节点组 ID |
| `--yes` | bool | 是 | - | 确认删除 |
| `--check` | bool | 否 | `false` | 检查节点组内是否仍有节点、防护对象、listener |

示例：

```bash
chaitin-cli safeline-3 node-group delete 3 --check
chaitin-cli safeline-3 node-group delete 3 --yes
```

### site

保护对象命令。设计上参考 SafeLine-2 的 `site create`：先暴露能力，再按当前目标支持的对象类型提供语义化参数，最后保留文件入口作为复杂结构兜底。不能只把 `site` 等价为反向代理。

SafeLine-3 的保护对象分为六类，并且必须和节点组模式兼容：

| CLI type | API path segment | 兼容节点组模式 | 类型 | 创建策略 | listener 形态 |
| --- | --- | --- | --- | --- | --- |
| `reverse-proxy` | `reverse-proxy` | `reverse-proxy` | 代理类 | `semantic` | 已存在 listener ID |
| `route-proxy` | `route-proxy` | `route-proxy` | 代理类 | `semantic_limited` | 已存在 listener ID |
| `transparent-proxy` | `transparent-proxy` | `transparent` | 代理类 | `semantic_limited` | 已存在 listener ID |
| `transparent` | `transparent` | `transparent`、`mirror` | 非代理类 | `semantic_limited` | 远端 listener 描述 |
| `mirror` | `mirror` | `mirror` | 非代理类 | `semantic_limited` | 远端 listener 描述 |
| `sdk` | `sdk` | `sdk` | 非代理类 | `semantic_limited` | 远端 listener 描述；仅后端已存在 SDK/software 节点组时可用 |

策略定义：

| 策略 | 含义 |
| --- | --- |
| `semantic` | CLI 用业务参数生成完整 payload，覆盖主流程。 |
| `semantic_limited` | CLI 覆盖对象、域名、listener、基础 application、防护状态等稳定字段；复杂检测、会话、响应文件、特殊应用配置用文件参数补充。 |
| `payload_file` | 用户提供完整 JSON body。用于新版本字段、排障和 CLI 尚未封装的复杂结构。 |

所有写操作统一支持：

| 参数 | 格式 | 必填 | 默认值 | 说明 |
| --- | --- | --- | --- | --- |
| `--check` | bool | 否 | `false` | 本地校验并打印将发送的 payload，不写入。 |
| `--explain` | bool | 否 | `false` | 打印字段来源、默认值、API 映射和风险提示，不写入。 |
| `--yes` | bool | 是 | `false` | 确认写入。`--check`、`--explain` 或根级 `--dry-run` 时不要求实际写入。 |
| `--payload-file` | JSON 文件路径或 `-` | 否 | - | 完整 API body；和语义化字段互斥，除 `--yes/--check/--explain` 外不再合并其它字段。 |

证书参数只支持普通证书 `--cert-id`，映射 `ssl_certificate`。国密证书不在首版范围内。

`site create <type>` 和 `site update <type>` 在发送 API 前必须读取目标 `--node-group` 的模式，并用上表做本地校验。不兼容时直接返回类似 `node group 2 is mode Transparent, cannot create reverse-proxy; supported types: transparent-proxy, transparent` 的错误。

#### `site capabilities`

输出当前 SafeLine-3 目标支持的保护对象创建能力。该命令必须让 AI 可以先判断“应该调用哪种 create 命令、哪些参数必填、哪些字段需要文件兜底”。

参数：

| 参数 | 格式 | 必填 | 默认值 | 说明 |
| --- | --- | --- | --- | --- |
| `--type` | `reverse-proxy|route-proxy|transparent-proxy|transparent|mirror|sdk` | 否 | 全部类型 | 只查看某类保护对象能力 |
| `--node-group` | ID | 否 | - | 结合节点组模式判断类型兼容性 |
| `--output` | `table|json` | 否 | 根级默认值 | `json` 输出包含 required fields、semantic flags 和 notes |

输出字段：

| 字段 | 说明 |
| --- | --- |
| `type` | 保护对象类型 |
| `supported` | 当前目标/节点组是否支持 |
| `create_strategy` | `semantic`、`semantic_limited` 或 `payload_file` |
| `required_flags` | CLI 必填参数 |
| `condition_required_flags` | 条件必填参数，例如 TLS listener 需要 `--cert-id` |
| `file_flags` | 可用于补充复杂结构的文件参数 |
| `notes` | 模式限制和实现注意事项 |

示例：

```bash
chaitin-cli safeline-3 site capabilities
chaitin-cli safeline-3 site capabilities --node-group 1 --output json
chaitin-cli safeline-3 site capabilities --type transparent
```

#### `site list`

API：按类型查询保护对象列表。未指定 `--type` 时按当前目标支持的类型汇总查询，不默认当作 `reverse-proxy`。

参数：

| 参数 | 格式 | 必填 | 默认值 | 说明 |
| --- | --- | --- | --- | --- |
| `--type` | `reverse-proxy|route-proxy|transparent-proxy|transparent|mirror|sdk` | 否 | 全部支持类型 | 保护对象类型 |
| `--name` | string | 否 | - | 名称模糊匹配 |
| `--domain` | string | 否 | - | 域名过滤 |
| `--node-group` | ID | 否 | - | 节点组 |
| `--enabled` | bool | 否 | - | 启用状态 |
| `--page` | int | 否 | `1` | 页码 |
| `--page-size` | int | 否 | `20` | 每页数量 |

示例：

```bash
chaitin-cli safeline-3 site list
chaitin-cli safeline-3 site list --type reverse-proxy --name app --page 1 --page-size 50
chaitin-cli safeline-3 site list --type transparent --node-group 2
```

#### `site get <id>`

参数：

| 参数 | 格式 | 必填 | 默认值 | 说明 |
| --- | --- | --- | --- | --- |
| `<id>` | ID | 是 | - | 保护对象 ID |
| `--type` | 同 `site list --type` | 否 | 自动探测 | 不传时按类型逐个探测 |

示例：

```bash
chaitin-cli safeline-3 site get 12
chaitin-cli safeline-3 site get 12 --type reverse-proxy
```

#### 通用 create 参数

下面参数适用于所有 `site create <type>`。各类型的额外必填参数见后续小节。

真实 3.0 对象必填：`node_group`、`name`、`domain_names`、至少一个 `applications`。应用内至少需要 `url_paths`，`protected_state` 必须合法。首版 `site create` 不暴露防护组和标签绑定参数。

参数：

| 参数 | 格式 | 必填 | 默认值 | 说明 |
| --- | --- | --- | --- | --- |
| `--name` | string | 是 | - | 保护对象名称 |
| `--node-group` | ID | 是 | - | 节点组 ID |
| `--domain` | string，可重复/逗号分隔 | 是 | - | 转成 `domain_names` |
| `--enabled` | bool | 否 | `false` | 创建后启用对象，映射 `is_enabled` |
| `--comment` | string | 否 | `""` | 备注 |
| `--app-name` | string | 否 | `default` | 默认 application 名称 |
| `--url-path` | string，可重复/逗号分隔 | 否 | `/` | 应用 URL 路径，转成 `applications[].url_paths` |
| `--url-path-op` | `prefix|exact|regex` | 否 | `prefix` | URL path 匹配方式 |
| `--state` | `detect|dry-run|bypass|forbidden|not-apply|redirect|response|cache` | 否 | `detect` | 应用防护状态，映射 `protected_state` |
| `--policy-group` | ID | 否 | - | 检测策略组 ID |
| `--application-file` | JSON 文件路径或 `-` | 否 | - | 覆盖/补充 `applications`；接受单个 object 或 array |
| `--detector-config-file` | JSON 文件路径或 `-` | 否 | - | 应用检测配置，映射 `detector_config` |
| `--session-file` | JSON 文件路径或 `-` | 否 | - | 会话配置，映射 `session_method` |
| `--access-log-file` | JSON 文件路径或 `-` | 否 | - | 访问日志配置，映射 `access_log` |
| `--check` | bool | 否 | `false` | 预检，不写入 |
| `--explain` | bool | 否 | `false` | 解释 payload，不写入 |
| `--yes` | bool | 是 | - | 确认写操作 |

#### 代理类 create 参数

适用于 `reverse-proxy`、`route-proxy`、`transparent-proxy`。API 参数嵌入 `common.ProxyObjectParams`，listener 使用已有 listener ID。

| 参数 | 格式 | 必填 | 默认值 | 说明 |
| --- | --- | --- | --- | --- |
| `--listener` | ID，可重复/逗号分隔 | 是 | - | 已存在 listener ID，转成 `listeners` |
| `--cert-id` | ID | 条件必填 | - | TLS listener 时必填；映射 `ssl_certificate` |
| `--proxy-detection-config-file` | JSON 文件路径或 `-` | 否 | - | 映射 `proxy_detection_config` |
| `--custom-nginx-config-file` | JSON 文件路径或 `-` | 否 | - | 映射 `custom_nginx_config` |

#### 非代理类 create 参数

适用于 `transparent`、`mirror`、`sdk`。API 的 `listeners` 是远端 listener 描述，不是已有 listener ID，因此不能使用 `--listener`。

| 参数 | 格式 | 必填 | 默认值 | 说明 |
| --- | --- | --- | --- | --- |
| `--remote-listener` | `IP:PORT` 或 `IP/CIDR:PORT`，可重复/逗号分隔 | 是 | - | 转成 `listeners` 里的远端 listener |
| `--remote-listener-file` | JSON 文件路径或 `-` | 否 | - | 复杂远端 listener 列表；和 `--remote-listener` 合并去重 |

远端 listener 格式：

```text
--remote-listener 10.0.0.10:80
--remote-listener 10.0.1.0/24:443
```

#### `site create reverse-proxy`

API：`POST /api/v3/protected-object/reverse-proxy`

创建策略：`semantic`。除通用对象/application 参数外，额外支持反代 action 的语义化参数。

真实 3.0 必填：通用对象必填字段、代理类 `--listener`、至少一个 application；当 `--backend-type proxy` 时至少一个 `--upstream`；当 `--backend-type redirect` 时必须 `--redirect-url`。如果关联 TLS listener，必须提供普通证书 `--cert-id`。

参数：

| 参数 | 格式 | 必填 | 默认值 | 说明 |
| --- | --- | --- | --- | --- |
| `--backend-type` | `proxy|redirect` | 否 | `proxy` | 反代 action 类型；响应文件首版用 `--application-file` |
| `--upstream` | `http://HOST:PORT[,weight=N]` 或 `https://HOST:PORT[,weight=N]`，可重复 | 条件必填 | - | `--backend-type proxy` 时必填 |
| `--load-balance` | `round-robin|ip-hash|least-conn` | 否 | `round-robin` | 负载均衡策略，映射 backend config |
| `--backend-http2` | bool | 否 | `false` | 后端使用 HTTP/2 |
| `--backend-ntlm` | bool | 否 | `false` | 后端 NTLM |
| `--redirect-url` | URL | 条件必填 | - | `--backend-type redirect` 时必填 |
| `--redirect-code` | `301|302|307|308` | 否 | `302` | 跳转状态码 |

`--upstream` 解析：

```text
http://10.0.0.1:8080
https://backend.example.com:8443,weight=5
```

转换为：

```json
{
  "protocol": "http",
  "servers": [
    {"host": "10.0.0.1", "port": 8080, "weight": 1}
  ]
}
```

文件入口：

| 参数 | 格式 | 说明 |
| --- | --- | --- |
| `--application-file` | JSON object/array | 用于响应文件、动态解析、健康检查、keepalive、proxy bind 等复杂 application 字段 |
| `--payload-file` | JSON object | 完整 `POST /api/v3/protected-object/reverse-proxy` body |

示例：

```bash
chaitin-cli safeline-3 site create reverse-proxy \
  --name app \
  --node-group 1 \
  --domain app.example.com \
  --listener 10 \
  --upstream http://10.0.0.1:8080 \
  --yes

chaitin-cli --dry-run safeline-3 site create reverse-proxy \
  --name app \
  --node-group 1 \
  --domain app.example.com \
  --listener 10 \
  --cert-id 5 \
  --upstream https://backend.example.com:8443,weight=5 \
  --policy-group 2 \
  --state detect \
  --yes

chaitin-cli safeline-3 site create reverse-proxy \
  --name redirect-app \
  --node-group 1 \
  --domain old.example.com \
  --listener 10 \
  --backend-type redirect \
  --redirect-url https://new.example.com \
  --redirect-code 301 \
  --yes
```

#### `site create route-proxy`

API：`POST /api/v3/protected-object/route-proxy`

创建策略：`semantic_limited`。支持通用对象参数、代理类 listener 参数和基础 application 参数；路由代理特有的复杂 application 配置通过 `--application-file` 补充。

必填参数：

| 参数 | 格式 | 说明 |
| --- | --- | --- |
| `--name` | string | 保护对象名称 |
| `--node-group` | ID | 节点组 |
| `--domain` | string，可重复/逗号分隔 | 域名 |
| `--listener` | ID，可重复/逗号分隔 | route-proxy listener ID |
| `--yes` | bool | 确认写入 |

示例：

```bash
chaitin-cli safeline-3 site create route-proxy \
  --name route-app \
  --node-group 1 \
  --domain app.example.com \
  --listener 20 \
  --url-path /api \
  --state detect \
  --policy-group 2 \
  --yes

chaitin-cli safeline-3 site create route-proxy \
  --payload-file route-proxy-object.json \
  --check
```

#### `site create transparent-proxy`

API：`POST /api/v3/protected-object/transparent-proxy`

创建策略：`semantic_limited`。支持通用对象参数、代理类 listener 参数和基础 application 参数；透明代理特有字段通过 `--application-file` 或 `--payload-file` 补充。

必填参数：

| 参数 | 格式 | 说明 |
| --- | --- | --- |
| `--name` | string | 保护对象名称 |
| `--node-group` | ID | 节点组 |
| `--domain` | string，可重复/逗号分隔 | 域名 |
| `--listener` | ID，可重复/逗号分隔 | transparent-proxy listener ID |
| `--yes` | bool | 确认写入 |

示例：

```bash
chaitin-cli safeline-3 site create transparent-proxy \
  --name tp-app \
  --node-group 2 \
  --domain tp.example.com \
  --listener 31 \
  --url-path / \
  --state detect \
  --yes
```

#### `site create transparent`

API：`POST /api/v3/protected-object/transparent`

创建策略：`semantic_limited`。该类型不是代理类，不使用已有 listener ID；必须提供远端 listener 描述。

必填参数：

| 参数 | 格式 | 说明 |
| --- | --- | --- |
| `--name` | string | 保护对象名称 |
| `--node-group` | ID | 节点组 |
| `--domain` | string，可重复/逗号分隔 | 域名 |
| `--remote-listener` | `IP:PORT` 或 `IP/CIDR:PORT` | 远端 listener |
| `--yes` | bool | 确认写入 |

示例：

```bash
chaitin-cli safeline-3 site create transparent \
  --name transparent-app \
  --node-group 3 \
  --domain app.internal \
  --remote-listener 10.0.0.10:80 \
  --url-path / \
  --state detect \
  --yes
```

#### `site create mirror`

API：`POST /api/v3/protected-object/mirror`

创建策略：`semantic_limited`。该类型不是代理类，不使用已有 listener ID；必须提供远端 listener 描述。

必填参数：

| 参数 | 格式 | 说明 |
| --- | --- | --- |
| `--name` | string | 保护对象名称 |
| `--node-group` | ID | 节点组 |
| `--domain` | string，可重复/逗号分隔 | 域名 |
| `--remote-listener` | `IP:PORT` 或 `IP/CIDR:PORT` | 远端 listener |
| `--yes` | bool | 确认写入 |

示例：

```bash
chaitin-cli safeline-3 site create mirror \
  --name mirror-app \
  --node-group 4 \
  --domain mirror.example.com \
  --remote-listener 10.0.0.20:443 \
  --url-path / \
  --state detect \
  --yes
```

#### `site create sdk`

API：`POST /api/v3/protected-object/sdk`

创建策略：`semantic_limited`。SDK 类型沿用基础对象、远端 listener 和基础 application 参数；SDK 应用特有结构通过 `--application-file` 或 `--payload-file` 补充。

前置条件：目标 `--node-group` 必须是后端已经返回的 SDK/software 节点组。CLI 不通过 `node-group create --mode sdk` 或 `node-group set-mode --mode sdk` 准备 SDK 模式；普通硬件/单机节点组不能创建 SDK site。

必填参数：

| 参数 | 格式 | 说明 |
| --- | --- | --- |
| `--name` | string | 保护对象名称 |
| `--node-group` | ID | 节点组 |
| `--domain` | string，可重复/逗号分隔 | 域名 |
| `--remote-listener` | `IP:PORT` 或 `IP/CIDR:PORT` | 远端 listener |
| `--yes` | bool | 确认写入 |

示例：

```bash
chaitin-cli safeline-3 site create sdk \
  --name sdk-app \
  --node-group 5 \
  --domain sdk.example.com \
  --remote-listener 10.0.0.30:8080 \
  --url-path / \
  --state detect \
  --yes
```

#### `site update <type> <id>`

API：`PUT /api/v3/protected-object/<type>`

行为：先 get 当前对象，按类型合并用户传入字段，再 PUT 完整 payload。避免用户重复传完整 3.0 对象结构。`node_group` 在后端不可变，CLI 更新时默认保留原值，不提供跨节点组迁移语义。

参数：同对应 `site create <type>`，但 `<id>` 必填，`--yes` 必填，其它字段不传则保留原值。`--payload-file` 表示完整更新 body，仍会校验 `<id>` 和类型一致。

示例：

```bash
chaitin-cli safeline-3 site update reverse-proxy 12 --domain app.example.com --domain api.example.com --yes
chaitin-cli safeline-3 site update reverse-proxy 12 --upstream http://10.0.0.2:8080 --yes
chaitin-cli safeline-3 site update transparent 30 --remote-listener 10.0.0.11:80 --yes
chaitin-cli safeline-3 site update sdk 44 --application-file sdk-app.json --check
```

#### `site delete <id...>`

参数：

| 参数 | 格式 | 必填 | 默认值 | 说明 |
| --- | --- | --- | --- | --- |
| `<id...>` | ID 列表 | 是 | - | 一个或多个保护对象 ID |
| `--type` | 同上 | 否 | 自动探测 | 保护对象类型 |
| `--check` | bool | 否 | `false` | 打印将删除的对象，不写入 |
| `--yes` | bool | 是 | - | 确认删除 |

示例：

```bash
chaitin-cli safeline-3 site delete 12 --yes
chaitin-cli safeline-3 site delete 12 13 --type reverse-proxy --yes
```

#### `site enable <id...>` / `site disable <id...>`

行为：切换保护对象启用状态。参数同 `site delete`，但写入前必须 get 对象并按类型更新完整 payload，避免只 patch 局部字段导致后端丢失嵌套配置。

### listener

#### `listener list`

参数：

| 参数 | 格式 | 必填 | 默认值 | 说明 |
| --- | --- | --- | --- | --- |
| `--node-group` | ID | 是 | - | 节点组 |
| `--ip` | IP/CIDR | 否 | - | 监听地址 |
| `--port` | 1-65535 | 否 | - | 监听端口 |
| `--tls` | bool | 否 | - | TLS 状态 |
| `--page` | int | 否 | `1` | 页码 |
| `--page-size` | int | 否 | `20` | 每页数量 |

示例：

```bash
chaitin-cli safeline-3 listener list --node-group 1
```

#### `listener create reverse-proxy`

API：`POST /api/v3/protected-object/reverse-proxy/listener`

真实 3.0 校验：`node_group` 必须存在且模式兼容；`ip` 必须是合法 IP/CIDR；`port` 范围 1-65535；未启用 TLS 时不能传 TLS protocol/cipher/http2。

参数：

| 参数 | 格式 | 必填 | 默认值 | 说明 |
| --- | --- | --- | --- | --- |
| `--name` | string | 是 | - | listener 名称 |
| `--node-group` | ID | 是 | - | 节点组 |
| `--ip` | IP/CIDR | 是 | - | 监听地址 |
| `--port` | 1-65535 | 是 | - | 监听端口 |
| `--yes` | bool | 是 | - | 确认写操作 |
| `--tls` | bool | 否 | `false` | 启用 TLS |
| `--tls-protocol` | `SSLv3|TLSv1|TLSv1.1|TLSv1.2|TLSv1.3`，可重复 | 否 | - | 仅 `--tls` 时有效 |
| `--tls-ciphers` | string | 否 | - | OpenSSL cipher string，仅 `--tls` 时有效 |
| `--http2` | bool | 否 | `false` | 仅 `--tls` 时有效 |
| `--protected-object` | ID，可重复/逗号分隔 | 否 | - | 绑定保护对象 |

示例：

```bash
chaitin-cli safeline-3 listener create reverse-proxy \
  --name https-443 \
  --node-group 1 \
  --ip 0.0.0.0 \
  --port 443 \
  --tls \
  --tls-protocol TLSv1.2 \
  --tls-protocol TLSv1.3 \
  --http2 \
  --yes
```

#### `listener update reverse-proxy <id>`

行为：发送完整 reverse-proxy listener payload。`<id>` 和 `--yes` 必填；需要保留旧字段时先 `listener list` 或使用 `--payload-file` 构造完整请求。

#### `listener create route-proxy`

API：`POST /api/v3/protected-object/route-proxy/listener`

真实 3.0 校验：节点组必须是 `route-proxy`；`--inbound-ip` 必填；`ip` 可为 IP/CIDR；未启用 TLS 时不能传 TLS protocol/cipher/http2。

参数：

| 参数 | 格式 | 必填 | 默认值 | 说明 |
| --- | --- | --- | --- | --- |
| `--name` | string | 是 | - | listener 名称 |
| `--node-group` | ID | 是 | - | 节点组 |
| `--ip` | IP/CIDR | 是 | - | 监听地址 |
| `--port` | 1-65535 | 是 | - | 监听端口 |
| `--inbound-ip` | IP，可重复/逗号分隔 | 是 | - | 入站 IP，映射 `inbound_ips` |
| `--yes` | bool | 是 | - | 确认写操作 |
| `--tls` | bool | 否 | `false` | 启用 TLS |
| `--tls-protocol` | TLS 协议，可重复 | 否 | - | 仅 `--tls` 时有效 |
| `--tls-ciphers` | string | 否 | - | 仅 `--tls` 时有效 |
| `--http2` | bool | 否 | `false` | 仅 `--tls` 时有效 |
| `--ntlm` | bool | 否 | `false` | 启用 NTLM |
| `--transparent-server` | bool | 否 | `false` | 启用透明 server |
| `--protected-object` | ID，可重复/逗号分隔 | 否 | - | 绑定 route-proxy 防护对象 |
| `--payload-file` | JSON file 或 `-` | 否 | - | 完整 API payload |

示例：

```bash
chaitin-cli safeline-3 listener create route-proxy \
  --name route-http \
  --node-group 2 \
  --ip 10.0.0.0/24 \
  --port 80 \
  --inbound-ip 10.0.0.10 \
  --yes
```

#### `listener update route-proxy <id>`

行为：发送完整 route-proxy listener payload。`<id>` 和 `--yes` 必填。

#### `listener create transparent-proxy`

API：`POST /api/v3/protected-object/transparent-proxy/listener`

真实 3.0 校验：节点组必须是 `transparent`；`--virtual-wire-pair` 必填；未启用 TLS 时不能传 TLS protocol/cipher/http2。

参数：

| 参数 | 格式 | 必填 | 默认值 | 说明 |
| --- | --- | --- | --- | --- |
| `--name` | string | 是 | - | listener 名称 |
| `--node-group` | ID | 是 | - | 节点组 |
| `--ip` | IP/CIDR | 是 | - | 监听地址 |
| `--port` | 1-65535 | 是 | - | 监听端口 |
| `--virtual-wire-pair` | string | 是 | - | 虚拟线名称 |
| `--yes` | bool | 是 | - | 确认写操作 |
| `--tls` | bool | 否 | `false` | 启用 TLS |
| `--tls-protocol` | TLS 协议，可重复 | 否 | - | 仅 `--tls` 时有效 |
| `--tls-ciphers` | string | 否 | - | 仅 `--tls` 时有效 |
| `--http2` | bool | 否 | `false` | 仅 `--tls` 时有效 |
| `--ntlm` | bool | 否 | `false` | 启用 NTLM |
| `--transparent-port` | bool | 否 | `false` | 启用透明端口 |
| `--protected-object` | ID，可重复/逗号分隔 | 否 | - | 绑定 transparent-proxy 防护对象 |
| `--payload-file` | JSON file 或 `-` | 否 | - | 完整 API payload |

示例：

```bash
chaitin-cli safeline-3 listener create transparent-proxy \
  --name tp-http \
  --node-group 3 \
  --ip 10.0.0.10 \
  --port 80 \
  --virtual-wire-pair wire0 \
  --yes
```

#### `listener update transparent-proxy <id>`

行为：发送完整 transparent-proxy listener payload。`<id>` 和 `--yes` 必填。

#### `listener delete <id...>`

参数：`<id...>` 必填，`--yes` 必填。

### ip-group

API：`/api/v3/detect/ip_group`

#### `ip-group list`

参数：

| 参数 | 格式 | 必填 | 默认值 | 说明 |
| --- | --- | --- | --- | --- |
| `--name` | string | 否 | - | 名称模糊匹配 |
| `--name-exact` | string | 否 | - | 名称精确匹配 |
| `--cidr` | IP/CIDR | 否 | - | CIDR 归属过滤 |
| `--page` | int | 否 | `1` | 页码 |
| `--page-size` | int | 否 | `20` | 每页数量 |

#### `ip-group get <id>`

`<id>` 必填。

#### `ip-group create`

真实 3.0 create payload：`name`、`comment`、`original`。

参数：

| 参数 | 格式 | 必填 | 默认值 | 说明 |
| --- | --- | --- | --- | --- |
| `--name` | string | 是 | - | IP 组名称 |
| `--ip` | IP/CIDR，可重复/逗号分隔 | 是 | - | 转成 `original` |
| `--comment` | string | 否 | `""` | 备注 |

示例：

```bash
chaitin-cli safeline-3 ip-group create --name office --ip 192.168.1.0/24 --ip 10.0.0.1 --comment "office network"
```

#### `ip-group update <id>`

行为：先 get 当前 IP 组，合并 `--name`、`--ip`、`--comment` 后 PUT 完整 payload。`<id>` 必填。

#### `ip-group add-ip <id>` / `ip-group remove-ip <id>`

参数：

| 参数 | 格式 | 必填 | 说明 |
| --- | --- | --- | --- |
| `<id>` | ID | 是 | IP 组 ID |
| `--ip` | IP/CIDR，可重复/逗号分隔 | 是 | 增加或删除的 IP/CIDR |

#### `ip-group delete <id...>`

`<id...>` 必填，`--yes` 必填。

#### `ip-group delete --all`

`--all` 和 `--yes` 必填。

### policy-group

API：`/api/v3/detect/PolicyGroup`

#### `policy-group list`

真实 3.0 `list` 请求硬必填 `offset` 和 `count`，CLI 通过 `--page` / `--page-size` 自动转换。

参数：

| 参数 | 格式 | 必填 | 默认值 | 说明 |
| --- | --- | --- | --- | --- |
| `--name` | string | 否 | - | 名称过滤 |
| `--id` | ID | 否 | - | ID 过滤 |
| `--page` | int | 否 | `1` | 转 offset |
| `--page-size` | int | 否 | `20` | 转 count |

#### `policy-group all`

无参数，调用 `/api/v3/detect/PolicyGroup/all`。

#### `policy-group get <id>`

`<id>` 必填，调用 `/api/v3/detect/PolicyGroup/detail?id=<id>`。

#### `policy-group create`

真实 3.0 必填：`name`、`template_id`。

| 参数 | 格式 | 必填 | 默认值 | 说明 |
| --- | --- | --- | --- | --- |
| `--name` | string | 是 | - | 策略组名称 |
| `--template-id` | ID | 是 | - | 模板策略组 ID |
| `--comment` | string | 否 | `""` | 备注 |

#### `policy-group rename <id>`

行为：先 get 策略组详情，再修改名称/备注并 PUT 完整 payload。

| 参数 | 格式 | 必填 | 说明 |
| --- | --- | --- | --- |
| `<id>` | ID | 是 | 策略组 ID |
| `--name` | string | 是 | 新名称 |
| `--comment` | string | 否 | 新备注 |

#### `policy-group module set <id>`

行为：先 get 详情，修改 `modules_detection_config` 指定模块状态，再 PUT 完整 payload。

| 参数 | 格式 | 必填 | 说明 |
| --- | --- | --- | --- |
| `<id>` | ID | 是 | 策略组 ID |
| `--module` | string，可重复/逗号分隔 | 是 | 如 `m_sqli,m_xss` |
| `--state` | `enabled|disabled` | 是 | 模块状态 |

#### `policy-group delete <id...>`

`<id...>` 必填，`--yes` 必填。

### policy-rule

API：`/api/v3/detect/PolicyRule`

策略规则的真实 3.0 结构复杂。CLI 提供常用简单模式和高级文件模式。

#### `policy-rule list`

参数：

| 参数 | 格式 | 必填 | 默认值 | 说明 |
| --- | --- | --- | --- | --- |
| `--global` | bool | 否 | `true` | 查询全局规则 |
| `--app-id` | ID | 条件必填 | - | `--global=false` 时使用 |
| `--name` | string | 否 | - | 规则名过滤 |
| `--enabled` | bool | 否 | - | 启用状态 |
| `--action` | `deny|allow|dry-run` | 否 | - | 动作过滤 |
| `--risk-level` | `none|low|medium|high|0|1|2|3` | 否 | - | 风险等级 |
| `--page` | int | 否 | `1` | 页码 |
| `--page-size` | int | 否 | `20` | 每页数量 |

#### `policy-rule get <id>`

`<id>` 必填。

#### `policy-rule create simple`

真实 3.0 至少要求规则 `name` 非空，且 condition 能通过后端校验。

参数：

| 参数 | 格式 | 必填 | 默认值 | 说明 |
| --- | --- | --- | --- | --- |
| `--name` | string | 是 | - | 规则名 |
| `--target` | string | 是 | - | 匹配字段，如 `url_path`、`host`、`src_ip` |
| `--op` | `equals|contains|regex|prefix|suffix|in` | 是 | 匹配操作 |
| `--value` | string，可重复/逗号分隔 | 是 | 匹配值 |
| `--action` | `deny|allow|dry-run` | 是 | 规则动作 |
| `--risk-level` | `none|low|medium|high|0|1|2|3` | 否 | `none` | 风险等级 |
| `--bind` | `global|app` | 否 | `global` | 绑定范围 |
| `--app-id` | ID | 条件必填 | - | `--bind app` 时必填 |
| `--enabled` | bool | 否 | `true` | 是否启用 |
| `--log-option` | `persistence|none` | 否 | `persistence` | 日志策略 |
| `--yes` | bool | 是 | - | 确认写操作 |

示例：

```bash
chaitin-cli safeline-3 policy-rule create simple \
  --name block-admin \
  --target url_path \
  --op contains \
  --value /admin \
  --action deny \
  --risk-level high \
  --bind global \
  --yes
```

#### `policy-rule create`

高级文件模式。

| 参数 | 格式 | 必填 | 说明 |
| --- | --- | --- | --- |
| `--name` | string | 是 | 规则名 |
| `--condition-file` | JSON 文件或 `-` | 是 | 3.0 condition JSON |
| `--action` | `deny|allow|dry-run` | 是 | 规则动作 |
| `--binding-file` | JSON 文件或 `-` | 否 | binding JSON |
| `--schedule-type` | string | 否 | 调度类型 |
| `--schedule-file` | JSON 文件或 `-` | 否 | schedule_config JSON |
| `--yes` | bool | 是 | 确认写操作 |

#### `policy-rule update <id>`

行为：先 get 规则，合并传入字段后 PUT。支持 simple 参数和高级文件参数。

#### `policy-rule enable <id...>` / `policy-rule disable <id...>`

`<id...>` 必填，`--yes` 必填。

#### `policy-rule delete <id...>`

`<id...>` 必填，`--yes` 必填。

#### `policy-rule move <id>`

真实 3.0 `priority` 要求 `id` 非 0、`position` 大于 0。

| 参数 | 格式 | 必填 | 说明 |
| --- | --- | --- | --- |
| `<id>` | ID | 是 | 规则 ID |
| `--position` | int，>=1 | 是 | 目标位置 |
| `--global` | bool | 否 | 全局规则 |
| `--app-id` | ID | 条件必填 | 非 global 时必填 |
| `--yes` | bool | 是 | 确认写操作 |

#### `policy-rule bind <id...>` / `policy-rule unbind <id...>`

| 参数 | 格式 | 必填 | 说明 |
| --- | --- | --- | --- |
| `<id...>` | ID 列表 | 是 | 规则 ID |
| `--app-id` | ID | 是 | 应用 ID |
| `--yes` | bool | 是 | 确认写操作 |

### acl

API：`/api/v3/acl`

#### `acl template list`

真实 3.0 硬必填 `offset` 和 `count`，CLI 通过分页参数转换。

参数：

| 参数 | 格式 | 必填 | 默认值 | 说明 |
| --- | --- | --- | --- | --- |
| `--name` | string | 否 | - | 名称过滤 |
| `--target-type` | `cidr|session|fingerprint` | 否 | - | 目标类型，CLI 会映射为真实 API 的 `CIDR|Session|Fingerprint` |
| `--mode` | `forbidden|dryrun` | 否 | - | 模式 |
| `--enabled` | bool | 否 | - | 启用状态 |
| `--node-group` | ID | 否 | - | 节点组 |
| `--page` | int | 否 | `1` | 页码 |
| `--page-size` | int | 否 | `20` | 每页数量 |

#### `acl template get <id>`

`<id>` 必填。

#### `acl template create`

真实 3.0 必填：`mode`、`node_group_ids`、`match_method.scope`、`match_method.limit`、`match_method.period`、`match_method.target_type`、`match_method.policy`、`action_type`。当 scope 为 policy rule 时，`policy.policy_rule` 必填。部分 action 要求 action 配置。

参数：

| 参数 | 格式 | 必填 | 默认值 | 说明 |
| --- | --- | --- | --- | --- |
| `--name` | string | 是 | - | 模板名 |
| `--mode` | `forbidden|dryrun` | 是 | - | 模式 |
| `--node-group` | ID，可重复/逗号分隔 | 是 | - | 节点组 |
| `--scope` | `all|url|url-prefix|policy-rule|hook-rule` | 是 | - | 匹配范围，CLI 会映射为真实 API 的 `All|Url|UrlPrefix|PolicyRule|HookRule` |
| `--target-type` | `cidr|session|fingerprint` | 是 | - | 目标类型，CLI 会映射为真实 API 的 `CIDR|Session|Fingerprint` |
| `--period` | int 秒 | 是 | - | 统计周期 |
| `--limit` | int | 是 | - | 限制次数 |
| `--action-type` | `deny|dryrun-limit|rate-limit` | 是 | - | 处置动作 |
| `--policy-rule` | ID | 条件必填 | - | `--scope policy-rule` 时必填 |
| `--status-code` | int | 否 | `403` | deny/rate-limit 响应码 |
| `--response-file` | ID | 条件必填 | - | 使用自定义响应页时必填 |
| `--expire-period` | int 秒 | 否 | `0` | 过期时间，0 表示不过期 |
| `--enabled` | bool | 否 | `true` | 是否启用 |
| `--yes` | bool | 是 | - | 确认写操作 |

示例：

```bash
chaitin-cli safeline-3 acl template create \
  --name rate-limit-office \
  --mode forbidden \
  --node-group 1 \
  --scope all \
  --target-type cidr \
  --period 60 \
  --limit 100 \
  --action-type deny \
  --status-code 403 \
  --yes
```

#### `acl template update <id>`

行为：先 get 模板，合并字段后 PUT 完整 payload。`<id>` 和 `--yes` 必填。

#### `acl template enable <id...>` / `acl template disable <id...>`

`<id...>` 必填，`--yes` 必填。

#### `acl template delete <id...>`

`<id...>` 必填，`--yes` 必填。

#### `acl rule list`

| 参数 | 格式 | 必填 | 默认值 | 说明 |
| --- | --- | --- | --- | --- |
| `--template-id` | ID | 是 | - | ACL 模板 ID |
| `--page` | int | 否 | `1` | 页码 |
| `--page-size` | int | 否 | `20` | 每页数量 |

#### `acl rule delete <id>`

`<id>` 必填，`--template-id` 必填，`--yes` 必填。删除某个模板下所有已生成规则时使用 `--all --template-id <id> --yes`，此时不传 `<id>`。

### log

日志 API 的 `start_time` 和 `end_time` 是真实 3.0 硬必填；CLI 默认补最近 24 小时。

通用日志参数：

| 参数 | 格式 | 必填 | 默认值 | 说明 |
| --- | --- | --- | --- | --- |
| `--start` | 时间 | 否 | `-24h` | 转 `start_time`，Unix 毫秒 |
| `--end` | 时间 | 否 | `now` | 转 `end_time`，Unix 毫秒 |
| `--page` | int | 否 | `1` | 页码 |
| `--page-size` | int | 否 | `20` | 每页数量 |

#### `log attack list`

API：`POST /api/v3/protected-logger/DetectLogList`

参数：通用日志参数，以及：

| 参数 | 格式 | 必填 | 说明 |
| --- | --- | --- | --- |
| `--src-ip` | IP/CIDR | 否 | 来源 IP |
| `--host` | string | 否 | 域名 |
| `--url-path` | string | 否 | URL path |
| `--event-id` | string | 否 | Event ID |
| `--attack-type` | int | 否 | 攻击类型 |
| `--action` | int/string | 否 | 执行动作 |
| `--risk-level` | `low|medium|high|0|1|2|3` | 否 | 风险等级 |
| `--method` | HTTP method | 否 | 请求方法 |
| `--rule-id` | string/int | 否 | 规则 ID |

#### `log attack get <event-id>`

API：`GET /api/v3/protected-logger/DetectLogDetail`

`<event-id>` 必填，`--start` / `--end` 默认最近 24 小时。

#### `log attack decode <event-id>`

API：`GET /api/v3/protected-logger/DetectLogDecode`

参数同 `log attack get`。

#### `log access list`

API：`POST /api/v3/protected-logger/AccessLogList`

参数：通用日志参数，以及 `--src-ip`、`--host`、`--url-path`、`--status-code`、`--method`、`--event-id`。

#### `log access get <event-id>` / `log access decode <event-id>`

参数同 attack detail。

#### `log bot list`

API：`POST /api/v3/protected-logger/BotLogList`

参数：通用日志参数，以及 `--src-ip`、`--dst-ip`、`--is-bot`、`--bot-type`、`--country`。

#### `log bot get <event-id>`

API：`GET /api/v3/protected-logger/BotLogDetail`

`<event-id>` 必填。

### system

| 命令 | 参数 | 说明 |
| --- | --- | --- |
| `system license` | 无 | `GET /api/v3/license` |
| `system license-check` | 无 | `GET /api/v3/license/check` |
| `system machine-ids` | 无 | `GET /api/v3/management/machine_ids`；后端取单机或管理节点机器码，不需要 `--node-id` |
| `system time` | 无 | `GET /api/v3/system/time`；后端当前取管理节点时间 |
| `system set-time` | `--time` 时间，必填；`--yes` 必填 | `PUT /api/v3/system/time`；后端设置所有已知节点时间，不需要 `--node-id` |

不提供 `system reboot`、`system shutdown`。

### network

| 命令 | 参数 | 说明 |
| --- | --- | --- |
| `network overview` | `--node-id` 必填 | `GET /api/v3/network/overview` |
| `network links` | `--node-id` 必填 | `GET /api/v3/network/link/list` |
| `network network-service` | `--node-id` 可选 | `GET /api/v3/network/link/network_service` |
| `network vrrp` | `--node-id` 可选 | 查询 VRRP |

不提供 `network soft-bypass`、`network hard-bypass` 及其 enter/leave 子命令。

### monitor

| 命令 | 参数 | 说明 |
| --- | --- | --- |
| `monitor node-groups` | 无 | 兼容别名，等价于 `node-group list`；主入口使用 `node-group` |
| `monitor nodes` | 无 | 查询节点 |
| `monitor node-state-history` | `--node-id` 必填、`--start` 必填、`--end` 必填 | 查询节点状态历史 |
| `monitor node-state-extended-history` | `--node-id` 必填、`--start` 必填、`--end` 必填 | 查询节点扩展状态历史 |

### raw

高级兜底入口。只在 raw 命令中允许低层 HTTP 参数。

```bash
chaitin-cli safeline-3 raw request GET /api/v3/license
chaitin-cli safeline-3 raw request POST /api/v3/protected-logger/DetectLogList --body-file body.json
```

参数：

| 参数 | 格式 | 必填 | 说明 |
| --- | --- | --- | --- |
| `METHOD` | `GET|POST|PUT|PATCH|DELETE` | 是 | HTTP 方法 |
| `PATH` | `/api/v3/...` 或完整 URL | 是 | 请求路径 |
| `--param` | `key=value`，可重复 | 否 | query 参数 |
| `--body` | JSON 字符串 | 否 | 请求 body |
| `--body-file` | JSON 文件或 `-` | 否 | 请求 body |

`--body` 和 `--body-file` 互斥。
