<template>
  <div class="setup-page">
    <div class="setup-card">
      <div class="setup-header">
        <div class="setup-brand">
          <img src="/logo.png" alt="智域 WAF" class="setup-logo" />
          <span>智域 WAF</span>
        </div>
        <h1>首次启动配置</h1>
        <p class="setup-subtitle">先完成最小可用配置，进入控制台后还可以继续调整高级策略。</p>
      </div>

      <el-steps :active="step" finish-status="success" class="steps-bar" align-center>
        <el-step title="管理员" />
        <el-step title="接入业务" />
        <el-step title="安全增强" />
        <el-step title="确认启动" />
      </el-steps>

      <!-- Step 0: Password -->
      <div class="step-body" v-if="step === 0">
        <div class="step-kicker">账号安全</div>
        <h2>设置控制台管理员密码</h2>
        <p class="step-desc">控制台默认管理员用户名为 <strong>admin</strong>。密码至少 12 位，建议包含大小写字母、数字和符号。</p>
        <div class="form-group">
          <label>新密码</label>
          <el-input v-model="form.password" type="password" show-password placeholder="请输入管理员密码" size="large" />
        </div>
        <div class="form-group">
          <label>确认密码</label>
          <el-input v-model="form.confirmPassword" type="password" show-password placeholder="请再次输入密码" size="large" />
        </div>
        <div class="password-rules">
          <span :class="{ ok: form.password.length >= 12 }">至少 12 位</span>
          <span :class="{ ok: form.password && form.password === form.confirmPassword }">两次输入一致</span>
        </div>
      </div>

      <!-- Step 1: Proxy -->
      <div class="step-body" v-if="step === 1">
        <div class="step-kicker">业务接入</div>
        <h2>告诉 WAF 你的业务服务在哪里</h2>
        <p class="step-desc">建议先使用本机端口验证链路，确认无误后再切换到正式回源和端口转发。</p>
        <div class="form-group">
          <label>后端地址 <span class="required">*</span></label>
          <el-input v-model="form.backendAddr" placeholder="例如: 127.0.0.1:8080" size="large" />
          <span class="field-hint">您的 Web 应用实际监听的地址和端口</span>
        </div>
        <div class="form-row">
          <div class="form-group">
            <label>WAF 监听端口</label>
            <el-input-number v-model="form.listenPort" :min="1" :max="65535" size="large" style="width: 100%" />
            <span class="field-hint">WAF 代理监听的端口，默认 8080</span>
          </div>
          <div class="form-group">
            <label>iptables 端口</label>
            <el-input-number v-model="form.iptablesPort" :min="1" :max="65535" size="large" style="width: 100%" />
            <span class="field-hint">将此端口流量重定向到 WAF，默认 80</span>
          </div>
        </div>
        <div class="option-card" :class="{ active: form.iptablesEnable }">
          <div>
            <strong>自动接管 80 端口流量</strong>
            <span>需要 root 权限和 iptables，建议生产服务器确认端口占用后再开启。</span>
          </div>
          <el-switch v-model="form.iptablesEnable" />
        </div>
      </div>

      <!-- Step 2: Enhancements -->
      <div class="step-body" v-if="step === 2">
        <div class="step-kicker">可选能力</div>
        <h2>选择要立即启用的增强能力</h2>
        <p class="step-desc">这些能力都可以稍后在控制台开启。首次启动建议保持简洁，先保证主链路跑通。</p>

        <div class="option-list">
          <div class="option-card" :class="{ active: form.aiEnabled }">
            <div>
              <strong>AI 智能检测</strong>
              <span>需要 OpenAI 兼容 API Key。未配置密钥时请保持关闭。</span>
            </div>
            <el-switch v-model="form.aiEnabled" />
          </div>
          <div class="option-card" :class="{ active: form.sshEnabled }">
            <div>
              <strong>SSH 暴力破解防护</strong>
              <span>依赖系统 auth 日志。云服务器路径不同可能需要后续调整。</span>
            </div>
            <el-switch v-model="form.sshEnabled" />
          </div>
        </div>

        <template v-if="form.aiEnabled">
          <div class="form-group">
            <label>API Key <span class="required">*</span></label>
            <el-input v-model="form.apiKey" placeholder="sk-xxxxxxxx" size="large" show-password />
            <span class="field-hint">兼容 OpenAI API 格式的密钥</span>
          </div>
          <div class="form-row">
            <div class="form-group">
              <label>模型名称</label>
              <el-input v-model="form.aiModel" placeholder="gpt-4o / deepseek-chat" size="large" />
            </div>
            <div class="form-group">
              <label>API Base URL</label>
              <el-input v-model="form.aiBaseURL" placeholder="https://api.openai.com/v1" size="large" />
            </div>
          </div>
        </template>

        <div class="section-divider"></div>
        <div class="form-row">
          <div class="form-group">
            <label>每分钟请求限制</label>
            <el-input-number v-model="form.rpm" :min="1" :max="10000" size="large" style="width: 100%" />
            <span class="field-hint">单 IP 每分钟最大请求数</span>
          </div>
          <div class="form-group">
            <label>突发容量</label>
            <el-input-number v-model="form.burstSize" :min="1" :max="1000" size="large" style="width: 100%" />
            <span class="field-hint">允许的瞬时突发请求数</span>
          </div>
        </div>

        <template v-if="form.sshEnabled">
          <div class="form-row">
            <div class="form-group">
              <label>最大失败次数</label>
              <el-input-number v-model="form.sshMaxFails" :min="1" :max="20" size="large" style="width: 100%" />
            </div>
            <div class="form-group">
              <label>封禁时长（分钟）</label>
              <el-input-number v-model="form.sshBanMinutes" :min="1" :max="1440" size="large" style="width: 100%" />
            </div>
          </div>
        </template>
      </div>

      <!-- Step 3: Summary -->
      <div class="step-body" v-if="step === 3">
        <div class="step-kicker">启动前确认</div>
        <h2>确认配置并进入控制台</h2>
        <p class="step-desc">启动后会自动登录管理员账号，并展示欢迎页。</p>
        <div class="summary-grid">
          <div class="summary-item">
            <span class="summary-label">管理员账号</span>
            <strong>admin</strong>
          </div>
          <div class="summary-item">
            <span class="summary-label">代理后端</span>
            <strong>{{ form.backendAddr || '未设置' }}</strong>
          </div>
          <div class="summary-item">
            <span class="summary-label">监听端口</span>
            <strong>:{{ form.listenPort }}</strong>
          </div>
          <div class="summary-item">
            <span class="summary-label">AI 引擎</span>
            <strong :class="form.aiEnabled ? 'text-green' : 'text-muted'">{{ form.aiEnabled ? '已启用' : '未启用' }}</strong>
          </div>
          <div class="summary-item" v-if="form.aiEnabled">
            <span class="summary-label">AI 模型</span>
            <strong>{{ form.aiModel || '默认' }}</strong>
          </div>
          <div class="summary-item">
            <span class="summary-label">速率限制</span>
            <strong>{{ form.rpm }} req/min</strong>
          </div>
          <div class="summary-item">
            <span class="summary-label">SSH 防护</span>
            <strong :class="form.sshEnabled ? 'text-green' : 'text-muted'">{{ form.sshEnabled ? '已启用' : '未启用' }}</strong>
          </div>
          <div class="summary-item">
            <span class="summary-label">iptables</span>
            <strong :class="form.iptablesEnable ? 'text-green' : 'text-muted'">{{ form.iptablesEnable ? '已启用' : '未启用' }}</strong>
          </div>
        </div>
      </div>

      <!-- Navigation -->
      <div class="step-nav">
        <el-button v-if="step > 0" @click="step--" size="large">上一步</el-button>
        <div class="nav-spacer"></div>
        <el-button v-if="step < 3" type="primary" @click="nextStep" size="large" :disabled="!canNext">
          下一步
        </el-button>
        <el-button v-if="step === 3" type="primary" @click="applySetup" size="large" :loading="applying">
          立即启动
        </el-button>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed } from 'vue'
