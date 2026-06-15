# SafeLine-3 CLI Agent Skill

本文档指导 AI Agent 使用 `chaitin-cli safeline-3` 操作 SafeLine-3。目标是让 Agent 会选命令、会补参数、会控制写操作风险，而不是直接拼 API。

## 总原则

- 默认前缀：`chaitin-cli safeline-3`。
- 不确定命令或参数时先查 help：`safeline-3 --help`、`safeline-3 site create reverse-proxy --help`。
- 查询结果给 AI 继续处理时使用 `--output json`。
- 写操作必须带 `--yes`；不确定时先用命令级 `--check` 或根级 `--dry-run`。
- 优先使用实体命令；只有实体命令缺失接口或字段时才用 `raw request`。
- 复杂嵌套结构使用文件参数，例如 `--payload-file`、`--application-file`、`--condition-file`。

常用全局参数：

```text
--url URL
--api-token TOKEN
--output table|json
--dry-run
--verbose
--insecure
```

## 实体和关系

优先使用这些实体命令：

```text
node-group
site
listener
ip-group
policy-group
policy-rule
acl
log
monitor
system
network
raw
```

核心关系：

- `node-group` 是工作模式边界，决定可创建的 `site` 类型和 listener 类型。
- `site` 是防护对象。
- 代理类 `site` 使用已有 listener ID，也就是 `--listener`。
- 非代理类 `site` 使用远端 listener，也就是 `--remote-listener IP:PORT` 或 `IP/CIDR:PORT`。
- `listener` 属于 `node-group`。
- `policy-group` 可被 `site` 或 application 引用。
- `policy-rule` 可全局生效，也可绑定 application。
- `acl template` 作用于节点组，并可生成 `acl rule`。

## site 创建决策

创建或更新 `site` 前，先查节点组能力：

```bash
chaitin-cli safeline-3 node-group list --output json
chaitin-cli safeline-3 node-group capabilities <node_group_id> --output json
```

兼容矩阵：

| node-group mode | 可创建 site 类型 | listener 参数 |
| --- | --- | --- |
| `reverse-proxy` | `reverse-proxy` | `--listener`，引用 reverse-proxy listener |
| `route-proxy` | `route-proxy` | `--listener`，引用 route-proxy listener |
| `transparent` | `transparent-proxy`, `transparent` | `transparent-proxy` 用 `--listener`；`transparent` 用 `--remote-listener` |
| `mirror` | `transparent`, `mirror` | `--remote-listener` |
| `sdk` | `sdk` | `--remote-listener`；仅后端已有 SDK/software 节点组时可用 |

决策流程：

```text
1. 查 node-group capabilities。
2. 判断目标 site 类型是否兼容。
3. 如果是 reverse-proxy / route-proxy / transparent-proxy，先 listener list，缺失则 listener create 对应类型。
4. 如果是 transparent / mirror / sdk，准备 --remote-listener。
5. 用 site create/update 执行；不确定先 --check。
```

## 最小任务模板

反向代理：

```bash
chaitin-cli safeline-3 listener list --node-group 1 --output json
chaitin-cli safeline-3 listener create reverse-proxy --name web-https --node-group 1 --ip 0.0.0.0 --port 443 --tls --http2 --yes
chaitin-cli safeline-3 site create reverse-proxy --name app --node-group 1 --domain app.example.com --listener 10 --upstream http://10.0.0.1:8080 --yes
```

路由代理：

```bash
chaitin-cli safeline-3 listener create route-proxy --name route-http --node-group 2 --ip 10.0.0.0/24 --port 80 --inbound-ip 10.0.0.10 --yes
chaitin-cli safeline-3 site create route-proxy --name route-app --node-group 2 --domain app.internal --listener 20 --yes
```

透明代理：

```bash
chaitin-cli safeline-3 node-group network 3 --output json
chaitin-cli safeline-3 listener create transparent-proxy --name tp-http --node-group 3 --ip 10.0.0.10 --port 80 --virtual-wire-pair wire0 --yes
chaitin-cli safeline-3 site create transparent-proxy --name tp-app --node-group 3 --domain app.internal --listener 31 --yes
```

非代理类 site：

```bash
chaitin-cli safeline-3 site create transparent --name transparent-app --node-group 3 --domain app.internal --remote-listener 10.0.0.10:80 --yes
chaitin-cli safeline-3 site create mirror --name mirror-app --node-group 4 --domain mirror.internal --remote-listener 10.0.0.10:80 --yes
```

