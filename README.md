# ZHIYU-WAF V2

**ZHIYU-WAF V2** 是面向自部署场景的本地优先 Web 应用防火墙。它以 Go 反向代理为入口，以统一风险引擎输出单一防护动作，并将短期恶意来源封禁同步到独立的 `nftables` 表。项目**不包含版本授权、在线激活、功能套餐或专业版门控**；已实现的控制面功能均可免费使用。

> V2 的核心原则是：请求先产生可解释的检测证据，再由风险引擎统一决定 `allow`、`log`、`rate_limit` 或 `block`。外部 AI 属于可选的异步辅助能力，网络故障不会中断核心防护链。

| 组件 | 责任 |
| --- | --- |
| V2 Pipeline | 规范化、ACL、频率信号、规则检测、行为画像与统一风险决策 |
| Rule Engine | 兼容本地 YAML 规则，输出检测证据而非自行阻断 |
| Risk Engine | 合并证据、抑制同类重复命中、计算风险分数与最终动作 |
| Firewall | 使用 Go Netlink 管理独立 `nftables` 表与 IPv4/IPv6 超时集合 |
| SQLite | 保存最小化安全事件、风险事件、站点和封禁状态，并支持重启回放 |
| Dashboard | Vue 管理控制面，提供趋势、事件、站点、封禁与运行状态视图 |

## 已实现能力

| 能力 | V2 行为 |
| --- | --- |
| SQL 注入、XSS、命令注入、路径穿越等规则 | 本地规则输出 Detection，由风险层统一判定 |
| IP 黑白名单与限流 | 作为 ACL/频率证据进入风险链；白名单可跳过后续检测 |
| 风险动作 | `allow`、`log`、`rate_limit`、`block` 四级动作 |
| 恶意 IP 封禁 | 高风险或人工封禁写入 SQLite 并同步 `nftables` 超时集合 |
| IPv4 / IPv6 | 同时支持 IPv4、IPv6 封禁集合与 SSH 暴力破解联动 |
| 多站点 | 按域名维护回源与 `monitor`、`protect`、`emergency` 模式 |
| 安全审计 | 事件、风险、封禁和管理操作均通过本地存储查询 |
| AI 辅助 | 默认关闭、可选配置、异步 fail-open；不依赖外部 AI 才能保护流量 |
| 免费控制面 | 站点、AI、地理策略、情报、RBAC 与大屏不再存在版本门控 |

## 快速开始

### 从源码运行

请使用 Go 1.25 或与 `go.mod` 兼容的工具链，并保留 CGO 以支持 SQLite。

```bash
git clone https://github.com/JayLee-sre/ZHIYU-WAF.git
cd ZHIYU-WAF

cd web
npm ci
npm run build:raw
cd ..
rm -rf internal/dashboard/dist
cp -R web/dist internal/dashboard/dist

CGO_ENABLED=1 go test ./...
CGO_ENABLED=1 go build -o bin/zhiyu-waf ./cmd/zhiyu-waf
./bin/zhiyu-waf -config configs/zhiyu-waf.yaml
```

默认情况下，WAF 监听 `:8080`，管理后台监听 `:9090`。首次运行应立刻修改管理员密码，并将管理端限制在私有网络、跳板机或受控 VPN 中。

### 反向代理接入

V2 不再使用 `iptables REDIRECT` 接管业务端口。请让已有的 Nginx、负载均衡器或服务网关将业务流量显式转发到 WAF，再由 WAF 回源到应用服务。

```text
Client
  │
  ▼
Ingress / Nginx / Load Balancer
  │  forwards to :8080
  ▼
ZHIYU-WAF V2 Pipeline
  │  forwards to backend_addr
  ▼
Application Backend
```

Nginx 应传递 `Host`、`X-Real-IP` 与 `X-Forwarded-For`。只有 `proxy.trusted_proxies` 中列出的直接对端才会被信任其转发头；不要将不受控制的公网网段加入该列表。

