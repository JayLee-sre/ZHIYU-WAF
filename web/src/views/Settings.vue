<template>
  <div class="settings-page">
    <header class="settings-hero">
      <div class="hero-copy">
        <p class="eyebrow">SYSTEM CONTROL</p>
        <div class="hero-title-row">
          <div class="hero-icon"><el-icon :size="21"><Setting /></el-icon></div>
          <div>
            <h1>系统设置</h1>
            <p>集中管理防护开关、控制台访问、本地配置与恢复流程。</p>
          </div>
        </div>
        <div class="hero-tags" aria-label="系统能力状态">
          <span class="hero-tag good"><el-icon :size="14"><CircleCheck /></el-icon> V2 全功能免费</span>
          <span class="hero-tag">本地优先</span>
          <span class="hero-tag">配置可回滚</span>
        </div>
      </div>
      <button class="refresh-button" :disabled="loadingHealth" @click="loadHealth">
        <el-icon :size="15"><RefreshRight /></el-icon>
        {{ loadingHealth ? '同步中…' : '刷新状态' }}
      </button>
    </header>

    <section class="status-strip" aria-label="系统状态概览">
      <article class="status-item">
        <span class="status-kicker">运行状态</span>
        <strong><i class="status-dot" :class="healthTone"></i>{{ healthLabel }}</strong>
        <small>{{ healthHint }}</small>
      </article>
      <article class="status-item">
        <span class="status-kicker">动态防护</span>
        <strong>{{ settingsForm.dynamicProtect ? '已启用' : '未启用' }}</strong>
        <small>{{ settingsForm.dynamicProtect ? '随机化脚本注入正在生效' : '可在防护策略中随时开启' }}</small>
      </article>
      <article class="status-item">
        <span class="status-kicker">控制台账户</span>
        <strong>{{ users.length }} 个</strong>
        <small>{{ adminCount ? `${adminCount} 个管理员 · 最小权限可控` : '根管理员与角色账户分离管理' }}</small>
      </article>
    </section>

    <main class="settings-layout">
      <div class="settings-main">
        <section class="workspace-card protection-card">
          <div class="card-heading">
            <div>
              <p class="card-kicker">PROTECTION</p>
              <h2>防护策略</h2>
              <span>调整对业务请求生效的本地保护选项。</span>
            </div>
            <span class="card-state">即时配置</span>
          </div>

          <div class="policy-list">
            <div class="policy-row">
              <div class="policy-mark shield-mark">
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8"><path d="M12 3 4.5 6v5.5c0 4.6 3.2 7.9 7.5 9.5 4.3-1.6 7.5-4.9 7.5-9.5V6L12 3Z"/><path d="M8.5 12h7M12 8.5v7"/></svg>
              </div>
              <div class="policy-copy">
                <strong>动态防护</strong>
                <p>为请求注入随机化脚本，降低自动化工具与爬虫的分析效率。</p>
              </div>
              <label class="switch" :aria-label="settingsForm.dynamicProtect ? '关闭动态防护' : '开启动态防护'">
                <input type="checkbox" v-model="settingsForm.dynamicProtect" />
                <span class="switch-track"></span>
              </label>
            </div>
            <div class="policy-row policy-row-muted">
              <div class="policy-mark protocol-mark">
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8"><rect x="3" y="5" width="18" height="14" rx="3"/><path d="M7 10h5M7 14h10"/></svg>
              </div>
              <div class="policy-copy">
                <strong>HTTP/2 协商</strong>
                <p>通过 TLS ALPN 由监听器自动协商协议；请在证书与反向代理侧统一配置。</p>
              </div>
              <span class="neutral-chip">自动协商</span>
            </div>
          </div>

          <div class="card-action-row">
            <p>保存后会立即重载本地配置，不会清空数据或重启服务。</p>
            <button class="primary-button compact" :disabled="savingSettings" @click="saveSettings">
              {{ savingSettings ? '保存中…' : '保存防护策略' }}
            </button>
          </div>
        </section>

        <section class="workspace-card lifecycle-card">
          <div class="card-heading">
            <div>
              <p class="card-kicker">LIFECYCLE</p>
              <h2>配置与运维</h2>
              <span>只对本地文件、规则和运行状态执行受控操作。</span>
            </div>
          </div>

          <div class="operation-grid">
            <article class="operation-tile">
              <div class="tile-top">
                <span class="tile-icon blue"><el-icon :size="17"><RefreshRight /></el-icon></span>
                <span class="operation-label">无需重启</span>
              </div>
              <h3>重载配置</h3>
              <p>重新读取本地配置与规则，并同步当前防护状态。</p>
              <button class="secondary-button" :disabled="reloading" @click="reloadConfig">
                {{ reloading ? '重载中…' : '立即重载' }}
              </button>
            </article>
            <article class="operation-tile update-tile" :class="updateTileTone">
              <div class="tile-top">
                <span class="tile-icon slate">
                  <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8"><path d="M12 3v12"/><path d="m8 7 4-4 4 4"/><path d="M5 15v4a2 2 0 0 0 2 2h10a2 2 0 0 0 2-2v-4"/></svg>
                </span>
                <span class="operation-label" :class="updateTileTone">{{ updateLabel }}</span>
              </div>
              <h3>GitHub 更新检查</h3>
              <p>{{ updateSummary }}</p>
              <div class="update-meta" v-if="updateResult">
                <span>当前 {{ updateResult.current_version }}</span>
                <span v-if="updateResult.latest_version">最新 {{ updateResult.latest_version }}</span>
              </div>
              <p class="release-notes" v-if="updateResult?.release_notes">{{ updateResult.release_notes }}</p>
              <div class="update-actions">
                <button class="secondary-button neutral-action" :disabled="checkingUpdate" @click="checkVersionUpdate">
                  {{ checkingUpdate ? '检查中…' : '检查更新' }}
                </button>
                <a v-if="updateResult?.release_url" class="release-link" :href="updateResult.release_url" target="_blank" rel="noopener noreferrer">查看发行页</a>
              </div>
            </article>
          </div>
        </section>

        <section class="workspace-card backup-card">
          <div class="card-heading">
            <div>
              <p class="card-kicker">BACKUP &amp; RECOVERY</p>
              <h2>配置备份与恢复</h2>
              <span>导出或导入规则、IP 列表、站点、地理围栏与系统设置。</span>
            </div>
          </div>

          <div class="backup-content">
            <div class="backup-visual">
              <svg viewBox="0 0 52 52" fill="none" stroke="currentColor" stroke-width="1.55"><path d="M14 7h18l9 9v27a3 3 0 0 1-3 3H14a3 3 0 0 1-3-3V10a3 3 0 0 1 3-3Z"/><path d="M32 7v10h10M17 27h18M17 34h11"/><path d="m29 22 3-3 3 3M32 19v10"/></svg>
            </div>
            <div class="backup-copy">
              <strong>安全地迁移本地控制面</strong>
              <p>备份文件采用 JSON 格式，建议在更新规则或迁移主机前完成一次导出，并在非生产环境先验证恢复结果。</p>
              <div class="backup-actions">
                <button class="primary-button" :disabled="exporting" @click="exportBackup">
                  {{ exporting ? '导出中…' : '导出配置' }}
                </button>
                <label class="secondary-button upload-button" :class="{ disabled: importing }">
                  {{ importing ? '导入中…' : '导入备份' }}
                  <input type="file" accept=".json" @change="importBackup" :disabled="importing" hidden />
                </label>
              </div>
            </div>
          </div>

          <div class="import-result" v-if="importResult">
            <div class="result-heading"><el-icon :size="14"><CircleCheck /></el-icon> 导入结果</div>
            <div class="import-summary">
              <span v-for="(count, key) in importResult.imported" :key="key">{{ importLabel(key) }} {{ count }}</span>
            </div>
            <div class="import-errors" v-if="importResult.errors?.length">
              <div v-for="err in importResult.errors" :key="err" class="error-line">{{ err }}</div>
            </div>
          </div>
        </section>
      </div>

      <aside class="settings-side">
        <section class="workspace-card password-card">
          <div class="card-heading tight">
            <div>
              <p class="card-kicker">ACCOUNT SECURITY</p>
              <h2>管理员密码</h2>
              <span>建议使用独立、高强度密码。</span>
            </div>
            <span class="tile-icon amber"><el-icon :size="16"><Key /></el-icon></span>
          </div>
          <div class="password-fields">
            <label class="input-field">
              <span>当前密码</span>
              <div class="password-input">
                <input v-model="pwdForm.old_password" :type="showOld ? 'text' : 'password'" autocomplete="current-password" placeholder="输入当前密码" />
                <button type="button" class="toggle-vis" @click="showOld = !showOld" aria-label="显示或隐藏当前密码"><el-icon :size="14"><View v-if="!showOld" /><Hide v-else /></el-icon></button>
              </div>
            </label>
            <label class="input-field">
              <span>新密码</span>
              <div class="password-input">
                <input v-model="pwdForm.new_password" :type="showNew ? 'text' : 'password'" autocomplete="new-password" placeholder="至少 12 位字符" />
                <button type="button" class="toggle-vis" @click="showNew = !showNew" aria-label="显示或隐藏新密码"><el-icon :size="14"><View v-if="!showNew" /><Hide v-else /></el-icon></button>
              </div>
            </label>
            <label class="input-field">
              <span>确认新密码</span>
              <div class="password-input">
                <input v-model="pwdConfirm" :type="showConfirm ? 'text' : 'password'" autocomplete="new-password" placeholder="再次输入新密码" />
                <button type="button" class="toggle-vis" @click="showConfirm = !showConfirm" aria-label="显示或隐藏确认密码"><el-icon :size="14"><View v-if="!showConfirm" /><Hide v-else /></el-icon></button>
              </div>
            </label>
            <button class="primary-button full" :disabled="changingPwd" @click="changePassword">
              {{ changingPwd ? '更新中…' : '更新管理员密码' }}
            </button>
          </div>
        </section>

        <section class="workspace-card users-card">
          <div class="card-heading tight">
            <div>
              <p class="card-kicker">ACCESS CONTROL</p>
              <h2>用户与角色</h2>
              <span>为日常运维分配最小必要权限。</span>
            </div>
            <button class="text-button" @click="showCreateUser = !showCreateUser"><el-icon :size="13"><Plus /></el-icon>{{ showCreateUser ? '收起' : '添加用户' }}</button>
          </div>

          <div class="create-user-form" v-if="showCreateUser">
            <input v-model="newUser.username" class="form-input" placeholder="用户名" />
            <input v-model="newUser.password" type="password" class="form-input" placeholder="密码（至少 12 位）" />
            <select v-model="newUser.role" class="form-select">
              <option value="operator">操作员</option>
              <option value="viewer">只读用户</option>
              <option value="admin">管理员</option>
            </select>
            <button class="primary-button full" :disabled="creatingUser || !newUser.username || !newUser.password" @click="createUser">{{ creatingUser ? '创建中…' : '创建用户' }}</button>
          </div>

          <div class="user-list" v-if="users.length">
            <article class="user-row" v-for="u in users" :key="u.id">
              <span class="user-avatar" :class="u.role">{{ u.username.charAt(0).toUpperCase() }}</span>
              <div class="user-copy"><strong>{{ u.username }}</strong><span class="role-badge" :class="u.role">{{ roleLabel(u.role) }}</span></div>
              <button class="remove-user" @click="deleteUser(u)" :disabled="u.role === 'admin' && adminCount <= 1">删除</button>
            </article>
          </div>
          <div class="empty-users" v-else>暂无额外控制台用户。</div>
        </section>

        <section class="guidance-card">
          <p class="card-kicker">OPERATING NOTE</p>
          <strong>推荐变更顺序</strong>
          <ol>
            <li>先导出当前配置并保留恢复点。</li>
            <li>修改策略后执行重载并观察安全态势。</li>
            <li>为团队成员创建最小权限账户。</li>
          </ol>
        </section>
      </aside>
    </main>

    <teleport to="body">
      <div v-if="showUpdateModal" class="update-modal-backdrop" role="presentation" @click.self="showUpdateModal = false">
        <section class="update-modal" role="dialog" aria-modal="true" aria-labelledby="update-modal-title">
          <header class="update-modal-head">
            <div class="update-modal-mark"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8"><path d="M12 3v12"/><path d="m8 7 4-4 4 4"/><path d="M5 15v4a2 2 0 0 0 2 2h10a2 2 0 0 0 2-2v-4"/></svg></div>
            <div><span>NEW GITHUB RELEASE</span><h2 id="update-modal-title">发现新的正式版本</h2></div>
            <button type="button" class="update-modal-close" aria-label="关闭更新说明" @click="showUpdateModal = false">×</button>
          </header>
          <div class="update-modal-body">
            <div class="version-comparison"><div><span>当前构建</span><strong>{{ updateResult?.current_version || '-' }}</strong></div><i>→</i><div class="latest"><span>最新正式版</span><strong>{{ updateResult?.latest_version || '-' }}</strong></div></div>
            <div class="update-modal-notes"><span class="modal-kicker">更新说明</span><pre>{{ updateResult?.release_notes || '发布方未提供详细说明。' }}</pre></div>
            <p class="update-modal-hint">该提示仅展示发行说明，不会自动下载、替换二进制或重启当前服务。</p>
          </div>
          <footer class="update-modal-actions"><button type="button" class="secondary-button" @click="showUpdateModal = false">稍后查看</button><a v-if="updateResult?.release_url" class="primary-button" :href="updateResult.release_url" target="_blank" rel="noopener noreferrer" @click="showUpdateModal = false">查看完整发行说明</a></footer>
        </section>
      </div>
    </teleport>
  </div>
