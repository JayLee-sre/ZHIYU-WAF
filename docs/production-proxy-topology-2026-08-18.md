# 生产代理与 NPS 拓扑审计基线

审计时间：2026-08-18（GMT+8）

## 运行中的相关服务

| 服务 | 状态 | 作用 |
| --- | --- | --- |
| `nginx.service` | active (running) | 统一处理公网 80/443 HTTPS 入口与现有站点反向代理。 |
| `Nps.service` | active (running) | 内网穿透服务，提供 Web 管理和 TCP 隧道端口。 |
| `zhiyu-waf.service` | active (running) | ZHIYU-WAF V2 控制面与应用层代理。 |

## 已发现监听端口

| 端口 | 监听范围 | 进程 | 说明 |
| --- | --- | --- | --- |
| 80、443 | 公网 IPv4/IPv6 | Nginx | 现有业务 HTTPS 入口，当前不可直接改写。 |
| 13000 | 公网 | NPS | NPS 客户端桥接端口。 |
| 13001、13002 | 127.0.0.1 | NPS | NPS 反向代理目标端口。 |
| 8081 | 公网 | NPS | NPS Web 管理服务；Nginx 以 HTTPS 反代至此端口。 |
| 8025 | 公网 | NPS | NPS 额外服务端口。 |
| 19090 | 公网 | ZHIYU-WAF | 当前控制台入口。 |
| 8080 | 公网 | ZHIYU-WAF | 当前应用层代理监听。 |
| 9081 | 127.0.0.1 | Python | `istoreos` 前置认证服务。 |

## Nginx 站点映射

| 域名 | 当前 Nginx 回源 | 备注 |
| --- | --- | --- |
| `blog.siboai.cc` | `http://127.0.0.1:13002` | 由 NPS 提供的本地回源。 |
| `grafana.siboai.cc` | `http://127.0.0.1:13000` | 由 NPS 提供的本地回源。 |
| `istoreos.siboai.cc` | 先经 `http://127.0.0.1:9081/auth`，后回源 `http://127.0.0.1:13001` | 带 Python 认证前置逻辑，不能直接替换为 WAF。 |
| `nps.siboai.cc` | `https://127.0.0.1:8081` | NPS 管理面，必须避免置于自身 NPS 隧道策略之前。 |

## NPS 服务要点

`Nps.service` 使用 `/usr/bin/nps service` 启动，配置文件为 `/etc/nps/conf/nps.conf`。已确认 Web 管理端口为 8081；桥接和鉴权值未在本审计记录中保存。NPS 服务本身不应被直接改写、重启或置于 WAF 前面，除非先完成独立回滚与连通性验证。

## 安全导入原则

第一阶段仅将 `blog.siboai.cc` 与 `grafana.siboai.cc` 导入 ZHIYU-WAF 的站点数据模型，初始状态使用观察模式，并保留 Nginx 现有回源不变。`istoreos.siboai.cc` 因包含认证前置，`nps.siboai.cc` 因属于隧道管理面，均不在未确认的情况下改写流量路径。

下一阶段需要核对 ZHIYU-WAF 的按 Host 路由能力、健康探测结果与 Nginx 可回滚配置后，才可考虑通过本机环回端口将指定业务域名接入 WAF 代理。

## 已执行的受控导入

2026-08-18 已将以下两个域名写入 ZHIYU-WAF 的站点配置，并设为 `monitor` 观察模式且启用。此操作只加载了 WAF 的按 Host 路由表；**未修改 Nginx 配置、未重载 Nginx、未改变 80/443 的现有公网流量路径**。

| WAF 站点 | WAF 回源 | 模式 | 本机 WAF 回源验证 | 现有 Nginx HTTPS 验证 |
| --- | --- | --- | --- | --- |
| `blog.siboai.cc` | `http://127.0.0.1:13002` | `monitor` | HTTP 200 | HTTP 200 |
| `grafana.siboai.cc` | `http://127.0.0.1:13000` | `monitor` | HTTP 302 | HTTP 302 |

导入前的 SQLite 回滚备份保存在：`/opt/zhiyu-waf/data/zhiyu-waf.db.pre-waf-site-import-20260818T142826+0800`。服务重启后控制台健康检查为 `status: ok`；`nginx -t` 也已通过。

`istoreos.siboai.cc` 和 `nps.siboai.cc` 仍保持原始路径，未纳入 WAF 流量切换。后续若要把业务公网入口真正切到 WAF，应当只对已验证的业务域名采用“先将对应 Nginx `proxy_pass` 改为本机 WAF 监听端口、再观察监控模式”的逐站点灰度方式，并在切换前备份对应 Nginx 虚拟主机文件。

## 已完成的观察模式灰度切流

在确认本机 WAF 回源与原 HTTPS 响应一致后，已将以下两个 Nginx 业务虚拟主机的 `proxy_pass` 切换到 `http://127.0.0.1:8080`。WAF 根据 Host 再回源至对应 NPS 端口，站点仍处于 `monitor` 模式，因此当前阶段只记录风险而不主动阻断。

| 公网入口 | 当前链路 | 切流后验证 |
| --- | --- | --- |
| `https://blog.siboai.cc` | Nginx → WAF:8080 → NPS:13002 | HTTP 200 |
| `https://grafana.siboai.cc` | Nginx → WAF:8080 → NPS:13000 | HTTP 302（保持原登录跳转） |

Nginx 配置回滚副本为：`/etc/nginx/.zhiyu-waf-observe-20260818T143429+0800`。`nginx.service`、`Nps.service`、`zhiyu-waf.service` 均处于 `active` 状态；NPS 的 13000、13001、13002、8025、8081 监听端口仍正常保留。

暂不切换 `istoreos.siboai.cc`（保留 Python 认证前置）和 `nps.siboai.cc`（NPS 管理面），以避免认证链或隧道管理面受到不必要影响。

## 观察模式误封修复

在首次将业务流量切入观察模式后，发现正常访问会被写入 ZHIYU 的本地封禁表。排查结果表明，问题并非业务规则本身，而是代理处理器在判断站点为 `monitor` 前，先把原始 `block` / `rate_limit` 决策传给了持久化回调；该回调会负责 nftables 升级，因此观察事件被错误地变成了内核封禁。

已完成以下处置：

| 项目 | 结果 |
| --- | --- |
| 紧急恢复 | 已先恢复两个 Nginx 站点的原始 NPS 回源，并清空 `blocked_ips` 与 `inet zhiyu` 专属封禁集合。 |
| 代码修复 | 观察模式下，`block` 与 `rate_limit` 决策在传入持久化回调前会被复制并降级为 `log`；原始决策不被修改，保护模式仍保留强制处置。 |
| 回归测试 | 新增 Block / RateLimit 降级测试；`go test -race ./...` 通过。 |
| 再次接入 | 修复二进制已部署后，`blog.siboai.cc` 和 `grafana.siboai.cc` 已重新经由 WAF 观察模式转发。 |
| 生产验证 | 两站点分别返回 HTTP 200 与 HTTP 302；`blocked_ips` 计数为 0；Nginx、NPS、ZHIYU-WAF 均为 `active`。 |

最新一次 Nginx 回滚备份位于：`/etc/nginx/.zhiyu-waf-observe-fixed-20260818T145017+0800`。在观察模式稳定运行前，不应将这两个站点切换至 `protect`。