import { useRouter } from 'vue-router'
import { ElInputNumber, ElMessage, ElStep, ElSteps, ElSwitch } from 'element-plus'
import api, { setAuthToken } from '../api'
import { markSetupComplete } from '../router'

const router = useRouter()
const step = ref(0)
const applying = ref(false)

const form = ref({
  password: '',
  confirmPassword: '',
  backendAddr: '127.0.0.1:80',
  listenPort: 8080,
  iptablesEnable: false,
  iptablesPort: 80,
  aiEnabled: false,
  apiKey: '',
  aiModel: '',
  aiBaseURL: '',
  rpm: 60,
  burstSize: 10,
  sshEnabled: false,
  sshMaxFails: 5,
  sshBanMinutes: 30,
})

const canNext = computed(() => {
  if (step.value === 0) {
    return form.value.password.length >= 12 && form.value.password === form.value.confirmPassword
  }
  if (step.value === 1) {
    return !!form.value.backendAddr
  }
  if (step.value === 2 && form.value.aiEnabled) {
    return !!form.value.apiKey.trim()
  }
  return true
})

async function nextStep() {
  if (step.value === 0) {
    try {
      await api.post('/setup/password', { password: form.value.password })
    } catch (e) {
      ElMessage.error('密码设置失败')
      return
    }
  }
  step.value++
}

