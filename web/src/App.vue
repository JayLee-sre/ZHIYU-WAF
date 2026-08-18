<template>
  <div class="layout" v-if="!isStandalone">
    <!-- 侧边栏 -->
    <aside class="sidebar" :class="{ open: sidebarOpen }">
      <div class="sidebar-header">
        <div class="brand">
          <div class="brand-icon">
            <img src="/logo.png" alt="智域 WAF" />
          </div>
          <div class="brand-text">
            <span class="brand-name">智域 WAF</span>
            <span class="brand-tag">{{ editionLabel }}</span>
          </div>
        </div>
      </div>

      <nav class="nav-menu">
        <div class="nav-group" v-for="group in menuGroups" :key="group.label">
          <div class="nav-group-title">{{ group.label }}</div>
          <router-link
            v-for="item in group.items"
            :key="item.path"
            :to="item.path"
            class="nav-item"
            :class="{ active: route.path === item.path }"
          >
            <div class="nav-icon-wrap">
              <div class="nav-icon" :class="item.color">
                <el-icon :size="18"><component :is="item.icon" /></el-icon>
              </div>
            </div>
            <div class="nav-content">
              <span class="nav-label">{{ item.label }}</span>
              <span class="nav-desc">{{ item.desc }}</span>
            </div>
            <div class="nav-indicator" v-if="route.path === item.path"></div>
          </router-link>
        </div>
      </nav>

      <div class="sidebar-footer">
        <div class="engine-status">
          <div class="status-dot" :class="{ active: systemOk, warning: !systemOk }"></div>
          <span class="status-text">{{ systemOk ? '防护引擎运行中' : '防护状态待确认' }}</span>
        </div>
        <button class="logout-btn" @click="logout">
          <el-icon :size="16"><SwitchButton /></el-icon>
          <span>退出登录</span>
        </button>
      </div>
    </aside>

    <!-- 主内容区 -->
    <!-- 移动端遮罩 -->
    <div class="sidebar-overlay" v-if="sidebarOpen" @click="sidebarOpen = false"></div>

    <main class="main-area">
      <header class="topbar">
        <div class="topbar-left">
          <button class="hamburger-btn" @click="sidebarOpen = !sidebarOpen">
            <el-icon :size="20"><Fold v-if="sidebarOpen" /><Expand v-else /></el-icon>
          </button>
          <div class="breadcrumb">
            <span class="breadcrumb-current">{{ currentMenu?.label }}</span>
            <span class="breadcrumb-desc">{{ currentMenu?.desc }}</span>
          </div>
        </div>
        <div class="topbar-right">
          <div class="status-chips" v-if="!statusLoading">
            <div class="chip">
              <span class="chip-dot" :class="aiEnabled ? 'indigo' : 'muted'"></span>
              <span>AI {{ aiStatus }}</span>
            </div>
            <div class="chip">
              <span class="chip-dot emerald"></span>
              <span>规则 {{ ruleCount }} 条</span>
            </div>
            <div class="chip">
              <span class="chip-dot" :class="systemOk ? 'emerald' : 'rose'"></span>
              <span>{{ systemOk ? '系统正常' : '系统异常' }}</span>
            </div>
          </div>
        </div>
      </header>
      <div class="content-wrapper">
        <router-view v-slot="{ Component }">
          <transition name="page" mode="out-in">
            <component :is="Component" />
          </transition>
        </router-view>
      </div>
      <footer class="app-footer">ZHIYU-WAF V2 · 本地优先 · 全功能免费</footer>
    </main>
  </div>
  <router-view v-else />
</template>

<script setup>
import { ref, computed, onMounted, watch, provide } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessageBox } from 'element-plus'
import { Monitor, Document, SetUp, Filter, Cpu, SwitchButton, Setting, Key, Fold, Expand, Connection, Location, Warning, DataAnalysis, Lock, List } from '@element-plus/icons-vue'
import api, { clearAuthToken, getAuthToken } from './api'