</template>

<script setup>
import { computed, ref, reactive, onMounted } from 'vue'
import { CircleCheck, Hide, Key, RefreshRight, View, Setting, Plus } from '@element-plus/icons-vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import api from '../api'

const health = ref({})
const loadingHealth = ref(false)
const showOld = ref(false)
const showNew = ref(false)
const showConfirm = ref(false)
const changingPwd = ref(false)
const reloading = ref(false)
const checkingUpdate = ref(false)
const updateResult = ref(null)
const showUpdateModal = ref(false)
const savingSettings = ref(false)
const pwdForm = reactive({ old_password: '', new_password: '' })
const pwdConfirm = ref('')
const settingsForm = reactive({ dynamicProtect: false })

const exporting = ref(false)
const importing = ref(false)
const importResult = ref(null)

const users = ref([])
const showCreateUser = ref(false)
const creatingUser = ref(false)
const newUser = reactive({ username: '', password: '', role: 'operator' })
const adminCount = computed(() => users.value.filter(u => u.role === 'admin').length)
const healthTone = computed(() => health.value?.status === 'ok' ? 'ok' : health.value?.status ? 'warning' : 'muted')
const healthLabel = computed(() => health.value?.status === 'ok' ? '系统正常' : health.value?.status ? '需要关注' : '等待状态同步')
const healthHint = computed(() => health.value?.status === 'ok' ? '本地控制面与服务健康检查可用' : '可使用刷新状态重新获取服务信息')
const updateTileTone = computed(() => {
  const status = updateResult.value?.status
  if (status === 'update_available') return 'warning'
  if (status === 'up_to_date') return 'good'
  if (status === 'unavailable') return 'danger'
  return 'neutral'
})
const updateLabel = computed(() => ({
  update_available: '发现更新',
  up_to_date: '已是最新',
  no_release: '等待发布',
  unavailable: '暂不可用',
}[updateResult.value?.status] || 'GITHUB RELEASES'))
const updateSummary = computed(() => updateResult.value?.message || '从 GitHub Releases 查询正式发布版本；不自动下载或覆盖当前服务。')

