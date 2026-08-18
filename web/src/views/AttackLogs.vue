<template>
  <div class="logs-page">
    <!-- 页面标题 -->
    <div class="page-toolbar">
      <div class="heading-group">
        <div class="heading-icon rose"><el-icon :size="18"><Document /></el-icon></div>
        <div>
          <div class="page-heading">攻击日志</div>
          <div class="page-sub">威胁事件追踪与分析</div>
        </div>
      </div>
    </div>

    <!-- 筛选栏 -->
    <div class="filter-bar">
      <div class="filter-group">
        <div class="filter-item">
          <label>站点范围</label>
          <select v-model="filters.site_id" class="filter-select site-select" @change="page = 1; loadLogs()">
            <option value="">全部站点</option>
            <option v-for="site in sites" :key="site.id" :value="site.id">
              {{ site.domain }} · {{ site.backend_url }}
            </option>
          </select>
        </div>
        <div class="filter-item">
          <label>来源 IP</label>
          <input v-model="filters.client_ip" placeholder="输入 IP 地址" class="filter-input" @keyup.enter="loadLogs" />
        </div>
        <div class="filter-item">
          <label>最终动作</label>
          <select v-model="filters.action" class="filter-select" @change="page = 1; loadLogs()">
            <option value="">全部</option>
            <option value="block">拦截</option>
            <option value="rate_limit">限流</option>
            <option value="log">仅记录</option>
            <option value="allow">放行</option>
          </select>
        </div>
        <div class="filter-item">
          <label>事件类别</label>
          <input v-model="filters.category" placeholder="如 xss" class="filter-input" @keyup.enter="page = 1; loadLogs()" />
        </div>
      </div>
      <div class="filter-actions">
        <button class="btn-primary" @click="loadLogs">
          <el-icon :size="14"><Search /></el-icon> 查询
        </button>
        <button class="btn-ghost" @click="resetFilters">重置</button>
      </div>
    </div>

    <!-- 数据表 -->
    <div class="table-card" v-if="logs.length || loading">
      <div class="table-header">
        <span class="table-title">攻击日志</span>
        <span class="table-count">{{ total }} 条记录</span>
      </div>
      <table class="data-table">
        <thead>
          <tr>
            <th>时间</th>
            <th>站点</th>
            <th>来源 IP</th>
            <th>方法</th>
            <th>攻击路径</th>
            <th>
              <span class="th-help">
                风险证据
                <button class="help-btn" type="button" @click.stop="toggleHelp('result')" aria-label="风险证据说明">?</button>
              </span>
            </th>
            <th>风险分</th>
            <th>最终动作</th>
            <th></th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="log in logs" :key="log.id" @click="showDetail(log)" class="clickable-row">
            <td class="mono time-cell">{{ fmt(log.created_at) }}</td>
            <td class="site-cell">{{ siteName(log) }}</td>
            <td class="mono">{{ log.client_ip }}</td>
            <td><span class="method-tag" :class="log.method?.toLowerCase()">{{ log.method }}</span></td>
            <td class="path-cell" :title="log.path">{{ log.path }}</td>
            <td class="result-cell">{{ evidence(log) }}</td>
            <td><span class="severity-pill" :class="riskLevel(log)">{{ log.risk_score ?? 0 }}</span></td>
            <td><span class="action-chip" :class="`action-${String(log.action || 'allow').replace('_', '-')}`">{{ actionText(log.action) }}</span></td>
            <td><span class="detail-link">详情</span></td>
          </tr>
        </tbody>
      </table>

      <div class="help-popover" v-if="helpKey === 'result'">
        <button class="help-close" @click="helpKey = ''">×</button>
        <strong>检测结果怎么看？</strong>
        <p>这里显示系统判断请求异常的主要原因。可以理解为“为什么这次访问被认为有风险”，例如尝试注入数据库、访问敏感路径、提交异常脚本或出现自动化扫描特征。</p>
      </div>

      <div class="pagination-bar" v-if="total > 0">
        <span class="page-info">共 {{ total }} 条 / {{ totalPages }} 页</span>
        <div class="page-controls">
          <select v-model="pageSize" class="page-size-select" @change="page = 1; loadLogs()">
            <option :value="10">10 条/页</option>
            <option :value="20">20 条/页</option>
            <option :value="50">50 条/页</option>
          </select>
          <button class="page-btn" :disabled="page <= 1" @click="page = 1; loadLogs()">首页</button>
          <button class="page-btn" :disabled="page <= 1" @click="page--; loadLogs()">上一页</button>
          <input type="number" v-model.number="jumpPage" class="page-jump" min="1" :max="totalPages" @keyup.enter="doJump" />
          <button class="page-btn" @click="doJump">跳转</button>
          <button class="page-btn" :disabled="page >= totalPages" @click="page++; loadLogs()">下一页</button>
          <button class="page-btn" :disabled="page >= totalPages" @click="page = totalPages; loadLogs()">末页</button>
        </div>
      </div>
    </div>

    <div class="empty-panel" v-else>
      <div class="empty-icon">
        <el-icon :size="30"><Document /></el-icon>
      </div>
      <div class="empty-title">暂无攻击记录</div>
      <div class="empty-desc">当前筛选条件下没有检测到攻击事件。可以调整筛选条件，或等待新的访问流量进入 WAF。</div>
      <button class="btn-ghost" @click="resetFilters">重置筛选</button>
    </div>

    <!-- 详情弹窗 -->
    <div class="modal-overlay" v-if="detailVisible" @click.self="detailVisible = false">
      <div class="modal-card">
        <div class="modal-header">
          <div>
            <div class="modal-title">风险事件详情</div>
            <div class="modal-subtitle">请求 ID: {{ d?.request_id || d?.id }}</div>
          </div>
          <button class="modal-close" @click="detailVisible = false">
            <el-icon :size="18"><Close /></el-icon>
          </button>
        </div>
        <div class="modal-body" v-if="d">
          <div class="detail-grid">
            <div class="detail-row">
              <span class="detail-label">时间</span>
              <span class="detail-value">{{ fmt(d.created_at) }}</span>
            </div>
            <div class="detail-row">
              <span class="detail-label">所属站点</span>
              <span class="detail-value">{{ siteName(d) }}</span>
            </div>
            <div class="detail-row" v-if="d.host">
              <span class="detail-label">访问域名</span>
              <span class="detail-value mono">{{ d.host }}</span>
            </div>
            <div class="detail-row">
              <span class="detail-label">来源 IP</span>
              <span class="detail-value mono">{{ d.client_ip }}</span>
            </div>
            <div class="detail-row">
              <span class="detail-label">请求方法</span>
              <span class="detail-value"><span class="method-tag" :class="d.method?.toLowerCase()">{{ d.method }}</span></span>
            </div>
            <div class="detail-row">
              <span class="detail-label">请求路径</span>
              <span class="detail-value mono">{{ d.path }}</span>
            </div>
            <div class="detail-row">
              <span class="detail-label label-with-help">
                风险证据
                <button class="help-btn" type="button" @click.stop="toggleHelp('detail-result')" aria-label="风险证据说明">?</button>
              </span>
              <span class="detail-value">
                {{ evidence(d) }}
                <span class="plain-explain">{{ detectionExplain(d) }}</span>
              </span>
            </div>
            <div class="detail-row">
              <span class="detail-label">风险分</span>
              <span class="detail-value"><span class="severity-pill" :class="riskLevel(d)">{{ d.risk_score ?? 0 }}</span></span>
            </div>
            <div class="detail-row">
              <span class="detail-label">最终动作</span>
              <span class="detail-value"><span class="action-chip" :class="`action-${String(d.action || 'allow').replace('_', '-')}`">{{ actionText(d.action) }}</span></span>
            </div>
          </div>
          <div class="detail-help" v-if="helpKey === 'detail-result' || helpKey === 'ai-result'">
            <button class="help-close" @click="helpKey = ''">×</button>
            <strong>{{ helpKey === 'ai-result' ? 'AI 分析说明' : '检测结果说明' }}</strong>
            <p>风险证据是本地防护链命中的规则标识或分类，用来说明这次请求为什么被判定为存在风险。建议结合来源 IP、路径、风险分和最终动作复核。</p>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, reactive, computed, onMounted } from 'vue'
