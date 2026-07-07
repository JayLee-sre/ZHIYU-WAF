<template>
  <div class="welcome-page">
    <main class="welcome-shell">
      <section class="hero-panel">
        <div class="brand-row">
          <img src="/logo.png" alt="智域 WAF" />
          <span>智域 WAF</span>
        </div>
        <div class="success-badge">
          <span class="success-dot"></span>
          初始化已完成
        </div>
        <h1>控制台已经准备好</h1>
        <p>
          WAF 已完成基础配置。接下来可以进入控制台查看安全态势，
          再逐步完善规则、证书、站点和高级防护能力。
        </p>
        <div class="hero-actions">
          <button class="primary-btn" @click="enterDashboard">进入控制台</button>
          <button class="ghost-btn" @click="openSettings">查看系统设置</button>
        </div>
      </section>

      <section class="checklist-panel">
        <div class="panel-head">
          <span>启动清单</span>
          <strong>{{ today }}</strong>
        </div>
        <div class="check-row" v-for="item in checks" :key="item.title">
          <span class="check-icon">✓</span>
          <div>
            <strong>{{ item.title }}</strong>
            <p>{{ item.desc }}</p>
          </div>
        </div>
      </section>

      <section class="next-grid">
        <button class="next-card" v-for="item in nextActions" :key="item.path" @click="go(item.path)">
          <span class="next-kicker">{{ item.kicker }}</span>
          <strong>{{ item.title }}</strong>
          <p>{{ item.desc }}</p>
        </button>
      </section>
    </main>
  </div>
</template>

<script setup>
import { useRouter } from 'vue-router'

const router = useRouter()
const today = new Date().toLocaleDateString('zh-CN', { year: 'numeric', month: 'long', day: 'numeric' })

const checks = [
  { title: '管理员账号已创建', desc: '默认用户名为 admin，请妥善保存刚刚设置的密码。' },
  { title: '代理监听已配置', desc: 'WAF 会按你的配置监听端口，并转发到后端业务服务。' },
  { title: '基础规则已加载', desc: 'SQL 注入、XSS、命令执行、路径穿越等基础检测规则已就绪。' },
]

const nextActions = [
  { path: '/dashboard', kicker: '第一步', title: '查看安全态势', desc: '确认请求、拦截、规则数量和系统状态是否正常。' },
  { path: '/rules', kicker: '策略', title: '检查规则引擎', desc: '按业务风险调整规则启停和危险等级。' },
  { path: '/certs', kicker: '上线', title: '准备证书配置', desc: '正式接入 HTTPS 前，先检查证书和域名配置。' },
]

function enterDashboard() {
  router.replace('/dashboard')
}

function openSettings() {
  router.replace('/settings')
}

function go(path) {
  router.replace(path)
}
</script>

<style scoped>
.welcome-page {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 32px 18px;
  background: #f4f6fb;
}

.welcome-shell {
  width: min(980px, 100%);
  display: grid;
  grid-template-columns: minmax(0, 1.15fr) minmax(320px, .85fr);
  gap: 18px;
}

.hero-panel,
.checklist-panel,
.next-card {
  background: #fff;
  border: 1px solid #e2e8f0;
  border-radius: 14px;
  box-shadow: 0 10px 30px rgba(15, 23, 42, .05);
}

.hero-panel {
  padding: 38px;
}

.brand-row {
  display: flex;
  align-items: center;
  gap: 10px;
  color: #0f172a;
  font-size: 16px;
  font-weight: 800;
}
.brand-row img {
  width: 36px;
  height: 36px;
  border-radius: 9px;
}

.success-badge {
  width: fit-content;
  display: inline-flex;
  align-items: center;
  gap: 8px;
  margin-top: 42px;
  padding: 6px 12px;
  border-radius: 999px;
  background: #ecfdf5;
  color: #047857;
  font-size: 13px;
  font-weight: 800;
}
.success-dot {
  width: 7px;
  height: 7px;
  border-radius: 50%;
  background: #10b981;
  box-shadow: 0 0 0 4px rgba(16,185,129,.12);
}

.hero-panel h1 {
  margin: 18px 0 12px;
  color: #0f172a;
  font-size: 34px;
  line-height: 1.15;
  font-weight: 900;
  letter-spacing: 0;
}
.hero-panel p {
  max-width: 560px;
  color: #475569;
  font-size: 15px;
  line-height: 1.8;
}

.hero-actions {
  display: flex;
  gap: 12px;
  margin-top: 30px;
}
.primary-btn,
.ghost-btn {
  height: 40px;
  padding: 0 18px;
  border-radius: 8px;
  font-size: 14px;
  font-weight: 800;
  cursor: pointer;
}
.primary-btn {
  border: 0;
  background: #4f46e5;
  color: #fff;
}
.ghost-btn {
  border: 1px solid #cbd5e1;
  background: #fff;
  color: #334155;
}

.checklist-panel {
  padding: 22px;
}
.panel-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding-bottom: 16px;
  border-bottom: 1px solid #f1f5f9;
}
.panel-head span {
  color: #0f172a;
  font-size: 16px;
  font-weight: 900;
}
.panel-head strong {
  color: #94a3b8;
  font-size: 12px;
}

.check-row {
  display: grid;
  grid-template-columns: 28px 1fr;
  gap: 12px;
  padding: 17px 0;
  border-bottom: 1px solid #f8fafc;
}
.check-row:last-child {
  border-bottom: 0;
}
.check-icon {
  width: 28px;
  height: 28px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border-radius: 8px;
  background: #eef2ff;
  color: #4f46e5;
  font-weight: 900;
}
.check-row strong {
  display: block;
  color: #0f172a;
  font-size: 14px;
  margin-bottom: 3px;
}
.check-row p {
  color: #64748b;
  font-size: 13px;
  line-height: 1.55;
}

.next-grid {
  grid-column: 1 / -1;
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 12px;
}
.next-card {
  min-height: 132px;
  padding: 18px;
  text-align: left;
  cursor: pointer;
}
.next-card:hover {
  border-color: #a5b4fc;
  box-shadow: 0 12px 30px rgba(79, 70, 229, .08);
}
.next-kicker {
  display: block;
  color: #6366f1;
  font-size: 12px;
  font-weight: 900;
  margin-bottom: 10px;
}
.next-card strong {
  display: block;
  color: #0f172a;
  font-size: 16px;
  margin-bottom: 7px;
}
.next-card p {
  color: #64748b;
  font-size: 13px;
  line-height: 1.55;
}

@media (max-width: 820px) {
  .welcome-shell,
  .next-grid {
    grid-template-columns: 1fr;
  }
  .hero-panel {
    padding: 26px;
  }
  .hero-panel h1 {
    font-size: 28px;
  }
  .hero-actions {
    flex-direction: column;
  }
}
</style>
