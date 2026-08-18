<template>
  <div class="dashboard">
    <header class="page-header">
      <div>
        <p class="eyebrow">OVERVIEW</p>
        <h1>安全态势</h1>
        <p class="subtitle">统一查看请求、风险事件和防护动作。</p>
      </div>
      <div class="header-actions">
        <select v-model="range" @change="loadAll">
          <option value="1h">最近 1 小时</option><option value="6h">最近 6 小时</option><option value="24h">最近 24 小时</option><option value="7d">最近 7 天</option><option value="30d">最近 30 天</option>
        </select>
        <button class="primary" :disabled="loading" @click="loadAll">{{ loading ? '刷新中…' : '刷新' }}</button>
      </div>
    </header>

    <section class="runtime-status" :class="{ warning: system.firewall?.degraded }">
      <span class="status-dot"></span>
      <div><strong>{{ system.firewall?.available ? 'nftables 防火墙已连接' : '应用层防护运行中' }}</strong><span>{{ firewallSummary }}</span></div>
      <button v-if="system.firewall?.degraded" type="button" class="text-btn" @click="showFirewallDetail = !showFirewallDetail">{{ showFirewallDetail ? '收起详情' : '查看详情' }}</button>
    </section>
    <div v-if="showFirewallDetail && system.firewall?.message" class="runtime-detail"><code>{{ system.firewall.message }}</code></div>

    <section class="metrics" aria-label="安全指标">
      <article v-for="card in cards" :key="card.label" class="metric"><span>{{ card.label }}</span><strong>{{ formatNumber(card.value) }}</strong><small>{{ card.note }}</small></article>
    </section>

    <section class="card trend-card">
      <div class="card-head"><div><h2>风险趋势</h2><p>按时间聚合请求、拦截和高风险事件。</p></div><span class="range-label">{{ range }}</span></div>
      <div v-if="series.length" class="chart" :class="{ sparse: series.length <= 4 }">
        <div v-for="point in series" :key="point.timestamp" class="chart-point" :title="`${point.timestamp}: ${point.requests} 请求 / ${point.blocked} 拦截`"><div class="bars"><i class="request" :style="{ height: barHeight(point.requests) }"></i><i class="blocked" :style="{ height: barHeight(point.blocked) }"></i><i class="risk" :style="{ height: barHeight(point.high_risk) }"></i></div><small>{{ shortTime(point.timestamp) }}</small></div>
      </div>
      <div v-else class="chart-empty"><strong>暂无趋势数据</strong><span>防护链收到请求后，会在此展示风险变化。</span></div>
      <div class="legend"><span><i class="request"></i>请求</span><span><i class="blocked"></i>拦截</span><span><i class="risk"></i>高风险</span></div>
    </section>

    <div class="two-col">
      <section class="card"><div class="card-head"><div><h2>威胁类别</h2><p>按主要检测证据聚合。</p></div></div><div v-if="summary.top_categories?.length" class="category-list"><div v-for="item in summary.top_categories" :key="item.category" class="category-row"><span>{{ item.category }}</span><div><i :style="{ width: categoryWidth(item.count) }"></i></div><strong>{{ item.count }}</strong></div></div><div v-else class="compact-empty">暂无分类命中。</div></section>
      <section class="card"><div class="card-head"><div><h2>风险动作</h2><p>每个请求只输出一次最终动作。</p></div></div><dl class="action-list"><div><dt>ALLOW</dt><dd>正常回源</dd></div><div><dt>LOG</dt><dd>记录风险证据</dd></div><div><dt>RATE LIMIT</dt><dd>限制异常频率</dd></div><div><dt>BLOCK</dt><dd>阻断高风险请求</dd></div></dl></section>
    </div>

    <section class="card events"><div class="card-head"><div><h2>最近安全事件</h2><p>仅保存定位和审计所需的信息。</p></div><router-link to="/logs">查看全部</router-link></div><div v-if="summary.recent_events?.length" class="table-wrap"><table><thead><tr><th>时间</th><th>来源</th><th>请求</th><th>证据</th><th>风险</th><th>动作</th></tr></thead><tbody><tr v-for="event in summary.recent_events" :key="event.id"><td>{{ formatTime(event.created_at) }}</td><td class="mono">{{ event.client_ip }}</td><td><b>{{ event.method }}</b> <span class="path">{{ event.path }}</span></td><td>{{ event.rule_id || event.category || '-' }}</td><td>{{ event.risk_score }}</td><td><span class="action-pill" :class="String(event.action || 'allow').replace('_', '-')">{{ event.action || 'allow' }}</span></td></tr></tbody></table></div><div v-else class="compact-empty">暂无安全事件。</div></section>
  </div>
