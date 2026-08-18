<div align="center">
  <img src="web/public/logo.png" width="108" alt="ZHIYU-WAF V2 logo" />

  # ZHIYU-WAF V2

  **本地优先 · 可解释风险决策 · 免费自部署的 Web 应用防火墙**

  [![Go](https://img.shields.io/badge/Go-1.25-00ADD8?logo=go&logoColor=white)](go.mod)
  [![Vue](https://img.shields.io/badge/Vue-3-42B883?logo=vuedotjs&logoColor=white)](web/)
  [![Firewall](https://img.shields.io/badge/Firewall-nftables-8B5CF6)](#内核封禁与降级语义)
  [![License](https://img.shields.io/badge/License-AGPL--3.0-2563EB)](LICENSE)
  [![Control%20Plane](https://img.shields.io/badge/Control%20Plane-V2%20Free-16A34A)](#v2-控制面)

  [快速开始](#快速开始) · [核心能力](#核心能力) · [部署模式](#部署模式) · [API](#v2-控制面) · [生产清单](#生产上线清单)
</div>

---

> **ZHIYU-WAF V2 是一个独立演进的开源项目。** 它取消了版本授权、联网激活、功能套餐与专业版门控；代码中已实现的控制面与防护能力均可免费使用。V2 将每次请求的检测证据汇聚到统一风险引擎，并在本地输出唯一、可审计的最终动作。

<div align="center">

| 本地优先 | 防护可解释 | 运行可降级 | 控制面免费 |
| :---: | :---: | :---: | :---: |
| 规则、事件与封禁状态均落在本地 | 每个动作均可追溯到风险证据 | 内核不可用时保留应用层阻断 | AI、站点、情报、RBAC 与大屏直接开放 |

</div>

## 项目概览

ZHIYU-WAF V2 以 Go 反向代理承接业务流量，并在本地完成规范化、访问控制、速率信号、规则匹配、行为分析与风险聚合。规则引擎只产生检测证据，风险引擎统一决定 `allow`、`log`、`rate_limit` 或 `block`；这避免了多个检测器各自阻断造成的重复事件和策略冲突。

外部 AI 是**可选的异步辅助能力**，不是核心防护链的依赖。未配置模型、模型超时或外部网络不可达时，基础规则、风险判定和 HTTP 阻断均继续运行。

```mermaid
flowchart LR
  A[Client] --> B[Ingress / Nginx / Load Balancer]
  B --> C[ZHIYU-WAF V2 Proxy]
  C --> D[规范化与可信客户端 IP]
  D --> E[ACL · 频率信号 · 规则检测 · 行为画像]
  E --> F[统一风险引擎]
  F -->|ALLOW / LOG| G[业务后端]
  F -->|RATE LIMIT / BLOCK| H[应用层响应]
  F -->|高风险或人工封禁| I[(SQLite 封禁记录)]
  I --> J[nftables IPv4 / IPv6 集合]
```

## 核心能力

| 模块 | V2 行为 | 关键价值 |
| --- | --- | --- |
| **本地规则引擎** | 支持 SQL 注入、XSS、命令注入、路径穿越、敏感路径等检测证据 | 检测与最终动作解耦，规则可审计、可维护 |
| **统一风险引擎** | 汇聚 ACL、限速、规则、行为与可选 AI 结果 | 每个请求只输出一次最终动作，避免重复阻断 |
| **四级风险动作** | `allow`、`log`、`rate_limit`、`block` | 对低风险观测、高风险拦截提供清晰分层 |
| **nftables 封禁** | 使用 Go Netlink 管理独立 `zhiyu_waf` 表及 IPv4/IPv6 超时集合 | 不执行 `iptables`、`ip6tables` 或防火墙 shell 命令 |
| **本地安全存储** | SQLite 保存最小化安全事件、站点、封禁与风险决策 | 重启可回放未过期临时封禁，减少状态丢失 |
| **多站点防护** | 按域名配置回源与 `monitor`、`protect`、`emergency` 模式 | 支持从观察到正式防护的渐进上线 |
| **治理控制面** | 规则、IP、地理策略、证书、SSH、审计、备份与 RBAC | 白色极简控制台，所有已实现模块均免费开放 |
| **AI 辅助分析** | OpenAI 兼容模型、异步 fail-open、配置与审计 | 可选增强，不影响本地核心防护链可用性 |

### 设计原则

| 原则 | 说明 |
| --- | --- |
| **Local First** | 防护决策、策略、事件与封禁状态优先在本机完成，不依赖中心化许可证服务。 |
| **Evidence Before Action** | 检测器输出证据，风险引擎根据分数、置信度和策略决定唯一动作。 |
| **Explicit Enforcement** | V2 不使用 `iptables REDIRECT` 劫持业务端口；流量通过显式反代进入 WAF。 |
| **Safe Degradation** | nftables 不可写时仍执行应用层判定与 HTTP 阻断，并公开报告降级状态。 |
| **Privacy by Default** | 安全事件保存定位、关联和审计所需字段，不把完整请求体写入事件表。 |

## 快速开始

### 1. 获取代码并构建控制台

> **运行前置：** Go `1.25+`、Node.js `20+`、CGO 可用，以及支持 `nftables` 的 Linux 环境（仅内核封禁需要）。

```bash
git clone https://github.com/JayLee-sre/ZHIYU-WAF.git
cd ZHIYU-WAF

# 构建 Vue 控制台并嵌入 Go 二进制
cd web
npm ci
npm run build:raw
cd ..
rm -rf internal/dashboard/dist
cp -R web/dist internal/dashboard/dist

# 测试与构建
CGO_ENABLED=1 go test ./...
CGO_ENABLED=1 go build -o bin/zhiyu-waf ./cmd/zhiyu-waf
```

### 2. 启动服务

```bash
./bin/zhiyu-waf -config configs/zhiyu-waf.yaml
```

默认情况下，WAF 监听 `:8080`，管理控制台监听 `:9090`。首次运行请立即完成初始化并修改管理员密码；管理端应限制在私有网络、跳板机或受控 VPN 中访问。

```text
业务入口      →  http://<host>:8080
管理控制台    →  http://<host>:9090
健康检查      →  GET /health
系统状态      →  GET /api/v1/system/status
```

## 部署模式

### 显式反向代理接入

V2 不再通过 `iptables REDIRECT` 接管业务端口。请让已有的 Ingress、Nginx、负载均衡器或网关显式把业务流量转发到 WAF，再由 WAF 回源到应用服务。

```text
Client
  │
  ▼
Ingress / Nginx / Load Balancer
  │  forwards to WAF :8080
  ▼
ZHIYU-WAF V2 Pipeline
  │  forwards to configured backend
  ▼
Application Backend
```

Nginx 应传递 `Host`、`X-Real-IP` 与 `X-Forwarded-For`。只有 `proxy.trusted_proxies` 中的直接对端才会被信任其转发头；请勿将未受控公网网段加入此列表。

```nginx
location / {
    proxy_pass http://127.0.0.1:8080;
    proxy_set_header Host $host;
    proxy_set_header X-Real-IP $remote_addr;
    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    proxy_set_header X-Forwarded-Proto $scheme;
}
```

### 站点防护模式

| 模式 | 适用阶段 | 行为 |
| --- | --- | --- |
| `monitor` | 上线前验证、规则调优 | 记录风险证据与事件，不主动执行防护动作。 |
| `protect` | 常规生产运行 | 按统一风险决策执行放行、记录、限流或阻断。 |
| `emergency` | 紧急维护、故障处置 | 优先进入维护策略，便于快速缩小暴露面。 |

## 内核封禁与降级语义

V2 的防火墙管理器只操作名为 `zhiyu_waf` 的独立 nftables 表，并维护 IPv4 与 IPv6 封禁集合。内核封禁需要 `CAP_NET_ADMIN` 或宿主机等效权限。

| 运行状态 | WAF 行为 | 运维建议 |
| --- | --- | --- |
| **nftables 可用** | 高风险或人工封禁同步到内核超时集合 | 监控集合写入与封禁 TTL。 |
| **degraded** | 保持规则、风险判定与应用层 HTTP 阻断；内核 IP 封禁不可用 | 检查 Capability、内核支持与 Netlink 错误，并接入告警。 |

| 风险分数 | 默认封禁时长 |
| ---: | :--- |
| 85–91 | 10 分钟 |
| 92–97 | 1 小时 |
| 98–100 | 24 小时 |

SQLite 中尚未到期的临时封禁会在重启时按**剩余时间**回放，不会因重启被无意延长。

## 配置导航

默认配置见 [`configs/zhiyu-waf.yaml`](configs/zhiyu-waf.yaml)。下面列出生产部署最常需要调整的关键项。

| 配置项 | 说明 | 生产建议 |
| --- | --- | --- |
| `proxy.listen_addr` | WAF HTTP 监听地址 | 使用受控入口或内部地址。 |
| `proxy.backend_addr` | 默认回源服务地址 | 配合站点级回源逐步切换。 |
| `proxy.trusted_proxies` | 可影响真实来源 IP 的可信代理 CIDR/IP | 仅保留直接受控代理。 |
| `dashboard.listen_addr` | 控制台监听地址 | 限制至内网、VPN 或跳板机。 |
| `dashboard.jwt_secret` | 管理会话签名密钥 | 使用随机强密钥或外部密钥管理。 |
| `engine.rules_dir` | 本地 YAML 规则目录 | 通过版本控制或受控发布流程更新。 |
| `storage.path` | SQLite 数据库路径 | 置于可靠磁盘并纳入备份。 |
| `ai.enabled` | 可选 AI 辅助开关 | 先完成数据脱敏与出网审批。 |

## V2 控制面

所有已实现的控制面能力均免费开放。受 JWT 保护的 V2 API 使用统一包装：成功响应包含 `data` 与 `request_id`；失败响应包含 `error.code`、`error.message` 与 `request_id`。

| 资源 | 常用端点 | 用途 |
| --- | --- | --- |
| Dashboard 汇总 | `GET /api/v1/dashboard/summary?range=24h` | 请求、阻断、风险来源与类别汇总。 |
| Dashboard 趋势 | `GET /api/v1/dashboard/timeseries?range=24h` | 时序请求、阻断与高风险变化。 |
| 安全事件 | `GET /api/v1/events` | 统一查询本地风险事件。 |
| 事件详情 | `GET /api/v1/events/{id}` | 获取单个事件的证据与最终动作。 |
| 站点 | `GET/POST /api/v1/sites` | 管理域名、回源、模式和启用状态。 |
| 封禁 IP | `GET/POST /api/v1/blocked-ips` | 查看、手工加入或删除本地封禁。 |
| 系统状态 | `GET /api/v1/system/status` | 查看防火墙、数据库和全功能免费状态。 |
| 可观测性 | `GET /metrics` | 暴露 Prometheus 指标。 |

## 数据与隐私

安全事件只保存定位、审计与关联所需的最小字段，例如来源 IP、Host、路径、规则标识、风险分数与最终动作。完整请求体不会写入事件表。若启用 AI 辅助，请单独评估数据脱敏、出网审批、供应商条款和业务数据合规要求。

## 生产上线清单

在将生产流量逐步接入 V2 前，建议先在 `monitor` 模式下观察规则命中，再分站点切换到 `protect`。以下项目应在变更窗口前完成验证。

- [ ] 回源可达性、上传、流式响应与 WebSocket 兼容性符合业务预期。
- [ ] 可信代理列表仅包含受控直接对端，来源 IP 不可被伪造。
- [ ] 规则误报处置、IP 白名单与站点模式切换流程已经演练。
- [ ] IPv4/IPv6 封禁、nftables 降级告警和重启回放均已验证。
- [ ] SQLite 数据库、配置文件和规则目录已纳入备份与恢复演练。
- [ ] 管理控制台已限制在受控网络，并启用强管理员密码与最小权限账户。

## 开发与质量检查

```bash
# 前端生产构建
cd web && npm ci && npm run build:raw && cd ..
rm -rf internal/dashboard/dist && cp -R web/dist internal/dashboard/dist

# 服务端测试、竞态检查与静态检查
CGO_ENABLED=1 go test ./...
go test -race ./...
go vet ./...
CGO_ENABLED=1 go build ./cmd/zhiyu-waf
```

## 社区与贡献

欢迎以 Issue、讨论和 Pull Request 的方式参与 V2 演进。提交前请保证代码通过格式检查、测试与构建，并清楚说明变更的防护语义、兼容影响和验证步骤。对于规则、风险策略和防火墙相关修改，请同时提供可复现的安全事件样例或测试用例。

## 许可证

本项目采用 [AGPL-3.0](LICENSE) 许可证。将本项目用于网络服务、再分发或二次交付前，请评估相应的开源义务。

---

<div align="center">
  <sub>ZHIYU-WAF V2 · Local First · Evidence Driven · Free Control Plane</sub>
</div>
