# NoiseTrace 噪声源贡献归因台

```bash
docker compose up -d
```

打开 `http://localhost:18538`。Compose 会依次启动 PostgreSQL 16、Go API 和 Nginx 前端，三个服务通过健康检查后才进入可用状态。

NoiseTrace 面向职业卫生工程师、厂区声学分析员和独立复核人，用于维护监测点与版本化设备声源谱、导入倍频程测量、在能量域扣除背景声，并用确定性非负最小二乘计算候选声源贡献。

> 系统只提供离线决策支持与仿真，不连接或控制采集器、生产设备或降噪装置。结果不声称满足法定测量规范，也不能替代专业声学报告、现场核查或持证人员判断。

## 主要功能

- 监测点：维护三维坐标、受声区、责任团队和完整 63-8000 Hz 背景谱。
- 测量工作台：导入八个固定倍频程、生成 SHA-256 checksum、执行质量判定和受控状态迁移。
- 声源谱：维护设备位置、参考距离、方向性、运行系数和不可变频谱版本。
- 贡献归因：冻结测量、声源版本、算法版本和输入快照，输出逐频带贡献、总贡献、残差及不可辨识提示。
- 独立复核：运行发起人不能确认自己的结果；复核与确认使用条件更新，历史结果不可覆盖。
- 审计中心：每次业务写入在同一数据库事务中保存操作者、request ID、before/after 和算法元数据。

## 演示账号

| 账号 | 密码 | 角色 | 主要权限 |
| --- | --- | --- | --- |
| `admin` | `admin123` | admin | 全部管理、复核与审计 |
| `engineer` | `engineer123` | acoustic_engineer | 监测点、测量、声源谱和归因 |
| `analyst` | `analyst123` | data_analyst | 导入并推进本人测量、运行归因 |
| `reviewer` | `reviewer123` | reviewer | 独立复核、确认、作废和审计 |
| `auditor` | `auditor123` | auditor | 只读业务数据和审计记录 |

这些账号只用于本地演示。生产使用前必须更换密码和 `JWT_SECRET`。

## 页面与工作流

| 路由 | 实体消费 | 关键工作 |
| --- | --- | --- |
| `/points` | MonitoringPoint + NoiseMeasurement | 建点、背景谱、最近测量质量、停用 |
| `/measurements` | NoiseMeasurement + MonitoringPoint | 导入频谱、checksum、背景扣除和状态推进 |
| `/sources` | SourceProfile + MonitoringPoint | 新建版本、位置/参考距离、启用和废止 |
| `/attribution` | AttributionRun + NoiseMeasurement + SourceProfile | 冻结输入、运行拟合、查看证据、复核和确认 |
| `/audit` | 四实体审计投影 | 按实体、request ID、操作者筛选前后快照 |

标准流程：

```text
创建监测点
  -> 导入测量
  -> captured -> validated -> normalized -> ready
  -> 创建并启用声源谱版本
  -> 冻结输入并运行归因
  -> completed -> reviewed -> confirmed
```

非法状态跳转返回 HTTP 409。缺失频带、无效谱或不可接受的背景扣除返回 HTTP 422。权限不足返回 HTTP 403。

## 架构与目录

```text
Browser -> Nginx :80 -> Vue 3 SPA
                    -> /api/v1 -> Gin :8080 -> GORM -> PostgreSQL 16

backend/
  cmd/server
  internal/config constants dto model repository service handler router
  internal/middleware algorithm util
frontend/src/
  api stores types components/common hooks pages router utils
database/
  init.sql
output/
  本地验证报告与内置 Browser 截图
```

后端使用构造器注入，handler 不直接访问数据库。四个实体分别拥有 model、DTO、repository、service、handler 和 router；前端分别拥有 type、API、Pinia store 和页面消费链路。正式 Compose 使用 PostgreSQL，runtime smoke 使用内存 SQLite，但两者走同一迁移、种子、业务服务和 HTTP 入口。

## API 清单

响应统一包含 `code`、`message`、`data` 和 `request_id`。业务接口统一使用 `/api/v1`。

