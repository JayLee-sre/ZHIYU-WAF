<template>
  <div class="layout" v-if="!isStandalone">
    <aside class="sidebar" :class="{ open: sidebarOpen }">
      <div class="sidebar-header">
        <div class="brand">
          <div class="brand-icon"><img src="/logo.png" alt="智域 WAF" /></div>
          <div class="brand-text">
            <span class="brand-name">智域 WAF</span>
            <span class="brand-tag">V2 · 本地安全控制台</span>
          </div>
        </div>
      </div>

      <nav class="nav-menu" aria-label="控制台主导航">
        <div class="nav-group" v-for="group in menuGroups" :key="group.label">
          <div class="nav-group-title">{{ group.label }}</div>
          <router-link
            v-for="item in group.items"
            :key="item.path"
            :to="item.path"
            class="nav-item"
            :class="{ active: route.path === item.path }"
          >
            <div class="nav-icon"><el-icon :size="17"><component :is="item.icon" /></el-icon></div>
            <div class="nav-content">
              <span class="nav-label">{{ item.label }}</span>
              <span class="nav-desc">{{ item.desc }}</span>
            </div>
            <span class="nav-indicator" v-if="route.path === item.path"></span>
          </router-link>
        </div>
      </nav>

      <div class="sidebar-footer">
        <div class="engine-status">
          <span class="status-dot" :class="{ active: systemOk, warning: !systemOk }"></span>
          <div>
            <strong>{{ systemOk ? '防护引擎运行中' : '防护状态待确认' }}</strong>
            <span>本地风险链路</span>
          </div>
        </div>
        <button class="logout-btn" @click="logout"><el-icon :size="15"><SwitchButton /></el-icon><span>退出登录</span></button>
      </div>
    </aside>

    <div class="sidebar-overlay" v-if="sidebarOpen" @click="sidebarOpen = false"></div>

    <main class="main-area">
      <header class="topbar">
        <div class="topbar-left">
          <button class="hamburger-btn" @click="sidebarOpen = !sidebarOpen" aria-label="切换导航菜单"><el-icon :size="20"><Fold v-if="sidebarOpen" /><Expand v-else /></el-icon></button>
          <div class="page-context">
            <span class="workspace-label">CONTROL PLANE</span>
            <div class="breadcrumb">
              <span class="breadcrumb-current">{{ currentMenu?.label }}</span>
              <span class="breadcrumb-desc">{{ currentMenu?.desc }}</span>
            </div>
          </div>
        </div>
        <div class="topbar-right">
          <div class="status-chips" v-if="!statusLoading" aria-label="系统运行摘要">
            <div class="chip"><span class="chip-dot" :class="aiEnabled ? 'blue' : 'muted'"></span><span>AI {{ aiStatus }}</span></div>
            <div class="chip"><span class="chip-dot emerald"></span><span>规则 {{ ruleCount }} 条</span></div>
            <div class="chip system-chip" :class="{ attention: !systemOk }"><span class="chip-dot" :class="systemOk ? 'emerald' : 'rose'"></span><span>{{ systemOk ? '系统正常' : '需要检查' }}</span></div>
          </div>
        </div>
      </header>

      <div class="content-wrapper">
        <router-view v-slot="{ Component }">
          <transition name="page" mode="out-in"><component :is="Component" /></transition>
        </router-view>
      </div>
      <footer class="app-footer"><span>ZHIYU-WAF V2</span><i></i><span>本地优先 · 全功能免费</span></footer>
    </main>
  </div>
  <router-view v-else />
</template>

<script setup>
import { ref, computed, onMounted, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessageBox } from 'element-plus'
import { Monitor, Document, SetUp, Filter, Cpu, SwitchButton, Setting, Key, Fold, Expand, Connection, Location, Warning, DataAnalysis, Lock, List } from '@element-plus/icons-vue'
import api, { clearAuthToken, getAuthToken } from './api'