import { Search, Document, Close } from '@element-plus/icons-vue'
import api from '../api'

const logs = ref([]), total = ref(0), page = ref(1), pageSize = ref(20), jumpPage = ref(1)
const loading = ref(false), detailVisible = ref(false), d = ref(null)
const sites = ref([])
const helpKey = ref('')
const filters = reactive({ site_id: '', client_ip: '', action: '', category: '' })
const totalPages = computed(() => Math.max(1, Math.ceil(total.value / pageSize.value)))

function fmt(ts) { return ts ? new Date(ts).toLocaleString('zh-CN') : '-' }
function riskLevel(log) {
  const score = Number(log?.risk_score || 0)
  if (score >= 90) return 'critical'
  if (score >= 60) return 'high'
  if (score >= 30) return 'medium'
  return 'low'
}
function evidence(log) { return log?.rule_id || log?.category || '本地风险事件' }
function actionText(action) { return { block: '拦截', rate_limit: '限流', log: '仅记录', allow: '放行' }[action] || action || '放行' }
function siteName(log) {
  const site = sites.value.find(item => Number(item.id) === Number(log?.site_id))
  return site?.domain || log?.host || (log?.site_id ? `站点 #${log.site_id}` : '默认站点')
}
function resetFilters() { Object.assign(filters, { site_id: '', client_ip: '', action: '', category: '' }); page.value = 1; loadLogs() }
function toggleHelp(key) { helpKey.value = helpKey.value === key ? '' : key }
function detectionExplain(log) {
  if (!log) return ''
  const name = `${evidence(log)}`.toLowerCase()
  if (name.includes('sql') || name.includes('sqli')) return '请求里像是在尝试拼接数据库语句，可能用于读取或修改数据。'
  if (name.includes('xss')) return '请求里包含可疑脚本，可能影响访问者浏览器安全。'
  if (name.includes('cmd') || name.includes('command')) return '请求疑似尝试执行服务器命令，风险较高。'
  if (name.includes('traversal') || name.includes('path')) return '请求疑似尝试访问非公开文件或目录。'
  if (name.includes('sensitive')) return '请求访问了敏感路径或敏感文件，建议确认是否为正常业务。'
  if (Number(log.risk_score || 0) >= 60) return '系统判断这次访问风险较高，建议优先查看来源和请求内容。'
  return '系统发现异常访问特征，建议结合业务场景确认是否为正常请求。'
}