| 方法与路径 | 说明 |
| --- | --- |
| `GET /healthz` | 数据库连通性健康检查 |
| `POST /api/v1/auth/login` | 登录并签发 JWT |
| `GET/POST /api/v1/monitoring-points` | 监测点列表与创建 |
| `GET/PUT /api/v1/monitoring-points/:id` | 详情和带版本更新 |
| `POST /api/v1/monitoring-points/:id/deactivate` | 停用监测点 |
| `GET/POST /api/v1/noise-measurements` | 测量列表、详情和导入 |
| `GET /api/v1/noise-measurements/:id` | 测量详情与归一化谱 |
| `POST /api/v1/noise-measurements/:id/transition` | 测量状态迁移 |
| `GET/POST /api/v1/source-profiles` | 声源谱列表与新版本 |
| `GET /api/v1/source-profiles/:id` | 声源谱详情 |
| `POST /api/v1/source-profiles/:id/transition` | 启用或废止谱版本 |
| `GET/POST /api/v1/attribution-runs` | 归因历史和幂等运行 |
| `GET /api/v1/attribution-runs/:id` | 冻结输入与完整证据 |
| `GET /api/v1/attribution-runs/:id/compare/:other_id` | 历史结果差异 |
| `POST /api/v1/attribution-runs/:id/review` | 记录独立复核 |
| `POST /api/v1/attribution-runs/:id/confirm` | 独立确认 |
| `POST /api/v1/attribution-runs/:id/void` | 作废未确认结果 |
| `GET /api/v1/audit-logs` | 审计筛选 |
| `GET /api/v1/meta/enums` | 共享枚举与算法版本 |

归因接口读取 `Idempotency-Key`，但最终防重依据是数据库中的 `input_hash + algorithm_version` 复合唯一约束。同一冻结输入即使更换客户端 key，也会复用原运行。

## 算法与可解释证据

### 固定频带

全部谱必须且只能包含：

```text
63 / 125 / 250 / 500 / 1000 / 2000 / 4000 / 8000 Hz
```

### 背景扣除

声级不能直接做 dB 算术减法。NoiseTrace 先转到相对能量域：

```text
E = 10 ^ ((L - 100) / 10)
E_corrected = E_measured - E_background
L_corrected = 100 + 10 log10(E_corrected)
```

当测量与背景差小于 3 dB，频带会进入 `unreliable_bands`；若超过一半频带不可可靠扣除，测量不能迁移到 normalized。

### 传播与拟合

- 每个候选声源按三维距离执行球面扩散修正。
- 每频带同时使用方向性修正与运行系数。
- 使用 Gonum 矩阵和列归一化投影梯度 NNLS，系数始终非负。
- 输出传播矩阵规模、迭代次数、收敛状态、目标函数、列相关性提示和相对残差。
- 当传播谱高度相关时，明确给出“不可唯一辨识”提示，不伪造确定结论。

每个 AttributionRun 冻结测量 checksum、监测点背景、声源谱版本、坐标、算法版本和 canonical JSON SHA-256。相同输入可重放，旧结果不可覆盖。

## 状态机

```text
NoiseMeasurement:
captured -> validated -> normalized -> ready -> superseded
       \-> rejected      \-> rejected

AttributionRun:
queued -> calculating -> completed -> reviewed -> confirmed
                    \-> failed      \-> voided
```

状态更新使用 `id + current_state + version` 条件更新，避免并发先读后写覆盖。业务实体与审计记录在同一事务提交；审计写入失败会回滚业务写入。

## 共享枚举位置

`MeasurementQuality = valid | contaminated | clipped | missing`

- 后端定义：`backend/internal/constants/measurement_quality.go`。
- 后端消费：NoiseMeasurement model/DTO、测量 service 质量判定、handler/router 响应链和算法/状态测试。
- 前端定义：`frontend/src/types/enums/measurement-quality.ts`。
- 前端消费：measurement type/store、`QualityBadge`、监测点/测量/归因页面和枚举测试。