async function applySetup() {
  applying.value = true
  try {
    const f = form.value
    const res = await api.post('/setup/apply', {
      password: f.password,
      backend_addr: f.backendAddr,
      listen_port: f.listenPort,
      iptables_enable: f.iptablesEnable,
      iptables_port: f.iptablesPort,
      ai_enabled: f.aiEnabled,
      api_key: f.apiKey,
      ai_model: f.aiModel,
      ai_base_url: f.aiBaseURL,
      rpm: f.rpm,
      burst_size: f.burstSize,
      ssh_enabled: f.sshEnabled,
      ssh_max_fails: f.sshMaxFails,
      ssh_ban_minutes: f.sshBanMinutes,
    })
    if (res.token) setAuthToken(res.token)
    markSetupComplete()
    ElMessage.success('配置完成，正在跳转...')
    setTimeout(() => router.replace('/welcome'), 600)
  } catch (e) {
    ElMessage.error(e.response?.data?.error || '配置应用失败')
  } finally {
    applying.value = false
  }
}
</script>

<style scoped>
.setup-page {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  background: #f4f6fb;
  padding: 20px;
}

.setup-card {
  background: #fff;
  border: 1px solid #e2e8f0;
  border-radius: 14px;
  box-shadow: 0 12px 34px rgba(15, 23, 42, 0.06);
  padding: 34px;
  width: 100%;
  max-width: 760px;
}

.setup-header {
  margin-bottom: 28px;
}

.setup-brand {
  display: inline-flex;
  align-items: center;
  gap: 10px;
  color: #0f172a;
  font-size: 15px;
  font-weight: 900;
  margin-bottom: 18px;
}

.setup-logo {
  width: 34px;
  height: 34px;
  border-radius: 9px;
  box-shadow: 0 2px 12px rgba(99, 102, 241, 0.15);
}

.setup-header h1 {
  font-size: 28px;
  font-weight: 900;
  color: #0f172a;
  margin-bottom: 8px;
  letter-spacing: 0;
}

.setup-subtitle {
  font-size: 14px;
  color: #94a3b8;
}

.steps-bar {
  margin-bottom: 32px;
}

.step-body {
  min-height: 300px;
  padding: 8px 0;
}

.step-kicker {
  color: #6366f1;
  font-size: 12px;
  font-weight: 900;
  margin-bottom: 8px;
}

.step-body h2 {
  font-size: 21px;
  font-weight: 900;
  color: #0f172a;
  margin-bottom: 8px;
  letter-spacing: 0;
}

.step-desc {
  font-size: 13px;
  color: #64748b;
  line-height: 1.7;
  margin-bottom: 24px;
}

.form-group {
  margin-bottom: 18px;
}

.form-group label {
  display: block;
  font-size: 13px;
  font-weight: 600;
  color: #334155;
  margin-bottom: 6px;
}

.required {
  color: #e11d48;
}

.field-hint {
  display: block;
  font-size: 11.5px;
  color: #94a3b8;
  margin-top: 4px;
}

.password-rules {
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
  margin-top: 6px;
}
.password-rules span {
  padding: 5px 9px;
  border-radius: 999px;
  background: #f1f5f9;
  color: #64748b;
  font-size: 12px;
  font-weight: 800;
}
.password-rules span.ok {
  background: #ecfdf5;
  color: #047857;
}

.form-row {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 16px;
}

.switch-label {
  font-size: 13px;
  color: #475569;
  margin-left: 10px;
  vertical-align: middle;
}

.option-list {
  display: grid;
  gap: 12px;
  margin-bottom: 18px;
}

.option-card {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 18px;
  padding: 16px;
  border: 1px solid #e2e8f0;
  border-radius: 12px;
  background: #fff;
}
.option-card.active {
  border-color: #a5b4fc;
  background: #f8faff;
}
.option-card strong {
  display: block;
  color: #0f172a;
  font-size: 14px;
  margin-bottom: 3px;
}
.option-card span {
  display: block;
  color: #64748b;
  font-size: 12.5px;
  line-height: 1.5;
}

.section-divider {
  height: 1px;
  background: #eef0f4;
  margin: 20px 0;
}

.summary-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 14px;
}

.summary-item {
  background: #f8f9fc;
  border-radius: 10px;
  padding: 14px 16px;
}

.summary-label {
  display: block;
  font-size: 11.5px;
  color: #94a3b8;
  margin-bottom: 4px;
}

.summary-item strong {
  font-size: 14px;
  color: #0f172a;
}

.text-green { color: #16a34a; }
.text-muted { color: #94a3b8; }

.step-nav {
  display: flex;
  align-items: center;
  margin-top: 32px;
  padding-top: 20px;
  border-top: 1px solid #eef0f4;
}

.nav-spacer {
  flex: 1;
}

@media (max-width: 640px) {
  .setup-card { padding: 24px 18px; }
  .form-row { grid-template-columns: 1fr; }
  .summary-grid { grid-template-columns: 1fr; }
}
</style>