async function loadSites() {
  try {
    const response = await api.get('/sites')
    sites.value = response?.data || response || []
  } catch {
    sites.value = []
  }
}

async function loadLogs() {
  loading.value = true
  try {
    const params = { offset: (page.value - 1) * pageSize.value, page_size: pageSize.value }
    if (filters.site_id) params.site_id = filters.site_id
    if (filters.client_ip) params.ip = filters.client_ip
    if (filters.action) params.action = filters.action
    if (filters.category) params.category = filters.category
    const response = await api.get('/events', { params })
    const data = response?.data || response || {}
    logs.value = data.items || []
    total.value = Number(data.total || 0)
    jumpPage.value = page.value
  } catch {
    logs.value = []
    total.value = 0
  } finally {
    loading.value = false
  }
}
function doJump() {
  const p = Math.min(Math.max(Number(jumpPage.value) || 1, 1), totalPages.value)
  page.value = p
  jumpPage.value = p
  loadLogs()
}
function showDetail(row) { d.value = row; detailVisible.value = true; helpKey.value = '' }
onMounted(() => { loadSites(); loadLogs() })
</script>

<style scoped>
.logs-page { }

/* Filter Bar */
.filter-bar {
  background: var(--bg-card); border-radius: var(--radius-card); border: 1px solid var(--border);
  padding: 18px 20px; margin-bottom: 16px;
  display: flex; justify-content: space-between; align-items: flex-end;
}
.filter-group { display: flex; gap: 16px; }
.filter-item { display: flex; flex-direction: column; gap: 5px; }
.filter-item label { font-size: 11.5px; font-weight: 600; color: var(--text-secondary); text-transform: uppercase; letter-spacing: 0.3px; }
.filter-input { width: 160px; }
.filter-select { width: 110px; cursor: pointer; }
.filter-select.site-select { width: 240px; }
.filter-actions { display: flex; gap: 8px; }
.btn-primary.danger { background: var(--danger); }
.btn-primary.danger:hover { background: #b91c1c; }

/* Table header override */
.table-header {
  display: flex; justify-content: space-between; align-items: center;
  padding: var(--card-pad); border-bottom: 1px solid var(--border-light);
}
.table-title { font-size: 14.5px; font-weight: 700; color: var(--text-primary); }
.table-count { font-size: 12px; color: var(--text-muted); background: var(--border-light); padding: 3px 10px; border-radius: 10px; }

.clickable-row { cursor: pointer; transition: background 0.15s; }
.clickable-row:hover { background: var(--bg-hover); }
.th-help, .label-with-help { display: inline-flex; align-items: center; gap: 6px; }
.help-btn {
  width: 18px; height: 18px; border-radius: 999px; border: 1px solid #cbd5e1;
  background: #fff; color: var(--text-secondary); font-size: 12px; line-height: 1; font-weight: 800;
  cursor: pointer;
}
.help-btn:hover { border-color: var(--primary); color: var(--primary-hover); background: var(--primary-light); }
.help-popover, .detail-help {
  background: #fff; border: 1px solid #dbe3ef; border-radius: var(--radius-card);
  box-shadow: 0 12px 32px rgba(15, 23, 42, 0.12);
  color: var(--text-secondary); font-size: 13px; line-height: 1.7;
}
.help-popover {
  position: absolute; right: 18px; top: 54px; z-index: 5;
  width: min(420px, calc(100% - 36px)); padding: 16px 18px;
}
.detail-help { position: relative; padding: 16px 18px; margin-top: 4px; }
.help-popover strong, .detail-help strong { display: block; color: var(--text-primary); margin-bottom: 6px; font-size: 14px; }
.help-close {
  position: absolute; top: 8px; right: 10px; border: none; background: transparent;
  color: var(--text-muted); font-size: 18px; cursor: pointer;
}

.time-cell { white-space: nowrap; }
.region-cell { max-width: 140px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; color: var(--text-secondary); font-size: 12.5px; }
.site-cell {
  max-width: 150px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap;
  color: var(--text-secondary); font-weight: 700;
}
.path-cell { max-width: 200px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
  .result-cell { max-width: 170px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
  .action-chip { display: inline-flex; align-items: center; border-radius: 999px; padding: 3px 8px; font-size: 11px; font-weight: 700; background: #f1f5f9; color: #475569; }
  .action-chip.action-block { background: #fef2f2; color: #dc2626; }
  .action-chip.action-rate-limit { background: #fffbeb; color: #a16207; }
  .action-chip.action-log { background: #eff6ff; color: #1d4ed8; }
  .action-chip.action-allow { background: #ecfdf5; color: #047857; }
  .detail-link { font-size: 12px; color: var(--primary); font-weight: 600; }

/* Override method-tag to global colors */
.method-tag.get { background: #eff6ff; color: #2563eb; }
.method-tag.post { background: #fff7ed; color: #ea580c; }
.method-tag.put { background: #f5f3ff; color: #7c3aed; }
.method-tag.delete { background: #fef2f2; color: #dc2626; }

/* Pagination override */
.pagination-bar {
  display: flex; justify-content: space-between; align-items: center;
  padding: 12px 20px; border-top: 1px solid var(--border-light);
}

/* Modal detail */
.modal-subtitle { font-size: 12px; color: var(--text-muted); margin-top: 2px; font-family: var(--font-mono); }
.detail-grid { display: flex; flex-direction: column; gap: 14px; }
.detail-row { display: flex; gap: 16px; }
.detail-label { width: 80px; font-size: 12.5px; color: var(--text-muted); font-weight: 500; flex-shrink: 0; }
.detail-value { font-size: 13px; color: #1e293b; flex: 1; }
.plain-explain {
  display: block; margin-top: 7px; color: var(--text-secondary); line-height: 1.65;
  white-space: normal; font-size: 13px;
}
.review-actions { display: flex; gap: 8px; flex-wrap: wrap; }
.review-actions .btn-primary,
.review-actions .btn-ghost { padding: 7px 12px; }

@media (max-width: 768px) {
  .filter-bar { flex-direction: column; gap: 12px; }
  .filter-group { flex-wrap: wrap; }
  .filter-input, .filter-select, .filter-select.site-select { width: 100%; min-width: 0; }
  .filter-actions { width: 100%; }
  .filter-actions .btn-primary { flex: 1; }
  .help-popover { left: 12px; right: 12px; width: auto; }
  .table-card { overflow-x: auto; }
  .data-table { min-width: 1040px; }
  .modal-card { width: 92vw; max-height: 90vh; }
  .pagination-bar { flex-direction: column; gap: 8px; }
}
</style>