async function loadHealth() {
  loadingHealth.value = true
  try {
    const [h, settings] = await Promise.all([
      api.get('/health', { suppressError: true }),
      api.get('/settings', { suppressError: true }),
    ])
    health.value = h || {}
    if (settings) settingsForm.dynamicProtect = settings.dynamic_protect === 'true'
  } catch {
    health.value = {}
  } finally {
    loadingHealth.value = false
  }
}

async function changePassword() {
  if (!pwdForm.old_password || !pwdForm.new_password || !pwdConfirm.value) return ElMessage.warning('请填写完整密码')
  if (pwdForm.new_password.length < 12) return ElMessage.warning('新密码至少 12 位字符')
  if (pwdForm.new_password !== pwdConfirm.value) return ElMessage.warning('两次输入的新密码不一致')
  changingPwd.value = true
  try {
    await api.post('/auth/password', pwdForm)
    ElMessage.success('密码已更新，请重新登录')
    pwdForm.old_password = ''
    pwdForm.new_password = ''
    pwdConfirm.value = ''
  } finally {
    changingPwd.value = false
  }
}

async function reloadConfig() {
  if (reloading.value) return
  reloading.value = true
  try {
    const response = await api.post('/config/reload')
    ElMessage.success(response.message || '配置已重载')
    await loadHealth()
  } finally {
    reloading.value = false
  }
}