const route = useRoute()
const router = useRouter()
const isLoginPage = computed(() => route.path === '/login')
const isStandalone = computed(() => ['/login', '/soc-dashboard', '/setup'].includes(route.path))
const edition = ref('free')
const editionLabel = computed(() => 'V2 Free')
const sidebarOpen = ref(false)

watch(() => route.path, () => { document.body.style.background = '#f6f8fb' }, { immediate: true })
watch(() => route.path, () => { sidebarOpen.value = false })

const menuGroups = [
  { label: '运营总览', items: [
    { path: '/dashboard', label: '安全态势', desc: '风险与拦截总览', icon: Monitor },
    { path: '/logs', label: '攻击日志', desc: '威胁事件追踪', icon: Document },
    { path: '/soc-dashboard', label: '监控大屏', desc: '态势可视化', icon: DataAnalysis },
  ] },
  { label: '防护策略', items: [
    { path: '/rules', label: '规则引擎', desc: '检测规则管理', icon: SetUp },
    { path: '/iplist', label: '访问控制', desc: 'IP 黑白名单', icon: Filter },
    { path: '/geo', label: '地理封锁', desc: '国家/地区策略', icon: Location },
    { path: '/threatintel', label: '威胁情报', desc: '恶意 IP 同步', icon: Warning },
  ] },
  { label: '资产与能力', items: [
    { path: '/sites', label: '站点管理', desc: '多站代理回源', icon: Connection },
    { path: '/certs', label: 'SSL 证书', desc: 'TLS 证书管理', icon: Lock },
    { path: '/ai', label: 'AI 模型', desc: '智能检测配置', icon: Cpu },
  ] },
  { label: '系统治理', items: [
    { path: '/ssh-logs', label: 'SSH 监控', desc: '暴力破解防护', icon: Key },
    { path: '/audit', label: '审计日志', desc: '操作记录追踪', icon: List },
    { path: '/settings', label: '系统设置', desc: '防护与安全配置', icon: Setting },
  ] },
]

const menuItems = menuGroups.flatMap(group => group.items)
const currentMenu = computed(() => menuItems.find(item => item.path === route.path))
const statusLoading = ref(true)
const aiEnabled = ref(false)
const systemOk = ref(true)
const ruleCount = ref(0)
const aiStatus = computed(() => aiEnabled.value ? '已启用' : '未启用')
let headerStatusLoadedAt = 0
let headerStatusRequest = null

function updateDocumentTitle() {
  let pageName = currentMenu.value?.label
  if (route.path === '/login') pageName = '登录'
  else if (route.path === '/soc-dashboard') pageName = '监控大屏'
  document.title = pageName ? `${pageName} - 智域 WAF ${editionLabel.value}` : `智域 WAF ${editionLabel.value}`
}

async function loadHeaderStatus(force = false) {
  if (isLoginPage.value || !getAuthToken()) return
  const now = Date.now()
  if (!force && headerStatusRequest) return headerStatusRequest
  if (!force && now - headerStatusLoadedAt < 15000) return
  statusLoading.value = true
  headerStatusRequest = Promise.all([
    api.get('/health', { suppressError: true }),
    api.get('/rules', { suppressError: true }),
  ]).then(([health, rules]) => {
    systemOk.value = health?.status === 'ok'
    aiEnabled.value = !!health?.ai_enabled
    ruleCount.value = Array.isArray(rules) ? rules.length : 0
    edition.value = 'free'
    headerStatusLoadedAt = Date.now()
  }).catch(() => {
    edition.value = 'free'
    systemOk.value = false
  }).finally(() => {
    statusLoading.value = false
    headerStatusRequest = null
  })
  return headerStatusRequest
}

watch(() => route.path, () => { if (!isStandalone.value) loadHeaderStatus() })
watch([() => route.path, editionLabel], updateDocumentTitle, { immediate: true })
onMounted(() => loadHeaderStatus(true))