const route = useRoute()
const router = useRouter()
const isLoginPage = computed(() => route.path === '/login')
const isStandalone = computed(() => route.path === '/login' || route.path === '/soc-dashboard' || route.path === '/setup' || route.path === '/welcome')
const edition = ref('free')
const isPro = computed(() => true) // 保留注入兼容性，V2 不再设有功能授权门槛。
const editionLabel = computed(() => 'V2 Free')
provide('edition', edition)
provide('isPro', isPro)
const sidebarOpen = ref(false)

watch(() => route.path, () => {
  document.body.style.background = '#f5f7fb'
}, { immediate: true })

watch(() => route.path, () => { sidebarOpen.value = false })

const menuGroups = [
  {
    label: '运营总览',
    items: [
      { path: '/dashboard', label: '安全态势', desc: '风险与拦截总览', icon: Monitor, color: 'indigo' },
      { path: '/logs', label: '攻击日志', desc: '威胁事件追踪', icon: Document, color: 'rose' },
      { path: '/soc-dashboard', label: '监控大屏', desc: '态势可视化', icon: DataAnalysis, color: 'green' },
    ],
  },
  {
    label: '防护策略',
    items: [
      { path: '/rules', label: '规则引擎', desc: '检测规则管理', icon: SetUp, color: 'amber' },
      { path: '/iplist', label: '访问控制', desc: 'IP 黑白名单', icon: Filter, color: 'cyan' },
      { path: '/geo', label: '地理封锁', desc: '国家/地区策略', icon: Location, color: 'indigo' },
      { path: '/threatintel', label: '威胁情报', desc: '恶意 IP 同步', icon: Warning, color: 'rose' },
    ],
  },
  {
    label: '资产与能力',
    items: [
      { path: '/sites', label: '站点管理', desc: '多站代理回源', icon: Connection, color: 'green' },
      { path: '/certs', label: 'SSL 证书', desc: 'TLS 证书管理', icon: Lock, color: 'green' },
      { path: '/ai', label: 'AI 模型', desc: '智能检测配置', icon: Cpu, color: 'violet' },
    ],
  },
  {
    label: '系统治理',
    items: [
      { path: '/ssh-logs', label: 'SSH 监控', desc: '暴力破解防护', icon: Key, color: 'amber' },
      { path: '/audit', label: '审计日志', desc: '操作记录追踪', icon: List, color: 'cyan' },
      { path: '/settings', label: '系统设置', desc: '防护与安全配置', icon: Setting, color: 'slate' },
    ],
  },
]

const menuItems = menuGroups.flatMap(group => group.items)
const currentMenu = computed(() => menuItems.find(i => i.path === route.path))
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
  document.title = pageName
    ? `${pageName} - 智域 WAF ${editionLabel.value}`
    : `智域 WAF ${editionLabel.value}`
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

watch(() => route.path, () => {
  if (!isStandalone.value) loadHeaderStatus()
})

watch([() => route.path, editionLabel], updateDocumentTitle, { immediate: true })

onMounted(() => loadHeaderStatus(true))

function logout() {
  ElMessageBox.confirm('确定退出当前账号？', '退出确认', {
    confirmButtonText: '退出',
    cancelButtonText: '取消',
    type: 'warning',
  }).then(() => {
    clearAuthToken()
    edition.value = 'free'
    router.push('/login')
  }).catch(() => {})
}
</script>

<style>
/* Page transitions */
.page-enter-active, .page-leave-active { transition: opacity 0.2s ease, transform 0.2s ease; }
.page-enter-from { opacity: 0; }
.page-leave-to { opacity: 0; }
</style>

<style scoped>
.layout {
  height: 100vh;
  display: flex;
  overflow: hidden;
  background: #fff;
}

/* ===== Sidebar ===== */
.sidebar {
  width: 252px;
  background: #fff;
  border-right: 1px solid #e5e7eb;
  display: flex;
  flex-direction: column;
  flex-shrink: 0;
  box-shadow: none;
}

.sidebar-header {
  padding: 18px 16px;
  border-bottom: 1px solid #eef2f7;
}