async function saveSettings() {
  savingSettings.value = true
  try {
    await api.put('/settings', { dynamic_protect: String(settingsForm.dynamicProtect) })
    ElMessage.success('防护策略已保存')
    await reloadConfig()
  } finally {
    savingSettings.value = false
  }
}

async function checkVersionUpdate() {
  if (checkingUpdate.value) return
  checkingUpdate.value = true
  try {
    const result = await api.post('/system/update/check', {})
    updateResult.value = result
    if (result.status === 'update_available') {
      showUpdateModal.value = true
      ElMessage.warning('发现新的正式版本，已打开更新说明。')
    } else if (result.status === 'up_to_date') ElMessage.success(result.message)
    else ElMessage.info(result.message)
  } catch (error) {
    updateResult.value = { status: 'unavailable', message: error.message || '无法完成 GitHub 更新检查。' }
    ElMessage.error(updateResult.value.message)
  } finally {
    checkingUpdate.value = false
  }
}

async function exportBackup() {
  exporting.value = true
  try {
    const response = await fetch('/api/v1/backup/export', { headers: { Authorization: `Bearer ${localStorage.getItem('zhiyu_waf_token')}` } })
    const backup = await response.json()
    const url = URL.createObjectURL(new Blob([JSON.stringify(backup, null, 2)], { type: 'application/json' }))
    const link = document.createElement('a')
    link.href = url
    link.download = `zhiyu-waf-backup-${new Date().toISOString().slice(0, 10)}.json`
    link.click()
    URL.revokeObjectURL(url)
    ElMessage.success('配置已导出')
  } finally {
    exporting.value = false
  }
}

async function importBackup(event) {
  const file = event.target.files?.[0]
  if (!file) return
  importing.value = true
  importResult.value = null
  try {
    const data = JSON.parse(await file.text())
    const result = await api.post('/backup/import', data)
    importResult.value = result
    result.errors?.length ? ElMessage.warning(`导入完成，${result.errors.length} 个错误`) : ElMessage.success('配置已导入')
  } catch (error) {
    ElMessage.error('导入失败: ' + (error.message || '未知错误'))
  } finally {
    importing.value = false
    event.target.value = ''
  }
}

function importLabel(key) { return { rules: '规则', ip_entries: 'IP', sites: '站点', geo_rules: '地理围栏', settings: '设置' }[key] || key }