`AttributionState = queued | calculating | completed | failed | reviewed | confirmed | voided`

- 后端定义：`backend/internal/constants/attribution_state.go`。
- 后端消费：AttributionRun model/DTO、service 状态机、repository 条件更新、handler/router 和状态测试。
- 前端定义：`frontend/src/types/enums/attribution-state.ts`。
- 前端消费：attribution type/store、`StateBadge`、归因页面、`AttributionDetailDrawer` 和枚举测试。

## 技术栈

| 层 | 技术 |
| --- | --- |
| 前端 | Vue 3、TypeScript、Vite、Element Plus、Pinia、ECharts、Lucide |
| 后端 | Go 1.22、Gin、GORM、Gonum、JWT、bcrypt |
| 数据库 | PostgreSQL 16；SQLite runtime smoke |
| 部署 | Docker Compose、Nginx Alpine、三服务健康依赖 |

## 环境变量与端口

新克隆的仓库先准备环境文件：

```bash
cp .env.example .env
docker compose up -d
```

| 服务 | 宿主机端口 | 容器端口 |
| --- | ---: | ---: |
| 前端 | `18538` | `80` |
| 后端 | `19538` | `8080` |
| PostgreSQL | `57538` | `5432` |
| runtime smoke | `20538` | 本机进程 |

关键变量：`DB_NAME`、`DB_USER`、`DB_PASSWORD`、`DB_DRIVER`、`JWT_SECRET`、`JWT_EXPIRY`、三个接口限流值和三个宿主机端口。`JWT_SECRET` 至少 16 字符；不要把 `.env`、访问令牌或真实凭据提交到 Git。

## 本地开发

后端可使用 SQLite：

```bash
cd backend
PORT=19538 \
DB_DRIVER=sqlite \
DB_DSN='file:local.db' \
JWT_SECRET='local-development-secret' \
go run ./cmd/server
```

另一个终端启动前端：

```bash
cd frontend
npm ci
npm run dev
```

Vite 把 `/api` 代理到 `http://127.0.0.1:19538`。前端始终使用相对 `/api/v1`，不包含静态业务 mock。

## 构建与测试

```bash
go work sync
go build ./backend/...
go vet ./backend/...
go test ./backend/...
go test -race ./backend/...

cd backend
go build ./...
go vet ./...
go test ./...
go test -race ./...

cd ../frontend
npm ci
npm test
npm run typecheck
npm run build
npm audit --registry=https://registry.npmjs.org
npm run audit:prod
```

普通项目测试覆盖 dB 能量换算、频带完整性、已知合成 NNLS 夹具、状态机、RBAC、权限和序列化边界。

## Runtime smoke

```bash

```

脚本读取根目录 `runtime_smoke.json`，在 `20538` 启动真实 Gin + 内存 SQLite，访问 `/healthz` 后停止进程。正式 Compose 不使用 SQLite。

## Docker 检查与停止

```bash
docker compose config --quiet
docker compose up -d --build
docker compose ps
```

健康端点：

- 后端：`http://localhost:19538/healthz`
- 前端代理：`http://localhost:18538/api/healthz`

停止并删除本项目容器、网络和 PostgreSQL 命名卷：

```bash
docker compose down -v --remove-orphans
```

## 常见问题

- `401`：令牌缺失、无效或已过期，重新登录。
- `403`：当前角色无对应权限；数据分析员不能推进他人导入的测量，运行发起人不能确认自己的结果。
- `409`：实体状态或乐观锁版本已变化，刷新页面后按最新状态操作。
- `422`：检查 8 个固定频带、背景谱、方向性范围、测量质量和 ready/active 输入状态。
- 高残差：候选声源不完整、测量条件不一致或背景扣除不可靠；系统不会猜测缺失声源。
- 不可辨识提示：候选源传播列高度相关，应增加监测点或独立工况，不能只凭当前排序下结论。

## License

MIT License。使用方仍须自行遵守当地职业卫生、环境噪声、测量和工程签认要求。