修改和删除 site：

```bash
chaitin-cli safeline-3 site get 12 --type reverse-proxy --output json
chaitin-cli safeline-3 site update reverse-proxy 12 --domain app.example.com --check
chaitin-cli safeline-3 site update reverse-proxy 12 --domain app.example.com --yes
chaitin-cli safeline-3 site disable 12 --yes
chaitin-cli safeline-3 site delete 12 --check
chaitin-cli safeline-3 site delete 12 --yes
```

节点组：

```bash
chaitin-cli safeline-3 node-group get 1 --with-nodes --with-network --output json
chaitin-cli safeline-3 node-group set-mode 1 --mode transparent --check
chaitin-cli safeline-3 node-group set-mode 1 --mode transparent --yes
```

`node-group set-mode` 是高风险操作，会影响 listener 和网络配置。单机默认节点组不可 `update/delete`；改变工作模式使用 `set-mode`。硬件集群环境才创建、更新、删除非默认节点组。

策略和 ACL：

```bash
chaitin-cli safeline-3 policy-group list --output json
chaitin-cli safeline-3 policy-group create --name prod-policy --template-id 1 --yes
chaitin-cli safeline-3 policy-group rename 3 --name prod-policy-new --yes
chaitin-cli safeline-3 policy-rule create simple --name block-ip --target src_ip --op in --value 10.0.0.1 --action block --bind global --yes
chaitin-cli safeline-3 acl template list --node-group 1 --output json
```

日志：

```bash
chaitin-cli safeline-3 log attack list --start -24h --page 1 --page-size 20 --output json
chaitin-cli safeline-3 log access list --host app.example.com --status-code 500 --output json
chaitin-cli safeline-3 log bot list --src-ip 1.2.3.4 --is-bot true --output json
```

系统和网络：

```bash
chaitin-cli safeline-3 system license --output json
chaitin-cli safeline-3 system machine-ids --output json
chaitin-cli safeline-3 system time --output json
chaitin-cli safeline-3 system set-time --time now --check
chaitin-cli safeline-3 network overview --node-id <node_id> --output json
chaitin-cli safeline-3 network links --node-id <node_id> --output json
```

`system set-time` 影响所有已知节点，没有 `--node-id`。

## 参数格式

- ID：正整数，例如 `12`。
- ID 列表：可重复传参或逗号分隔，例如 `--node 1 --node 2` 或 `--node 1,2`。
- 时间：RFC3339、`YYYY-MM-DD HH:MM:SS`、`YYYY-MM-DD`、Unix 秒、Unix 毫秒、`now`、相对时间如 `-24h`。
- upstream：`http://HOST:PORT[,weight=N]` 或 `https://HOST:PORT[,weight=N]`。
- remote listener：`IP:PORT` 或 `IP/CIDR:PORT`。
- 文件参数：路径或 `-`，`-` 表示 stdin。

## raw request

只有实体命令缺失时才使用：

```bash
chaitin-cli safeline-3 raw request GET /api/v3/license --output json
chaitin-cli safeline-3 raw request POST /api/v3/protected-logger/DetectLogList --body-file body.json --output json
```

raw 参数：

```text
--param key=value
--body JSON
--body-file PATH|-
```

不要使用 raw 绕过 `--yes`、节点组兼容性、高风险限制或禁止参数。

## 禁止使用

不要生成这些命令或参数：

```text
safeline-3 protected-object ...
safeline-3 log-event ...
safeline-3 sites
safeline-3 node-groups
safeline-3 attacks
safeline-3 access-logs
safeline-3 bot-logs
safeline-3 time
safeline-3 license
safeline-3 system reboot
safeline-3 system shutdown
safeline-3 network soft-bypass
safeline-3 network hard-bypass
--gm-cert-id
--protected-group
--tag
--query
--request
--request-file
```

## 失败处理

- 节点组不兼容：重新执行 `node-group get` 和 `node-group capabilities`，不要强行换参数或 raw。
- `--listener is required`：当前类型是代理类，先 `listener list` 或创建对应 listener。
- `--remote-listener is required`：当前类型是非代理类，传 `--remote-listener`，不要传 `--listener`。
- TLS 失败：先确认 listener 是否启用 TLS；普通证书用 `--cert-id`；不要使用国密证书参数。
- 复杂配置失败：查子命令 `--help`；语义化参数无法表达时使用 `--payload-file` 或对应 `*-file`。