</template>
<script setup>
import { computed, onMounted, ref } from 'vue'
import api from '../api'
const range = ref('24h'), loading = ref(false), summary = ref({ top_categories: [], recent_events: [] }), series = ref([]), system = ref({ firewall: {} }), showFirewallDetail = ref(false)
const cards = computed(() => [{ label: '请求总数', value: summary.value.request_count || 0, note: '进入 V2 防护链' }, { label: '已拦截', value: summary.value.blocked_count || 0, note: '最终动作：BLOCK' }, { label: '攻击来源', value: summary.value.attack_ip_count || 0, note: '风险分数 ≥ 30' }, { label: '高风险事件', value: summary.value.high_risk_count || 0, note: '风险分数 ≥ 60' }])
const firewallSummary = computed(() => system.value.firewall?.available ? '内核封禁集合可用。' : system.value.firewall?.degraded ? '内核封禁不可用，已降级为应用层阻断。' : '本地规则与风险引擎正在运行。')
function unwrap(response) { return response?.data ?? response ?? {} }
async function loadAll() { loading.value = true; try { const [nextSummary, nextSeries, nextSystem] = await Promise.all([api.get('/dashboard/summary', { params: { range: range.value } }), api.get('/dashboard/timeseries', { params: { range: range.value } }), api.get('/system/status', { suppressError: true })]); summary.value = unwrap(nextSummary); series.value = unwrap(nextSeries); system.value = unwrap(nextSystem); showFirewallDetail.value = false } finally { loading.value = false } }
function formatNumber(value) { return Number(value || 0).toLocaleString('zh-CN') }
function formatTime(value) { return value ? new Date(value).toLocaleString('zh-CN', { hour12: false }) : '-' }
function shortTime(value) { return value ? String(value).slice(5, 16) : '' }
function barHeight(value) { const max = Math.max(...series.value.flatMap(p => [p.requests || 0, p.blocked || 0, p.high_risk || 0]), 1); return `${Math.max(3, Math.round((Number(value || 0) / max) * 100))}%` }
function categoryWidth(value) { const max = Math.max(...(summary.value.top_categories || []).map(item => item.count), 1); return `${Math.max(4, Math.round((value / max) * 100))}%` }
onMounted(loadAll)
</script>
<style scoped>
.dashboard { display: flex; flex-direction: column; gap: 14px; color: #0f172a; }.page-header,.card,.runtime-status { border: 1px solid #e2e8f0; border-radius: 10px; background: #fff; }.page-header { display: flex; align-items: center; justify-content: space-between; gap: 20px; padding: 22px 24px; }.eyebrow { margin: 0 0 4px; color: #64748b; font: 700 10px/1.4 ui-monospace,monospace; letter-spacing: .08em; }.page-header h1,.card h2 { margin: 0; font-size: 18px; letter-spacing: -.01em; }.subtitle,.card-head p { margin: 5px 0 0; color: #64748b; font-size: 12px; }.header-actions { display: flex; gap: 8px; }.header-actions select,.primary { height: 34px; border-radius: 7px; padding: 0 10px; font: inherit; font-size: 12px; }.header-actions select { border: 1px solid #dbe3ee; background: #fff; color: #334155; }.primary { border: 1px solid #2563eb; background: #2563eb; color: #fff; font-weight: 700; cursor: pointer; }.primary:disabled { opacity: .6; }.runtime-status { display: flex; align-items: center; gap: 10px; min-height: 50px; padding: 10px 14px; }.runtime-status.warning { border-color: #fde68a; background: #fffbeb; }.status-dot { width: 7px; height: 7px; border-radius: 50%; background: #22c55e; }.warning .status-dot { background: #d97706; }.runtime-status div { display: flex; gap: 6px; align-items: baseline; min-width: 0; }.runtime-status strong { font-size: 12px; }.runtime-status span { color: #64748b; font-size: 12px; }.text-btn { margin-left: auto; padding: 4px; border: 0; background: none; color: #2563eb; cursor: pointer; font-size: 12px; }.runtime-detail { margin-top: -8px; padding: 10px 14px; border: 1px solid #fde68a; border-radius: 8px; background: #fff; color: #92400e; font-size: 12px; }.runtime-detail code { word-break: break-word; }.metrics { display: grid; grid-template-columns: repeat(4,1fr); gap: 12px; }.metric { min-height: 96px; padding: 16px; border: 1px solid #e2e8f0; border-radius: 10px; background: #fff; }.metric span,.metric small { display: block; color: #64748b; font-size: 12px; }.metric strong { display: block; margin: 8px 0 5px; font-size: 25px; line-height: 1; letter-spacing: -.04em; }.card { overflow: hidden; }.card-head { display: flex; justify-content: space-between; align-items: flex-start; gap: 16px; padding: 16px 18px; border-bottom: 1px solid #eef2f7; }.card-head h2 { font-size: 14px; }.card-head>a { color: #2563eb; font-size: 12px; font-weight: 700; text-decoration: none; }.range-label { color: #64748b; font: 11px ui-monospace,monospace; }.chart { display: flex; align-items: end; gap: 8px; min-height: 202px; padding: 18px 18px 0; overflow-x: auto; background-image: linear-gradient(to bottom, #f1f5f9 1px, transparent 1px); background-size: 100% 42px; }.chart.sparse { justify-content: center; }.chart.sparse .chart-point { flex: 0 0 64px; }.chart-point { display: flex; flex: 1; flex-direction: column; justify-content: end; min-width: 24px; height: 184px; gap: 7px; }.bars { display: flex; align-items: end; gap: 2px; height: 150px; border-bottom: 1px solid #cbd5e1; }.bars i { display: block; flex: 1; min-height: 2px; border-radius: 2px 2px 0 0; }.request,.legend .request { background: #2563eb; }.blocked,.legend .blocked { background: #dc2626; }.risk,.legend .risk { background: #d97706; }.chart-point small { overflow: hidden; color: #94a3b8; font-size: 9px; text-align: center; text-overflow: ellipsis; white-space: nowrap; }.legend { display: flex; gap: 14px; padding: 10px 18px 14px; color: #64748b; font-size: 11px; }.legend span { display: flex; align-items: center; gap: 5px; }.legend i { width: 7px; height: 7px; border-radius: 2px; }.chart-empty,.compact-empty { color: #64748b; font-size: 12px; }.chart-empty { display: flex; min-height: 202px; flex-direction: column; align-items: center; justify-content: center; gap: 5px; }.chart-empty strong { color: #334155; }.compact-empty { padding: 30px 18px; }.two-col { display: grid; grid-template-columns: 1fr 1fr; gap: 14px; }.category-list { padding: 7px 18px; }.category-row { display: grid; grid-template-columns: 120px 1fr 30px; align-items: center; gap: 10px; min-height: 42px; font-size: 12px; }.category-row>div { height: 5px; overflow: hidden; border-radius: 4px; background: #f1f5f9; }.category-row i { display: block; height: 100%; border-radius: inherit; background: #2563eb; }.category-row strong { text-align: right; }.action-list { margin: 0; padding: 7px 18px; }.action-list div { display: grid; grid-template-columns: 100px 1fr; gap: 8px; min-height: 42px; align-items: center; border-bottom: 1px solid #f1f5f9; }.action-list div:last-child { border-bottom: 0; }.action-list dt { color: #475569; font: 700 10px ui-monospace,monospace; }.action-list dd { margin: 0; color: #64748b; font-size: 12px; }.table-wrap { overflow: auto; }table { width: 100%; min-width: 720px; border-collapse: collapse; }th,td { padding: 12px 16px; border-bottom: 1px solid #eef2f7; text-align: left; font-size: 12px; }th { background: #f8fafc; color: #64748b; font-size: 10px; font-weight: 700; }tbody tr:hover { background: #f8fafc; }.mono { font-family: ui-monospace,monospace; }.path { color: #64748b; }.action-pill { display: inline-block; border-radius: 4px; padding: 3px 6px; background: #f1f5f9; color: #475569; font: 700 10px ui-monospace,monospace; text-transform: uppercase; }.action-pill.block { background: #fef2f2; color: #dc2626; }.action-pill.rate-limit { background: #fffbeb; color: #a16207; }.action-pill.log { background: #eff6ff; color: #1d4ed8; }@media(max-width:860px) { .metrics { grid-template-columns: repeat(2,1fr); }.two-col { grid-template-columns: 1fr; } }@media(max-width:600px) { .page-header { align-items: flex-start; flex-direction: column; padding: 18px; }.header-actions { width: 100%; }.header-actions>* { flex: 1; }.metrics { grid-template-columns: 1fr; }.runtime-status { align-items: flex-start; }.runtime-status div { flex-direction: column; gap: 2px; }.text-btn { margin-left: 0; }.chart.sparse { justify-content: flex-start; } }
</style>