async function loadUsers() {
  try { users.value = await api.get('/users') || [] } catch { users.value = [] }
}

async function createUser() {
  if (!newUser.username || !newUser.password) return
  creatingUser.value = true
  try {
    await api.post('/users', { ...newUser })
    ElMessage.success('用户已创建')
    newUser.username = ''
    newUser.password = ''
    newUser.role = 'operator'
    showCreateUser.value = false
    await loadUsers()
  } finally {
    creatingUser.value = false
  }
}

async function deleteUser(user) {
  try { await ElMessageBox.confirm(`确定删除用户 "${user.username}"？`, '删除确认', { type: 'warning' }) } catch { return }
  try {
    await api.delete(`/users/${user.id}`)
    ElMessage.success('用户已删除')
    await loadUsers()
  } catch {}
}

function roleLabel(role) { return { admin: '管理员', operator: '操作员', viewer: '只读' }[role] || role }

onMounted(async () => {
  await loadHealth()
  await loadUsers()
})
</script>

<style scoped>
.settings-page { max-width: 1240px; display: flex; flex-direction: column; gap: 16px; color: #0f172a; }
.settings-hero { display: flex; align-items: flex-start; justify-content: space-between; gap: 20px; padding: 24px 26px; border: 1px solid #e2e8f0; border-radius: 14px; background: linear-gradient(112deg, #ffffff 0%, #f8fbff 100%); box-shadow: 0 10px 28px rgba(15, 23, 42, .035); }
.hero-copy { min-width: 0; }.eyebrow,.card-kicker,.status-kicker { display: block; margin: 0; color: #64748b; font: 700 10px/1.2 ui-monospace, SFMono-Regular, Menlo, monospace; letter-spacing: .1em; }
.hero-title-row { display: flex; align-items: center; gap: 13px; margin-top: 9px; }.hero-icon { display: grid; flex: 0 0 auto; width: 42px; height: 42px; place-items: center; border: 1px solid #bfdbfe; border-radius: 12px; background: #eff6ff; color: #2563eb; }.hero-title-row h1 { margin: 0; font-size: 22px; line-height: 1.15; letter-spacing: -.03em; }.hero-title-row p { margin: 4px 0 0; color: #64748b; font-size: 12.5px; }.hero-tags { display: flex; flex-wrap: wrap; gap: 7px; margin-top: 17px; }.hero-tag,.card-state,.neutral-chip,.operation-label { display: inline-flex; align-items: center; gap: 5px; border-radius: 999px; padding: 4px 8px; background: #f1f5f9; color: #475569; font-size: 11px; font-weight: 700; }.hero-tag.good { background: #ecfdf5; color: #047857; }.refresh-button,.primary-button,.secondary-button,.text-button,.remove-user { display: inline-flex; align-items: center; justify-content: center; gap: 7px; border-radius: 8px; font: inherit; font-weight: 700; cursor: pointer; transition: transform 160ms cubic-bezier(.23, 1, .32, 1), border-color 160ms ease, background 160ms ease, color 160ms ease; }.refresh-button:active,.primary-button:active,.secondary-button:active,.text-button:active { transform: scale(.97); }.refresh-button { min-height: 34px; padding: 0 11px; border: 1px solid #dbe3ee; background: #fff; color: #475569; font-size: 12px; }.refresh-button:hover { border-color: #93c5fd; color: #2563eb; }.refresh-button:disabled,.primary-button:disabled,.secondary-button:disabled { cursor: not-allowed; opacity: .55; }
.status-strip { display: grid; grid-template-columns: repeat(3, 1fr); overflow: hidden; border: 1px solid #e2e8f0; border-radius: 12px; background: #fff; }.status-item { min-width: 0; padding: 14px 18px; border-right: 1px solid #eef2f7; }.status-item:last-child { border-right: 0; }.status-item strong { display: flex; align-items: center; gap: 7px; margin-top: 6px; font-size: 15px; }.status-item small { display: block; overflow: hidden; margin-top: 4px; color: #64748b; font-size: 11px; text-overflow: ellipsis; white-space: nowrap; }.status-dot { width: 7px; height: 7px; border-radius: 50%; background: #94a3b8; }.status-dot.ok { background: #16a34a; box-shadow: 0 0 0 4px #f0fdf4; }.status-dot.warning { background: #d97706; box-shadow: 0 0 0 4px #fffbeb; }
.settings-layout { display: grid; grid-template-columns: minmax(0, 1.48fr) minmax(320px, .92fr); align-items: start; gap: 16px; }.settings-main,.settings-side { display: flex; flex-direction: column; gap: 16px; }.workspace-card { overflow: hidden; border: 1px solid #e2e8f0; border-radius: 13px; background: #fff; box-shadow: 0 7px 22px rgba(15, 23, 42, .025); }.card-heading { display: flex; align-items: flex-start; justify-content: space-between; gap: 14px; padding: 18px 20px 16px; border-bottom: 1px solid #eef2f7; }.card-heading.tight { padding-bottom: 14px; }.card-heading h2 { margin: 5px 0 0; font-size: 15px; letter-spacing: -.01em; }.card-heading span:not(.card-kicker):not(.tile-icon):not(.text-button) { display: block; margin-top: 4px; color: #64748b; font-size: 12px; line-height: 1.55; }.card-state { flex: 0 0 auto; margin-top: 3px; background: #eff6ff; color: #2563eb; }.policy-list { padding: 4px 20px; }.policy-row { display: flex; align-items: center; gap: 12px; padding: 16px 0; border-bottom: 1px solid #f1f5f9; }.policy-row:last-child { border-bottom: 0; }.policy-mark,.tile-icon { display: grid; flex: 0 0 auto; width: 34px; height: 34px; place-items: center; border-radius: 10px; }.policy-mark svg,.tile-icon svg { width: 17px; height: 17px; }.shield-mark { background: #eff6ff; color: #2563eb; }.protocol-mark { background: #f8fafc; color: #64748b; }.policy-copy { flex: 1; min-width: 0; }.policy-copy strong { display: block; font-size: 13px; }.policy-copy p { margin: 3px 0 0; color: #64748b; font-size: 12px; line-height: 1.5; }.neutral-chip { flex: 0 0 auto; color: #64748b; }.switch { position: relative; flex: 0 0 auto; width: 42px; height: 24px; }.switch input { width: 0; height: 0; opacity: 0; }.switch-track { position: absolute; inset: 0; border-radius: 999px; background: #cbd5e1; cursor: pointer; transition: background 160ms ease; }.switch-track::before { position: absolute; left: 3px; top: 3px; width: 18px; height: 18px; border-radius: 50%; background: #fff; box-shadow: 0 1px 2px rgba(15,23,42,.16); content: ''; transition: transform 180ms cubic-bezier(.23, 1, .32, 1); }.switch input:checked + .switch-track { background: #2563eb; }.switch input:checked + .switch-track::before { transform: translateX(18px); }.card-action-row { display: flex; align-items: center; justify-content: space-between; gap: 14px; padding: 13px 20px; border-top: 1px solid #eef2f7; background: #fafcff; }.card-action-row p { margin: 0; color: #64748b; font-size: 11px; line-height: 1.5; }.primary-button { min-height: 34px; padding: 0 13px; border: 1px solid #2563eb; background: #2563eb; color: #fff; font-size: 12px; }.primary-button:hover { border-color: #1d4ed8; background: #1d4ed8; }.primary-button.compact { flex: 0 0 auto; }.primary-button.full { width: 100%; }.operation-grid { display: grid; grid-template-columns: 1fr 1fr; gap: 12px; padding: 16px 20px 20px; }.operation-tile { min-height: 164px; padding: 15px; border: 1px solid #dbeafe; border-radius: 11px; background: #f8fbff; }.operation-tile.muted-tile { border-color: #e2e8f0; background: #fafafa; }.operation-tile.update-tile { border-color: #e2e8f0; background: #fafcff; }.operation-tile.update-tile.good { border-color: #bbf7d0; background: #f0fdf4; }.operation-tile.update-tile.warning { border-color: #fde68a; background: #fffbeb; }.operation-tile.update-tile.danger { border-color: #fecaca; background: #fffafa; }.tile-top { display: flex; align-items: center; justify-content: space-between; gap: 8px; }.tile-icon.blue { background: #dbeafe; color: #2563eb; }.tile-icon.slate { background: #f1f5f9; color: #475569; }.tile-icon.amber { background: #fffbeb; color: #b45309; }.operation-label { padding: 3px 6px; background: #eff6ff; color: #2563eb; font-size: 10px; }.operation-label.neutral { background: #f1f5f9; color: #64748b; }.operation-label.good { background: #dcfce7; color: #047857; }.operation-label.warning { background: #fef3c7; color: #b45309; }.operation-label.danger { background: #fee2e2; color: #b91c1c; }.operation-tile h3 { margin: 13px 0 0; font-size: 13px; }.operation-tile p { min-height: 37px; margin: 5px 0 14px; color: #64748b; font-size: 11.5px; line-height: 1.55; }.secondary-button { min-height: 32px; padding: 0 11px; border: 1px solid #bfdbfe; background: #fff; color: #2563eb; font-size: 11.5px; }.secondary-button:hover { border-color: #2563eb; background: #eff6ff; }.neutral-action { border-color: #cbd5e1; color: #475569; }.neutral-action:hover { background: #f8fafc; color: #334155; }.update-meta { display: flex; flex-wrap: wrap; gap: 5px; margin: -7px 0 10px; }.update-meta span { border-radius: 5px; padding: 3px 6px; background: rgba(255,255,255,.76); color: #475569; font: 700 10px ui-monospace, SFMono-Regular, Menlo, monospace; }.release-notes { display: -webkit-box; overflow: hidden; min-height: 0 !important; margin: -3px 0 10px !important; color: #64748b; font-size: 10.5px !important; line-height: 1.45 !important; white-space: pre-line; -webkit-box-orient: vertical; -webkit-line-clamp: 3; }.update-actions { display: flex; flex-wrap: wrap; align-items: center; gap: 8px; }.release-link { color: #2563eb; font-size: 11px; font-weight: 700; text-decoration: none; }.release-link:hover { text-decoration: underline; }.backup-content { display: flex; align-items: center; gap: 15px; padding: 18px 20px; }.backup-visual { display: grid; flex: 0 0 auto; width: 58px; height: 58px; place-items: center; border-radius: 14px; background: #f5f3ff; color: #7c3aed; }.backup-visual svg { width: 34px; height: 34px; }.backup-copy { min-width: 0; }.backup-copy strong { font-size: 13px; }.backup-copy p { margin: 5px 0 12px; color: #64748b; font-size: 12px; line-height: 1.6; }.backup-actions { display: flex; flex-wrap: wrap; gap: 8px; }.upload-button { box-sizing: border-box; }.upload-button.disabled { cursor: not-allowed; opacity: .55; }.import-result { margin: 0 20px 18px; padding: 12px; border: 1px solid #bfdbfe; border-radius: 10px; background: #f8fbff; }.result-heading { display: flex; align-items: center; gap: 6px; color: #2563eb; font-size: 11px; font-weight: 800; }.import-summary { display: flex; flex-wrap: wrap; gap: 6px; margin-top: 9px; }.import-summary span { border-radius: 5px; padding: 3px 7px; background: #fff; color: #475569; font-size: 11px; font-weight: 700; }.import-errors { margin-top: 8px; padding-top: 8px; border-top: 1px solid #dbeafe; }.error-line { color: #dc2626; font-size: 11px; line-height: 1.5; }
.password-fields { display: flex; flex-direction: column; gap: 12px; padding: 16px 20px 20px; }.input-field > span { display: block; margin-bottom: 6px; color: #475569; font-size: 11px; font-weight: 700; }.password-input { position: relative; }.password-input input,.form-input,.form-select { box-sizing: border-box; width: 100%; height: 38px; border: 1px solid #dbe3ee; border-radius: 8px; padding: 0 36px 0 10px; outline: none; background: #fff; color: #0f172a; font: inherit; font-size: 12px; transition: border-color 160ms ease, box-shadow 160ms ease; }.password-input input:focus,.form-input:focus,.form-select:focus { border-color: #60a5fa; box-shadow: 0 0 0 3px rgba(37,99,235,.09); }.password-input input::placeholder,.form-input::placeholder { color: #94a3b8; }.toggle-vis { position: absolute; right: 5px; top: 50%; display: grid; width: 28px; height: 28px; place-items: center; transform: translateY(-50%); border: 0; border-radius: 6px; background: transparent; color: #94a3b8; cursor: pointer; }.toggle-vis:hover { color: #2563eb; background: #eff6ff; }.text-button { flex: 0 0 auto; min-height: 30px; border: 0; background: transparent; color: #2563eb; font-size: 11px; }.text-button:hover { background: #eff6ff; }.create-user-form { display: grid; grid-template-columns: 1fr 1fr; gap: 8px; margin: 0 20px 13px; padding: 12px; border: 1px solid #dbeafe; border-radius: 10px; background: #f8fbff; }.create-user-form .form-select { padding-right: 8px; }.create-user-form .primary-button { grid-column: 1 / -1; }.user-list { display: flex; flex-direction: column; padding: 4px 20px 17px; }.user-row { display: flex; align-items: center; gap: 10px; padding: 10px 0; border-bottom: 1px solid #f1f5f9; }.user-row:last-child { border-bottom: 0; }.user-avatar { display: grid; flex: 0 0 auto; width: 31px; height: 31px; place-items: center; border-radius: 9px; font-size: 12px; font-weight: 800; }.user-avatar.admin { background: #eff6ff; color: #2563eb; }.user-avatar.operator { background: #ecfdf5; color: #047857; }.user-avatar.viewer { background: #f1f5f9; color: #64748b; }.user-copy { flex: 1; min-width: 0; }.user-copy strong { display: block; overflow: hidden; font-size: 12px; text-overflow: ellipsis; white-space: nowrap; }.role-badge { display: inline-block; margin-top: 2px; border-radius: 4px; padding: 1px 5px; font-size: 10px; font-weight: 700; }.role-badge.admin { background: #eff6ff; color: #2563eb; }.role-badge.operator { background: #ecfdf5; color: #047857; }.role-badge.viewer { background: #f1f5f9; color: #64748b; }.remove-user { min-height: 26px; padding: 0 6px; border: 0; background: transparent; color: #dc2626; font-size: 10.5px; }.remove-user:hover { background: #fef2f2; }.remove-user:disabled { cursor: not-allowed; opacity: .35; }.empty-users { padding: 14px 20px 19px; color: #94a3b8; font-size: 12px; }.guidance-card { padding: 17px 19px; border: 1px solid #dbeafe; border-radius: 13px; background: linear-gradient(135deg, #eff6ff, #fff); }.guidance-card strong { display: block; margin-top: 7px; color: #1e3a8a; font-size: 13px; }.guidance-card ol { margin: 10px 0 0 17px; padding: 0; color: #475569; font-size: 11.5px; line-height: 1.85; }
.update-modal-backdrop { position: fixed; z-index: 5000; inset: 0; display: grid; place-items: center; padding: 20px; background: rgba(15, 23, 42, .48); backdrop-filter: blur(5px); }.update-modal { width: min(650px, 100%); overflow: hidden; border: 1px solid #dbeafe; border-radius: 16px; background: #fff; box-shadow: 0 28px 72px rgba(15, 23, 42, .28); }.update-modal-head { display: flex; align-items: center; gap: 12px; padding: 20px 22px 17px; border-bottom: 1px solid #eaf0f7; background: linear-gradient(135deg, #eff6ff, #fff); }.update-modal-mark { display: grid; width: 39px; height: 39px; place-items: center; border-radius: 11px; background: #dbeafe; color: #2563eb; }.update-modal-mark svg { width: 20px; height: 20px; }.update-modal-head > div:nth-child(2) { flex: 1; min-width: 0; }.update-modal-head span,.modal-kicker { display: block; color: #2563eb; font: 800 9px/1.2 ui-monospace, SFMono-Regular, Menlo, monospace; letter-spacing: .1em; }.update-modal-head h2 { margin: 5px 0 0; color: #0f172a; font-size: 18px; letter-spacing: -.02em; }.update-modal-close { display: grid; width: 30px; height: 30px; place-items: center; border: 0; border-radius: 7px; background: transparent; color: #94a3b8; font-size: 24px; line-height: 1; cursor: pointer; }.update-modal-close:hover { background: #f1f5f9; color: #334155; }.update-modal-body { padding: 20px 22px; }.version-comparison { display: grid; grid-template-columns: 1fr 26px 1fr; align-items: center; gap: 9px; padding: 12px; border: 1px solid #e2e8f0; border-radius: 10px; background: #f8fafc; }.version-comparison div span,.version-comparison div strong { display: block; }.version-comparison div span { color: #64748b; font-size: 10px; }.version-comparison div strong { margin-top: 4px; color: #334155; font: 800 15px ui-monospace, SFMono-Regular, Menlo, monospace; }.version-comparison .latest { padding: 8px 10px; border-radius: 8px; background: #eff6ff; }.version-comparison .latest strong { color: #1d4ed8; }.version-comparison > i { color: #94a3b8; font-style: normal; text-align: center; }.update-modal-notes { margin-top: 17px; }.update-modal-notes pre { max-height: 285px; margin: 8px 0 0; overflow: auto; padding: 13px; border: 1px solid #e2e8f0; border-radius: 9px; background: #fbfdff; color: #475569; font: 12px/1.65 var(--font-body, Inter, "PingFang SC", "Microsoft YaHei", sans-serif); white-space: pre-wrap; word-break: break-word; }.update-modal-hint { margin: 12px 0 0; color: #64748b; font-size: 11px; line-height: 1.55; }.update-modal-actions { display: flex; justify-content: flex-end; gap: 8px; padding: 14px 22px; border-top: 1px solid #eef2f7; background: #fafcff; }.update-modal-actions .primary-button { text-decoration: none; } @media (max-width: 960px) { .settings-layout { grid-template-columns: 1fr; }.settings-side { display: grid; grid-template-columns: 1fr 1fr; align-items: start; }.guidance-card { grid-column: 1 / -1; }.password-card,.users-card { height: 100%; } }
@media (max-width: 680px) { .update-modal-backdrop { align-items: end; padding: 10px; }.update-modal { border-radius: 15px 15px 10px 10px; }.update-modal-head,.update-modal-body,.update-modal-actions { padding-left: 16px; padding-right: 16px; }.update-modal-actions { flex-direction: column-reverse; }.update-modal-actions > * { width: 100%; }.settings-hero { flex-direction: column; padding: 19px; }.refresh-button { width: 100%; }.status-strip { grid-template-columns: 1fr; }.status-item { border-right: 0; border-bottom: 1px solid #eef2f7; }.status-item:last-child { border-bottom: 0; }.settings-side { display: flex; }.operation-grid { grid-template-columns: 1fr; }.backup-content { align-items: flex-start; }.card-action-row { align-items: flex-start; flex-direction: column; }.card-action-row .primary-button { width: 100%; }.create-user-form { grid-template-columns: 1fr; }.hero-title-row h1 { font-size: 20px; } }
@media (prefers-reduced-motion: reduce) { .refresh-button,.primary-button,.secondary-button,.text-button,.switch-track,.switch-track::before { transition: none; } }
</style>