function logout() {
  ElMessageBox.confirm('确定退出当前账号？', '退出确认', { confirmButtonText: '退出', cancelButtonText: '取消', type: 'warning' })
    .then(() => { clearAuthToken(); edition.value = 'free'; router.push('/login') })
    .catch(() => {})
}
</script>

<style>
.page-enter-active, .page-leave-active { transition: opacity 180ms cubic-bezier(.23, 1, .32, 1); }
.page-enter-from, .page-leave-to { opacity: 0; }
</style>

<style scoped>
.layout { height: 100vh; display: flex; overflow: hidden; background: #f6f8fb; }
.sidebar { width: 238px; display: flex; flex-direction: column; flex-shrink: 0; border-right: 1px solid #e2e8f0; background: #fff; }
.sidebar-header { padding: 19px 17px 16px; border-bottom: 1px solid #eef2f7; }
.brand { display: flex; align-items: center; gap: 10px; }.brand-icon { width: 37px; height: 37px; overflow: hidden; flex: 0 0 auto; border: 1px solid #dbe4ef; border-radius: 10px; background: #fff; }.brand-icon img { width: 100%; height: 100%; display: block; object-fit: cover; }.brand-text { display: flex; min-width: 0; flex-direction: column; }.brand-name { color: #0f172a; font-size: 16px; font-weight: 800; letter-spacing: -.02em; }.brand-tag { margin-top: 2px; color: #94a3b8; font-size: 10px; }
.nav-menu { flex: 1; overflow-y: auto; padding: 10px 10px 16px; }.nav-group { padding: 10px 0 9px; }.nav-group + .nav-group { border-top: 1px solid #f1f5f9; }.nav-group-title { padding: 0 10px 7px; color: #94a3b8; font-size: 10px; font-weight: 800; letter-spacing: .08em; }.nav-item { position: relative; display: flex; align-items: center; gap: 9px; min-height: 45px; margin: 2px 0; padding: 7px 9px; border-radius: 8px; color: #64748b; text-decoration: none; transition: color 160ms ease, background 160ms ease; }.nav-item:hover { color: #334155; background: #f8fafc; }.nav-item.active { color: #1d4ed8; background: #eff6ff; }.nav-icon { display: grid; width: 29px; height: 29px; place-items: center; flex: 0 0 auto; border: 1px solid #e7edf5; border-radius: 7px; background: #fff; color: #64748b; transition: color 160ms ease, background 160ms ease, border-color 160ms ease; }.nav-item.active .nav-icon { border-color: #bfdbfe; background: #dbeafe; color: #2563eb; }.nav-content { display: flex; min-width: 0; flex-direction: column; gap: 1px; }.nav-label { color: inherit; font-size: 12.5px; font-weight: 700; }.nav-desc { overflow: hidden; color: #94a3b8; font-size: 10.5px; text-overflow: ellipsis; white-space: nowrap; }.nav-item.active .nav-desc { color: #60a5fa; }.nav-indicator { position: absolute; left: 0; width: 2px; height: 19px; border-radius: 99px; background: #2563eb; }
.sidebar-footer { padding: 12px 13px 14px; border-top: 1px solid #eef2f7; }.engine-status { display: flex; align-items: center; gap: 8px; margin-bottom: 9px; padding: 8px 9px; border: 1px solid #edf2f7; border-radius: 8px; background: #f8fafc; }.engine-status strong, .engine-status span { display: block; }.engine-status strong { color: #475569; font-size: 11px; line-height: 1.25; }.engine-status span { margin-top: 2px; color: #94a3b8; font-size: 10px; }.status-dot { width: 7px; height: 7px; flex: 0 0 auto; border-radius: 50%; background: #94a3b8; }.status-dot.active { background: #22c55e; box-shadow: 0 0 0 3px rgba(34, 197, 94, .12); }.status-dot.warning { background: #f59e0b; box-shadow: 0 0 0 3px rgba(245, 158, 11, .12); }.logout-btn { display: inline-flex; width: 100%; align-items: center; justify-content: center; gap: 6px; height: 33px; border: 1px solid #dbe4ef; border-radius: 7px; background: #fff; color: #64748b; font-size: 11.5px; font-weight: 700; cursor: pointer; transition: color 160ms ease, border-color 160ms ease, background 160ms ease, transform 160ms cubic-bezier(.23, 1, .32, 1); }.logout-btn:hover { border-color: #fecaca; background: #fef2f2; color: #dc2626; }.logout-btn:active { transform: scale(.98); }
.main-area { display: flex; min-width: 0; flex: 1; flex-direction: column; }.topbar { position: sticky; top: 0; z-index: 100; display: flex; height: 64px; align-items: center; justify-content: space-between; padding: 0 26px; border-bottom: 1px solid #e2e8f0; background: rgba(255, 255, 255, .93); backdrop-filter: blur(12px); -webkit-backdrop-filter: blur(12px); }.topbar-left, .topbar-right { display: flex; align-items: center; }.page-context { display: flex; flex-direction: column; gap: 3px; }.workspace-label { color: #94a3b8; font-family: ui-monospace, SFMono-Regular, Menlo, monospace; font-size: 9px; font-weight: 800; letter-spacing: .11em; }.breadcrumb { display: flex; align-items: baseline; gap: 9px; }.breadcrumb-current { color: #0f172a; font-size: 15px; font-weight: 800; letter-spacing: -.015em; }.breadcrumb-desc { color: #94a3b8; font-size: 11.5px; }.status-chips { display: flex; gap: 7px; }.chip { display: inline-flex; align-items: center; gap: 6px; height: 28px; padding: 0 10px; border: 1px solid #e2e8f0; border-radius: 999px; background: #fff; color: #64748b; font-size: 11px; font-weight: 700; }.system-chip { border-color: #dcfce7; background: #f0fdf4; color: #15803d; }.system-chip.attention { border-color: #fed7aa; background: #fff7ed; color: #c2410c; }.chip-dot { width: 6px; height: 6px; border-radius: 50%; }.chip-dot.blue { background: #2563eb; }.chip-dot.emerald { background: #22c55e; }.chip-dot.rose { background: #ef4444; }.chip-dot.muted { background: #94a3b8; }
.content-wrapper { flex: 1; overflow-y: auto; padding: 22px 26px; background: linear-gradient(180deg, #f8fafc 0%, #f5f7fb 100%); }.app-footer { display: flex; height: 32px; align-items: center; justify-content: center; gap: 8px; border-top: 1px solid #e2e8f0; background: #fff; color: #94a3b8; font-size: 10.5px; }.app-footer span:first-child { color: #64748b; font-weight: 800; }.app-footer i { width: 3px; height: 3px; border-radius: 50%; background: #cbd5e1; }
.hamburger-btn { display: none; margin-right: 10px; padding: 6px; border: 0; border-radius: 7px; background: transparent; color: #475569; cursor: pointer; }.hamburger-btn:hover { background: #f1f5f9; }.sidebar-overlay { display: none; position: fixed; inset: 0; z-index: 1000; background: rgba(15, 23, 42, .32); }
@media (max-width: 768px) { .sidebar { position: fixed; top: 0; bottom: 0; left: -238px; z-index: 1001; transition: left 220ms cubic-bezier(.23, 1, .32, 1); box-shadow: 8px 0 26px rgba(15, 23, 42, .14); }.sidebar.open { left: 0; }.sidebar-overlay { display: block; }.hamburger-btn { display: grid; place-items: center; }.topbar { height: 58px; padding: 0 15px; }.workspace-label, .breadcrumb-desc, .status-chips { display: none; }.content-wrapper { padding: 14px; }.app-footer { height: 29px; font-size: 10px; } }
@media (prefers-reduced-motion: reduce) { .nav-item, .nav-icon, .logout-btn, .sidebar { transition: none; } }
</style>
