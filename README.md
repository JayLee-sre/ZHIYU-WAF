<div align="center">

<img src="https://cdn.jsdelivr.net/gh/JayLee-sre/ZHIYU-WAF@ef31412/images/preview/logo.webp" alt="智域 WAF" width="104">

# 智域 WAF

面向中小企业、独立开发者和私有化场景的轻量级 Web 应用防火墙。

把规则防护、访问控制、SSH 暴力破解防护、AI 辅助分析和可视化管理放进一个可自部署的安全网关里。

<br>

![Go](https://img.shields.io/badge/Go-1.25-00ADD8?logo=go&logoColor=white)
![Vue](https://img.shields.io/badge/Vue-3-42b883?logo=vue.js&logoColor=white)
![Docker](https://img.shields.io/badge/Docker-ready-2496ED?logo=docker&logoColor=white)
![License](https://img.shields.io/badge/License-AGPL--3.0-green)
![Release](https://img.shields.io/github/v/release/JayLee-sre/ZHIYU-WAF)
![WeChat](https://img.shields.io/badge/WeChat-cc8c887-07C160?logo=wechat&logoColor=white)

[快速开始](#快速开始) · [功能能力](#功能能力) · [部署方式](#部署方式) · [版本对比](#版本对比) · [联系开发者](#联系开发者) · [Release](https://github.com/JayLee-sre/ZHIYU-WAF/releases)

</div>

---

## 项目定位

智域 WAF 不是一个只展示日志的面板，而是一个可以真正接入业务流量的防护代理。

它默认通过 `iptables REDIRECT` 接管公网入口流量，将请求转发到 WAF 代理端口，再由规则引擎、速率限制、访问控制、地理封锁、威胁情报和 AI 分析共同判断，最后回源到真实业务服务。

适合这些场景：

- 中小型网站、后台系统、API 服务的边界防护
- 没有完整安全团队，但需要可视化攻击日志和快速封禁能力的业务
- 希望私有化部署，不把访问日志交给第三方云 WAF 的团队
- 需要社区版起步，后续平滑升级专业版能力的产品化场景

## 功能能力

| 能力 | 说明 |
| --- | --- |
| 规则引擎 | 内置 SQL 注入、XSS、命令注入、路径穿越、敏感文件探测等常见 Web 攻击规则 |
| 速率限制 | 按请求频率和突发流量控制异常访问，降低扫描器和爆破流量影响 |
| IP 黑白名单 | 管理后台实时维护访问控制列表，防护链路立即生效 |
| 攻击日志 | 记录命中规则、来源 IP、请求路径、严重等级和处理动作 |
| SSH 防护 | 监控 SSH 登录失败日志，自动封禁暴力破解来源 |
| SSL/TLS | 支持自定义证书与 ACME 自动证书能力 |
| AI 辅助分析 | 可接入 OpenAI 兼容接口，对高风险请求进行语义分析和研判 |
| 威胁情报 | 同步恶意 IP 情报源，并联动黑名单防护 |
| 地理封锁 | 按国家或地区封禁访问来源 |
| 多站点管理 | 支持多站点回源配置，适合一套 WAF 管理多个业务 |
| 审计日志 | 记录关键管理操作，便于追踪配置变更 |
| 可视化大屏 | 展示安全态势、攻击趋势和防护效果 |

## 界面预览

<table>
<tr>
<td width="50%"><img src="https://cdn.jsdelivr.net/gh/JayLee-sre/ZHIYU-WAF@ef31412/images/preview/login.webp" alt="登录页面"></td>
<td width="50%"><img src="https://cdn.jsdelivr.net/gh/JayLee-sre/ZHIYU-WAF@ef31412/images/preview/attack-logs.webp" alt="攻击日志"></td>
</tr>
<tr>
<td width="50%"><img src="https://cdn.jsdelivr.net/gh/JayLee-sre/ZHIYU-WAF@ef31412/images/preview/rules-engine.webp" alt="规则引擎"></td>
<td width="50%"><img src="https://cdn.jsdelivr.net/gh/JayLee-sre/ZHIYU-WAF@ef31412/images/preview/ssh-monitor.webp" alt="SSH 监控"></td>
</tr>
<tr>
<td width="50%"><img src="https://cdn.jsdelivr.net/gh/JayLee-sre/ZHIYU-WAF@ef31412/images/preview/threat-intel.webp" alt="威胁情报"></td>
<td width="50%"><img src="https://cdn.jsdelivr.net/gh/JayLee-sre/ZHIYU-WAF@ef31412/images/preview/access-control.webp" alt="访问控制"></td>
</tr>
</table>

<img src="https://cdn.jsdelivr.net/gh/JayLee-sre/ZHIYU-WAF@ef31412/images/preview/settings.webp" alt="系统设置" width="100%">

## 快速开始

### Docker Compose

```bash
git clone https://github.com/JayLee-sre/ZHIYU-WAF.git
cd ZHIYU-WAF

docker compose up -d
```

默认入口：

| 服务 | 地址 |
| --- | --- |
| 管理后台 | `http://127.0.0.1:9090` |
| WAF 代理 | `http://127.0.0.1:8080` |
| 公网入口 | 默认由 `iptables` 将 `80` 转发到 WAF |

首次启动会在日志中输出管理员初始密码：

```bash
docker logs zhiyu-waf
```

### 一键安装脚本

适合 Linux 服务器直接部署到 `/opt/zhiyu-waf`：

```bash
sudo bash scripts/install-zhiyu-waf.sh \
  --backend 127.0.0.1:3000 \
  --public-port 80 \
  --dashboard-port 9090
```

常用参数：

| 参数 | 说明 |
| --- | --- |
| `--backend` | 真实业务服务地址，例如 `127.0.0.1:3000` |
| `--public-port` | 对外暴露的业务端口，通常是 `80` |
| `--waf-port` | WAF 代理监听端口，默认 `8080` |
| `--dashboard-port` | 管理后台端口，默认 `9090` |
| `--no-iptables` | 不接管公网端口，只启动 WAF 代理 |
| `--no-firewall` | 不自动修改 firewalld/防火墙规则 |

### 源码构建

```bash
git clone https://github.com/JayLee-sre/ZHIYU-WAF.git
cd ZHIYU-WAF

make build
sudo ./bin/zhiyu-waf -config configs/zhiyu-waf.yaml
```

也可以分开构建：

```bash
cd web && npm install && npm run build && cd ..
CGO_ENABLED=1 go build -o bin/zhiyu-waf ./cmd/zhiyu-waf
```

## 部署方式

### 透明代理模式

推荐生产部署使用该模式。业务访问 `80`，系统将流量转发到 WAF，再由 WAF 回源到真实服务。

```text
Client
  |
  | :80
  v
iptables REDIRECT
  |
  | :8080
  v
ZhiYu-WAF
  |
  | backend_addr
  v
Your Backend
```

关键配置：

```yaml
proxy:
  listen_addr: ":8080"
  backend_addr: "127.0.0.1:3000"
  iptables_enable: true
  iptables_port: 80
```

### 反向代理模式

如果你已经有 Nginx、SLB 或云负载均衡，也可以关闭 `iptables`，让上游代理转发到 WAF。

```yaml
proxy:
  listen_addr: ":8080"
  backend_addr: "127.0.0.1:3000"
  iptables_enable: false
```

### Docker 生产建议

透明代理需要 host 网络和 `NET_ADMIN` 能力：

```bash
docker build -t zhiyu-waf:latest .

docker run -d \
  --name zhiyu-waf \
  --restart unless-stopped \
  --network host \
  --cap-add NET_ADMIN \
  -v "$(pwd)/configs:/opt/zhiyu-waf/configs:ro" \
  -v zhiyu-waf-data:/opt/zhiyu-waf/data \
  zhiyu-waf:latest
```

如果不使用透明代理，请关闭 `iptables_enable`，再按你的网络架构映射端口。

## 配置说明

主配置文件位于 [configs/zhiyu-waf.yaml](configs/zhiyu-waf.yaml)。

| 配置项 | 说明 |
| --- | --- |
| `proxy.listen_addr` | WAF 代理监听地址 |
| `proxy.backend_addr` | 真实业务回源地址 |
| `proxy.iptables_enable` | 是否启用本机端口接管 |
| `dashboard.listen_addr` | 管理后台监听地址 |
| `dashboard.jwt_secret` | 后台登录令牌密钥，生产环境必须修改 |
| `dashboard.cors_origins` | 管理后台允许的访问来源 |
| `engine.rules_dir` | 检测规则目录 |
| `storage.path` | SQLite 数据文件路径 |
| `ai.enabled` | 是否启用 AI 辅助分析 |

生产环境建议：

- 修改 `dashboard.jwt_secret`
- 将 `dashboard.cors_origins` 改成实际后台域名
- 限制 `9090` 管理端口的公网访问
- 为 `data/` 做定期备份
- 启用 HTTPS 或通过上游网关终止 TLS
- 先在观察模式或测试流量中验证规则，再接入核心业务

## 版本对比

| 功能 | 社区版 | 专业版 |
| --- | :---: | :---: |
| 规则引擎 | 支持 | 支持 |
| 速率限制 | 支持 | 支持 |
| 攻击日志 | 支持 | 支持 |
| IP 黑白名单 | 支持 | 支持 |
| SSH 暴力破解防护 | 支持 | 支持 |
| SSL/TLS 管理 | 支持 | 支持 |
| 审计日志 | 支持 | 支持 |
| AI 辅助分析 | 基础能力 | 高级能力 |
| 地理封锁 | - | 支持 |
| 威胁情报同步 | - | 支持 |
| 多站点管理 | - | 支持 |
| 多用户与 RBAC | - | 支持 |
| 安全态势大屏 | - | 支持 |
| 商业支持 | - | 支持 |

专业版能力用于需要团队协作、多站管理、情报联动和更完整安全运营闭环的场景。具体开通方式请在管理后台的系统设置中查看。

## 联系开发者

欢迎交流部署、商业授权、私有化交付和功能定制。

![WeChat](https://img.shields.io/badge/微信-cc8c887-07C160?logo=wechat&logoColor=white)

## 规则目录

默认规则位于 [configs/rules](configs/rules)：

| 文件 | 说明 |
| --- | --- |
| `sqli.yaml` | SQL 注入检测 |
| `xss.yaml` | XSS 检测 |
| `cmdi.yaml` | 命令注入检测 |
| `traversal.yaml` | 路径穿越检测 |
| `sensitive.yaml` | 敏感文件与路径检测 |
| `enterprise.yaml` | 企业级增强规则 |

修改规则后，可通过管理后台或重启服务使配置生效。

## 开发

后端：

```bash
go test ./...
go run ./cmd/zhiyu-waf -config configs/zhiyu-waf.yaml
```

前端：

```bash
cd web
npm install
npm run dev
```

完整构建：

```bash
make build
```

## Release

最新版本请查看 [GitHub Releases](https://github.com/JayLee-sre/ZHIYU-WAF/releases)。

当前正式版本：

- [v1.0.0](https://github.com/JayLee-sre/ZHIYU-WAF/releases/tag/v1.0.0)

## 许可证

本项目采用 [AGPL-3.0](LICENSE) 协议发布。

如果你计划在商业产品、SaaS 服务、私有化交付或闭源环境中使用，请先确认 AGPL-3.0 的网络服务开源义务，并根据实际场景选择合适的授权方式。

## 免责声明

WAF 是防御体系的一部分，不应替代应用自身的安全开发、依赖治理、身份认证、权限控制和漏洞修复。

在生产环境接入前，请务必完成规则验证、回源连通性检查、误报评估和回滚预案。