```nginx
location / {
  proxy_pass http://127.0.0.1:8080;
  proxy_set_header Host $host;
  proxy_set_header X-Real-IP $remote_addr;
  proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
  proxy_set_header X-Forwarded-Proto $scheme;
}
```

## 防火墙与降级语义

V2 的 `nftables` 管理器只操作名为 `zhiyu_waf` 的独立表，并在表内维护 IPv4 与 IPv6 封禁集合。它不调用 `iptables`、`ip6tables` 或任意防火墙 shell 命令。进程需要 `CAP_NET_ADMIN`（或宿主机等效权限）才能写入内核集合。

如果缺少权限、内核不支持或 Netlink 操作失败，系统会在 Dashboard 报告 **degraded**。此时应用层的风险判定与 HTTP 阻断仍会继续执行，但内核级 IP 封禁不可用。生产上线前应把该状态纳入监控与告警。

| 风险分数 | 默认封禁时长 |
| --- | --- |
| 85–91 | 10 分钟 |
| 92–97 | 1 小时 |
| 98–100 | 24 小时 |

重启时，SQLite 中尚未到期的临时封禁会按**剩余**时间回放，不会因为重启而延长 TTL。

## 配置要点

默认配置位于 [configs/zhiyu-waf.yaml](configs/zhiyu-waf.yaml)。

| 配置项 | 说明 |
| --- | --- |
| `proxy.listen_addr` | WAF 监听地址 |
| `proxy.backend_addr` | 默认回源服务地址 |
| `proxy.trusted_proxies` | 唯一可影响转发 IP 头的直接代理 CIDR/IP |
| `dashboard.listen_addr` | 管理后台监听地址 |
| `dashboard.jwt_secret` | 管理会话签名密钥；生产应使用密钥管理系统提供 |
| `engine.rules_dir` | 本地 YAML 规则目录 |
| `storage.path` | SQLite 数据文件路径 |
| `ai.enabled` | 是否启用可选 AI 辅助；默认 `false` |

## V2 REST API

受 JWT 保护的 V2 API 采用统一封装：成功时返回 `data` 与 `request_id`，失败时返回 `error.code`、`error.message` 与 `request_id`。

| 资源 | 端点 |
| --- | --- |
| Dashboard 汇总 | `GET /api/v1/dashboard/summary?range=24h` |
| Dashboard 趋势 | `GET /api/v1/dashboard/timeseries?range=24h` |
| 事件列表与详情 | `GET /api/v1/events`、`GET /api/v1/events/{id}` |
| 站点 | `GET/POST /api/v1/sites`、`PUT/DELETE /api/v1/sites/{id}` |
| 封禁 IP | `GET/POST /api/v1/blocked-ips`、`DELETE /api/v1/blocked-ips/{id}` |
| 系统状态 | `GET /api/v1/system/status` |

## 数据与隐私

安全事件只保存定位、审计与关联所需的字段，例如来源 IP、Host、路径、规则标识、风险分数与动作。完整请求体不写入事件表。接入 AI 前，请独立评估数据脱敏、出网审批、供应商条款和业务数据合规要求。

## 生产上线清单

在将流量切换到 V2 前，应首先在 `monitor` 模式验证规则命中，再逐站启用 `protect`。请完成回源连通性、上传/流式请求兼容性、误报处理、IPv4/IPv6 封禁、`nftables` 降级告警、SQLite 备份与恢复、管理端访问隔离等验证。WAF 是纵深防御的一层，不能替代应用认证授权、输入校验、依赖治理、漏洞修复和安全监控。

## 开发与测试

```bash
cd web && npm ci && npm run build:raw && cd ..
rm -rf internal/dashboard/dist && cp -R web/dist internal/dashboard/dist
CGO_ENABLED=1 go test ./...
```

## 许可证

本项目使用 [AGPL-3.0](LICENSE)。请在将其用于网络服务、交付或再分发前，评估相应的开源义务。