.brand {
  display: flex;
  align-items: center;
  gap: 12px;
}
.brand-icon {
  width: 40px;
  height: 40px;
  flex-shrink: 0;
  border-radius: 8px;
  overflow: hidden;
  background: #fff;
  box-shadow: 0 0 0 1px #e2e8f0;
}
.brand-icon img { width: 100%; height: 100%; object-fit: cover; display: block; }
.brand-text { display: flex; flex-direction: column; }
.brand-name { font-size: 18px; font-weight: 800; color: #0f172a; letter-spacing: 0; }
.brand-tag { font-size: 11px; color: #64748b; letter-spacing: 0; margin-top: 1px; }

.nav-menu {
  flex: 1;
  padding: 10px 12px 14px;
  overflow-y: auto;
}

.nav-group {
  padding: 10px 0;
  border-bottom: 1px solid #edf2f7;
}
.nav-group:last-child { border-bottom: none; }
.nav-group-title {
  padding: 0 10px 8px;
  color: #64748b;
  font-size: 10px;
  font-weight: 700;
  letter-spacing: .04em;
  text-transform: none;
}

.nav-item {
  display: flex;
  align-items: center;
  gap: 10px;
  min-height: 48px;
  padding: 8px 9px;
  border-radius: 10px;
  margin-bottom: 4px;
  text-decoration: none;
  color: #64748b;
  transition: all 0.2s ease;
  position: relative;
}
.nav-item:hover {
  background: #f3f7fc;
  color: #1e3a5f;
}
.nav-item.active {
  background: #eff6ff;
  color: #1d4ed8;
  box-shadow: inset 0 0 0 1px #dbeafe;
}

.nav-icon-wrap { flex-shrink: 0; }
.nav-icon {
  width: 32px; height: 32px;
  border-radius: 8px;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: all 0.2s;
}
.nav-icon.indigo { background: #eff6ff; color: #2563eb; }
.nav-icon.rose { background: #fff1f2; color: #e11d48; }
.nav-icon.amber { background: #fffbeb; color: #d97706; }
.nav-icon.cyan { background: #ecfeff; color: #0891b2; }
.nav-icon.violet { background: #f5f3ff; color: #7c3aed; }
.nav-icon.slate { background: #f1f5f9; color: #475569; }
.nav-icon.green { background: #f0fdf4; color: #16a34a; }

.nav-item.active .nav-icon.indigo { background: #2563eb; color: #fff; }
.nav-item.active .nav-icon.rose { background: #e11d48; color: #fff; }
.nav-item.active .nav-icon.amber { background: #d97706; color: #fff; }
.nav-item.active .nav-icon.cyan { background: #0891b2; color: #fff; }
.nav-item.active .nav-icon.violet { background: #7c3aed; color: #fff; }
.nav-item.active .nav-icon.slate { background: #334155; color: #fff; }
.nav-item.active .nav-icon.green { background: #16a34a; color: #fff; }

.nav-content { display: flex; flex-direction: column; min-width: 0; gap: 1px; }
.nav-label { font-size: 13px; font-weight: 700; }
.nav-desc { font-size: 11px; color: #94a3b8; margin-top: 0; }
.nav-item.active .nav-desc { color: #2563eb; }

.nav-indicator {
  position: absolute;
  left: 0; top: 50%;
  transform: translateY(-50%);
  width: 3px; height: 22px;
  border-radius: 3px;
  background: #2563eb;
  box-shadow: 0 0 0 3px rgba(37, 99, 235, .10);
}

.nav-lock {
  flex-shrink: 0;
  margin-left: auto;
  color: #d97706;
  opacity: 0.6;
}

.sidebar-footer {
  padding: 14px 18px;
  border-top: 1px solid #edf2f7;
  background: #fff;
}

.engine-status {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 12px;
}
.status-dot {
  width: 7px; height: 7px;
  border-radius: 50%;
  background: #94a3b8;
}
.status-dot.active {
  background: #22c55e;
  box-shadow: 0 0 6px rgba(34,197,94,0.5);
  animation: pulse 2s infinite;
}
.status-dot.warning {
  background: #f59e0b;
  box-shadow: 0 0 6px rgba(245,158,11,0.35);
}
@keyframes pulse { 0%, 100% { opacity: 1; } 50% { opacity: 0.5; } }
.status-text { font-size: 12px; color: #64748b; }

.logout-btn {
  width: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  padding: 8px;
  border: 1px solid #dfe7f1;
  border-radius: 8px;
  background: #fff;
  color: #64748b;
  font-size: 12.5px;
  cursor: pointer;
  transition: all 0.2s;
}
.logout-btn:hover {
  background: #fef2f2;
  border-color: #fecaca;
  color: #dc2626;
}

/* ===== Main Area ===== */
.main-area {
  flex: 1;
  display: flex;
  flex-direction: column;
  min-width: 0;
}

.topbar {
  height: 60px;
  background: #fff;
  backdrop-filter: blur(10px);
  -webkit-backdrop-filter: blur(10px);
  border-bottom: 1px solid #dfe7f1;
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 24px;
  flex-shrink: 0;
  position: sticky;
  top: 0;
  z-index: 100;
}

.topbar-left { display: flex; align-items: center; }
.breadcrumb { display: flex; align-items: baseline; gap: 10px; }
.breadcrumb-current { font-size: 16px; font-weight: 800; color: #0f172a; }
.breadcrumb-desc { font-size: 12.5px; color: #64748b; }

.topbar-right { display: flex; align-items: center; }
.status-chips { display: flex; gap: 8px; }
.chip {
  display: flex;
  align-items: center;
  gap: 6px;
  height: 30px;
  padding: 0 11px;
  background: #f8fafc;
  border: 1px solid #dfe7f1;
  border-radius: 999px;
  font-size: 12px;
  color: #475569;
  font-weight: 500;
  transition: background 0.2s ease;
}
.chip:hover {
  background: #f1f5f9;
}
.chip-dot {
  width: 6px; height: 6px;
  border-radius: 50%;
}
.chip-dot.indigo { background: #6366f1; }
.chip-dot.emerald {
  background: #22c55e;
  box-shadow: 0 0 4px rgba(34, 197, 94, 0.4);
}
.chip-dot.rose { background: #ef4444; }
.chip-dot.muted { background: #94a3b8; }

.content-wrapper {
  flex: 1;
  padding: 22px 24px;
  overflow-y: auto;
  background: #f8fafc;
}

.app-footer {
  flex-shrink: 0;
  height: 34px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-top: 1px solid #dfe7f1;
  background: #fff;
  color: #94a3b8;
  font-size: 12px;
}

/* ===== Mobile ===== */
.hamburger-btn {
  display: none;
  background: none; border: none; color: #475569; cursor: pointer;
  padding: 6px; border-radius: 8px;
}
.hamburger-btn:hover { background: #f1f5f9; }

.sidebar-overlay {
  display: none;
  position: fixed; inset: 0; background: rgba(0,0,0,0.35); z-index: 1000;
}

@media (max-width: 768px) {
  .sidebar {
    position: fixed; left: -252px; top: 0; bottom: 0;
    z-index: 1001; transition: left 0.3s ease;
    box-shadow: 4px 0 20px rgba(0,0,0,0.1);
  }
  .sidebar.open { left: 0; }
  .sidebar-overlay { display: block; }
  .hamburger-btn { display: flex; }
  .brand-tag { display: none; }
  .nav-group-title { padding-top: 4px; }
  .status-chips { display: none; }
  .topbar { padding: 0 16px; }
  .breadcrumb-desc { display: none; }
  .content-wrapper { padding: 16px; }
  .app-footer { height: 30px; font-size: 11px; }
}

/* White-console override: semantic colors belong to data, not navigation chrome. */
.nav-icon[class] { background: #f8fafc; color: #64748b; }
.nav-item.active .nav-icon[class] { background: #eff6ff; color: #2563eb; }
.nav-indicator { width: 2px; height: 18px; background: #2563eb; box-shadow: none; }
.sidebar-footer { background: #fff; }
.engine-status { padding: 8px 9px; margin-bottom: 10px; border-radius: 7px; background: #f8fafc; }
.topbar { border-bottom-color: #e5e7eb; }
.content-wrapper { padding: 20px 24px; }
@media (max-width: 768px) { .content-wrapper { padding: 14px; } }
</style>
