# SafeLine-3 CLI 与产品 AI 化定位

本文档记录 `chaitin-cli safeline-3` 的定位判断，以及 SafeLine-3 长期 AI 化时 CLI、API schema、业务语义应该如何归属。

## 结论

`workspace-cli` 中的 `safeline-3` CLI 适合作为过渡层、验证层和原型层，不适合作为 SafeLine-3 长期的业务语义权威来源。

如果目标是让 SafeLine-3 产品本身面向 AI Agent 变得可操作、可解释、可维护，那么“业务语义”应该沉淀到 SafeLine-3 产品自身，而不是长期保存在外挂 CLI 项目里。

这里的业务语义包括：

- 节点组工作模式与防护对象类型的兼容关系。
- `site create`、`listener create` 等命令的必填、条件必填和默认值。
- TLS listener 与证书参数的关系。
- 哪些字段首版不暴露，例如国密证书、防护组、标签。
- 哪些操作是高风险操作，以及是否必须二次确认。
- 日志查询时间范围默认值。
- 系统和网络命令哪些允许暴露、哪些不允许暴露。
- AI 需要理解的参数格式、枚举、示例和错误提示。

这些规则本质上属于 SafeLine-3 产品操作模型，而不是某个外部 CLI 工具的私有逻辑。

## 为什么外挂 CLI 不能成为长期权威

如果业务规则长期只存在于 `workspace-cli/products/safeline3`，会出现几个问题：

- **规则分叉**：前端、后端、CLI、AI schema 各自维护一份理解，版本迭代后容易不一致。
- **漂移不可见**：SafeLine-3 后端新增字段、修改 required、调整枚举后，CLI 可能继续通过测试，但已经不符合真实产品能力。
- **产品自身缺少 AI 操作面**：AI 只能依赖外挂工具，而不是从产品自身获得稳定的操作描述。
- **维护成本持续升高**：越多业务逻辑进入外挂 CLI，后续每次产品迭代都需要人工同步。
- **测试证明范围有限**：外挂 CLI 的单测只能证明 CLI 自己没坏，不能证明 SafeLine-3 产品的 AI 化契约完整。

因此，外挂 CLI 可以先做，但不能让它成为业务语义的唯一来源。

## 更合理的长期形态

SafeLine-3 产品自身应该提供一个稳定的机器交互层。这个交互层可以表现为：

- 产品内置 CLI，或官方 CLI 包。
- 机器可读的 command manifest。
- 机器可读的 entity manifest，用于描述实体、关系、兼容矩阵和风险等级。
- OpenAPI / tool schema / action schema。
- 参数 required、optional、condition-required 描述。
- 枚举、格式、默认值、示例。
- 高风险操作标记和确认策略。
- 不同版本之间的兼容声明。

HTTP API 是产品能力面，AI CLI / tool schema 是产品操作面。两者相关，但不等价。

长期更合适的依赖方向应该是：

```text
SafeLine-3 产品仓库
  -> 导出 API schema / command manifest / tool schema
  -> workspace-cli 消费这些描述
  -> 生成或校验 safeline-3 CLI
```

而不是：

```text
workspace-cli 手写并长期维护 SafeLine-3 业务规则
```

## 当前 workspace-cli 的合理定位

当前 `workspace-cli/products/safeline3` 仍然有价值，但定位应明确为：

- 验证 SafeLine-3 面向 AI 的命令形态是否合理。
- 验证“实体命令 + raw 逃生口”的交互模型是否足够清晰。
- 在产品内置机器交互层完成前，提供临时可用的 CLI。
- 反向推动 SafeLine-3 后端补齐 OpenAPI、required、枚举和 schema。
- 沉淀实体关系、命令分层、参数设计、help 结构和测试思路。
- 作为跨产品统一入口的一种适配层。

它不应该长期承担：

- 定义 SafeLine-3 业务规则。
- 独立决定后端字段是否必填。
- 独立维护节点组、防护对象、listener、策略等业务关系。
- 作为 AI 操作 SafeLine-3 的唯一依据。

## 建议演进路径

### 短期

继续保留 `workspace-cli` 中的 `safeline-3` CLI，用它验证命令形态和 AI 可用性。

当前阶段只做实体命令和 `raw request`，不实现短命令。短命令后续只能作为高频实体命令的薄包装，不承担复杂业务工作流编排。

测试重点放在：

- CLI 命令树和 help 是否稳定。
- 业务参数是否生成正确 HTTP 请求。
- 高风险操作是否强制 `--yes`。
- 明确不暴露的参数不会出现在 CLI 中。

### 中期

在 SafeLine-3 产品仓库中沉淀机器可读的操作描述。

优先沉淀：

- Entity manifest。
- API manifest。
- request/response schema。
- required 和 condition-required 规则。
- 实体关系和兼容矩阵。
- 枚举值和 CLI 短名映射。
- 操作风险等级。
- help/example 元数据。

`workspace-cli` 改为消费这些描述，并用测试发现漂移。

### 长期

SafeLine-3 产品自身拥有 AI 操作面。

`workspace-cli` 只作为消费者和跨产品统一入口：

- 读取 SafeLine-3 官方 manifest。
- 生成或装配 Cobra 命令。
- 处理统一配置、输出格式、dry-run、认证等外壳能力。
- 不再手写产品核心业务规则。

## 对测试策略的影响

在这个定位下，测试不应只验证当前手写 CLI 是否通过，而要验证 CLI 是否仍然符合产品声明的操作契约。

建议测试分层：

- **Command Manifest Test**：生成当前 Cobra 命令树，与期望命令清单对比。
- **Entity Model Test**：验证实体关系、兼容矩阵和禁止暴露参数符合 manifest。
- **Help Golden Test**：关键命令 help 必须包含 API、必填参数、格式和示例。
- **HTTP Contract Test**：用 `httptest.Server` 验证 CLI 参数最终生成的 method、path、query、body。
- **Spec Drift Test**：从 SafeLine-3 产品仓库生成 API/command manifest，与 workspace-cli 中引用的版本对比。
- **Integration Smoke Test**：在真实 SafeLine-3 测试环境中只跑少量只读接口。

其中 Spec Drift Test 是长期关键。它的目的不是自动解决所有变化，而是让产品迭代导致的 CLI 漂移变成可见失败。

## 判断标准

后续如果某个规则满足以下条件，应优先考虑沉淀到 SafeLine-3 产品仓库，而不是只写在 workspace-cli 中：

- 前端也需要同样规则。
- 后端 API required 或 enum 会影响它。
- AI Agent 需要通过该规则决定下一步操作。
- 规则会随产品版本变化。
- 规则错误会造成配置失败、误删、误停、误暴露高风险操作。

符合这些条件的内容，就是产品操作语义，应由 SafeLine-3 产品自身拥有。
