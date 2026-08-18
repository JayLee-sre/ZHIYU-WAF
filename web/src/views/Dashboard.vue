<template>
  <div class="page dashboard-page">
    <section class="workspace-hero">
      <div class="hero-copy">
        <div class="hero-kicker"><i></i>SECURITY WORKSPACE</div>
        <h1>安全态势工作台</h1>
        <p>集中查看本地风险决策、拦截动作与关键安全事件。</p>
      </div>
      <div class="hero-actions">
        <div class="sync-meta"><span>最近同步</span><strong>{{ lastSynced }}</strong></div>
        <select v-model="range" class="range-select" aria-label="选择统计时间范围" @change="loadAll">
          <option value="1h">最近 1 小时</option><option value="6h">最近 6 小时</option><option value="24h">最近 24 小时</option><option value="7d">最近 7 天</option><option value="30d">最近 30 天</option>
        </select>
        <button class="btn-ghost refresh-button" type="button" :disabled="loading" @click="loadAll"><svg viewBox="0 0 24 24" :class="{ spinning: loading }"><path d="M20 11a8 8 0 10-2.34 5.66"/><path d="M20 4v7h-7"/></svg>刷新</button>
        <router-link to="/soc-dashboard" class="btn-primary soc-link"><svg viewBox="0 0 24 24"><path d="M4 19V5m5 14v-8m5 8V4m5 15v-5"/></svg>监控大屏</router-link>
      </div>
    </section>

    <section class="runtime-banner" :class="{ degraded: system.firewall?.degraded }">
      <div class="runtime-icon"><svg viewBox="0 0 24 24"><path d="M12 3l7 3.2v5.1c0 4.4-2.9 7.7-7 9.7-4.1-2-7-5.3-7-9.7V6.2L12 3z"/><path d="M9 12l2 2 4-4"/></svg></div>
      <div class="runtime-copy"><strong>{{ firewallTitle }}</strong><span>{{ firewallSummary }}</span></div>
      <div class="runtime-status"><i></i>{{ system.firewall?.degraded ? '降级保护' : '运行中' }}</div>
      <button v-if="system.firewall?.degraded && system.firewall?.message" type="button" class="detail-button" @click="showFirewallDetail = !showFirewallDetail">{{ showFirewallDetail ? '收起详情' : '查看详情' }}</button>
    </section>
    <div v-if="showFirewallDetail && system.firewall?.message" class="runtime-detail"><code>{{ system.firewall.message }}</code></div>

    <section class="summary-grid" aria-label="安全核心指标">
      <article v-for="card in cards" :key="card.label" class="summary-card" :class="card.tone">
        <div class="summary-icon"><svg viewBox="0 0 24 24"><path :d="card.icon"/></svg></div>
        <div><span>{{ card.label }}</span><strong>{{ formatNumber(card.value) }}</strong><small>{{ card.note }}</small></div>
      </article>
    </section>

    <section class="panel trend-panel">
      <header class="panel-header">
        <div><span class="section-kicker">RISK TELEMETRY</span><h2>风险趋势</h2><p>按时间聚合的请求、拦截与高风险事件。</p></div>
        <div class="chart-legend"><span><i class="request"></i>请求</span><span><i class="blocked"></i>拦截</span><span><i class="high-risk"></i>高风险</span></div>
      </header>
      <div class="chart-stage">
        <div ref="trendRef" class="trend-chart" aria-label="风险趋势折线图"></div>
        <div v-if="!series.length" class="chart-empty"><div class="empty-wave"><i></i><i></i><i></i></div><strong>暂无趋势数据</strong><span>防护链收到请求后，会在这里展示风险变化。</span></div>
      </div>
    </section>

    <section class="insight-grid">
      <section class="panel category-panel">
        <header class="panel-header compact"><div><span class="section-kicker">THREAT EVIDENCE</span><h2>威胁类别</h2><p>按主要检测证据聚合。</p></div><span class="panel-badge">TOP {{ summary.top_categories?.length || 0 }}</span></header>
        <div v-if="summary.top_categories?.length" class="category-list">
          <div v-for="(item, index) in summary.top_categories" :key="item.category" class="category-row">
            <span class="category-rank">{{ String(index + 1).padStart(2, '0') }}</span>
            <strong>{{ item.category }}</strong>
            <div class="category-track"><i :style="{ width: categoryWidth(item.count) }"></i></div>
            <b>{{ formatNumber(item.count) }}</b>
          </div>
        </div>
        <div v-else class="compact-empty">暂无分类命中。</div>
      </section>

      <section class="panel action-panel">
        <header class="panel-header compact"><div><span class="section-kicker">DECISION CHAIN</span><h2>风险动作</h2><p>每个请求只输出一次最终动作。</p></div></header>
        <div class="action-list">
          <div v-for="item in actionItems" :key="item.code"><i :class="item.tone"><svg viewBox="0 0 24 24"><path :d="item.icon"/></svg></i><div><strong>{{ item.code }}</strong><span>{{ item.description }}</span></div><b :class="item.tone">{{ item.state }}</b></div>
        </div>
      </section>
    </section>

    <section class="panel events-panel">
      <header class="panel-header">
        <div><span class="section-kicker">EVENT STREAM</span><h2>最近安全事件</h2><p>仅保存定位和审计所需的信息。</p></div>
        <router-link to="/logs" class="view-all">查看全部<svg viewBox="0 0 24 24"><path d="M5 12h14m-6-6l6 6-6 6"/></svg></router-link>
      </header>
      <div v-if="summary.recent_events?.length" class="table-wrap">
        <table class="events-table">
          <thead><tr><th>时间</th><th>来源</th><th>请求</th><th>检测证据</th><th>风险</th><th>最终动作</th></tr></thead>
          <tbody><tr v-for="event in summary.recent_events" :key="event.id"><td class="event-time">{{ formatTime(event.created_at) }}</td><td><code>{{ event.client_ip }}</code></td><td class="request-cell"><b>{{ event.method }}</b><span :title="event.path">{{ event.path }}</span></td><td>{{ event.rule_id || event.category || '-' }}</td><td><span class="risk-score" :class="riskTone(event.risk_score)">{{ event.risk_score }}</span></td><td><span class="action-pill" :class="actionClass(event.action)">{{ event.action || 'allow' }}</span></td></tr></tbody>
        </table>
      </div>
      <div v-else class="events-empty"><svg viewBox="0 0 24 24"><path d="M12 3l7 3.2v5.1c0 4.4-2.9 7.7-7 9.7-4.1-2-7-5.3-7-9.7V6.2L12 3z"/><path d="M9 12l2 2 4-4"/></svg><strong>当前没有待关注的安全事件</strong><span>风险事件进入防护链后会显示在此处。</span></div>
    </section>
  </div>
</template>

<script setup>
import { computed, nextTick, onBeforeUnmount, onMounted, ref } from 'vue'
import * as echarts from 'echarts'
import api from '../api'

const range = ref('24h')
const loading = ref(false)
const summary = ref({ top_categories: [], recent_events: [] })
const series = ref([])
const system = ref({ firewall: {} })
const showFirewallDetail = ref(false)
const lastSynced = ref('等待同步')
const trendRef = ref(null)
let trendChart
let resizeObserver

const cards = computed(() => [
  { label: '请求总数', value: summary.value.request_count || 0, note: '进入 V2 防护链', tone: 'blue', icon: 'M4 18V6m5 12V10m5 8V4m5 14V8' },
  { label: '已拦截', value: summary.value.blocked_count || 0, note: '最终动作：BLOCK', tone: 'rose', icon: 'M12 3l8 4v5c0 4.8-3.2 7.9-8 9-4.8-1.1-8-4.2-8-9V7l8-4zM9 12h6' },
  { label: '攻击来源', value: summary.value.attack_ip_count || 0, note: '风险分数 ≥ 30', tone: 'violet', icon: 'M20 10c0 5-8 11-8 11S4 15 4 10a8 8 0 1116 0zM12 12.5a2.5 2.5 0 100-5 2.5 2.5 0 000 5z' },
  { label: '高风险事件', value: summary.value.high_risk_count || 0, note: '风险分数 ≥ 60', tone: 'amber', icon: 'M12 8v4m0 4h.01M5.1 19h13.8c1.5 0 2.5-1.6 1.7-2.9L13.7 4.3c-.8-1.4-2.8-1.4-3.6 0L3.4 16.1C2.6 17.4 3.6 19 5.1 19z' },
])
const firewallTitle = computed(() => system.value.firewall?.available ? 'nftables 防火墙已连接' : '应用层防护正在运行')
const firewallSummary = computed(() => system.value.firewall?.available ? '内核封禁集合可用，应用层风险动作与内核规则协同执行。' : system.value.firewall?.degraded ? '内核封禁暂不可用，系统已安全降级为应用层阻断。' : '本地规则与风险引擎正在持续评估每一次请求。')
const actionItems = [
  { code: 'ALLOW', description: '低风险请求正常回源', state: '监测', tone: 'slate', icon: 'M5 12l4 4L19 6' },
  { code: 'LOG', description: '保留可追溯风险证据', state: '记录', tone: 'blue', icon: 'M12 4v16m-6-8h12' },
  { code: 'RATE LIMIT', description: '限制异常访问频率', state: '限制', tone: 'amber', icon: 'M5 12h14M12 5v14' },
  { code: 'BLOCK', description: '阻断高风险恶意请求', state: '拦截', tone: 'rose', icon: 'M12 3l7 3.2v5.1c0 4.4-2.9 7.7-7 9.7-4.1-2-7-5.3-7-9.7V6.2L12 3z' },
]

function unwrap(response) { return response?.data ?? response ?? {} }
function formatNumber(value) { return Number(value || 0).toLocaleString('zh-CN') }
function formatTime(value) { return value ? new Date(value).toLocaleString('zh-CN', { hour12: false }) : '-' }
function formatBucket(value) { return String(value || '').match(/(\d{2}:\d{2})/)?.[1] || String(value || '').slice(-5) }
function categoryWidth(value) { const max = Math.max(...(summary.value.top_categories || []).map(item => Number(item.count || 0)), 1); return `${Math.max(5, Math.round(Number(value || 0) / max * 100))}%` }
function riskTone(score) { const value = Number(score || 0); return value >= 60 ? 'critical' : value >= 30 ? 'medium' : 'low' }
function actionClass(action) { return String(action || 'allow').toLowerCase().replace(/_/g, '-') }
function initChart() { if (!trendRef.value || trendChart) return; trendChart = echarts.init(trendRef.value); updateChart() }
function updateChart() {
  if (!trendChart) return
  const labels = series.value.map(item => formatBucket(item.timestamp))
  trendChart.setOption({ animationDuration: 500, animationEasing: 'cubicOut', tooltip: { trigger: 'axis', backgroundColor: '#0f172a', borderColor: '#334155', borderWidth: 1, textStyle: { color: '#eff6ff', fontSize: 12 }, padding: [9, 11], extraCssText: 'border-radius:8px;box-shadow:0 16px 32px rgba(15,23,42,.18)', formatter: values => { const point = series.value[values[0]?.dataIndex] || {}; return `<b>${point.timestamp || ''}</b><br/>请求 <b>${formatNumber(point.requests)}</b><br/>拦截 <b>${formatNumber(point.blocked)}</b><br/>高风险 <b>${formatNumber(point.high_risk)}</b>` } }, grid: { top: 24, right: 25, bottom: 28, left: 42 }, xAxis: { type: 'category', boundaryGap: false, data: labels, axisTick: { show: false }, axisLine: { lineStyle: { color: '#dbe4ef' } }, axisLabel: { color: '#94a3b8', fontSize: 10, interval: Math.max(0, Math.floor(labels.length / 6) - 1) } }, yAxis: { type: 'value', minInterval: 1, splitNumber: 4, axisLine: { show: false }, axisTick: { show: false }, axisLabel: { color: '#94a3b8', fontSize: 10 }, splitLine: { lineStyle: { color: '#edf2f7', type: 'dashed' } } }, series: [{ name: '请求', type: 'line', smooth: .32, showSymbol: false, data: series.value.map(item => item.requests || 0), lineStyle: { color: '#2563eb', width: 2.5 }, areaStyle: { color: new echarts.graphic.LinearGradient(0, 0, 0, 1, [{ offset: 0, color: 'rgba(37,99,235,.24)' }, { offset: 1, color: 'rgba(37,99,235,.01)' }]) } }, { name: '拦截', type: 'line', smooth: .32, showSymbol: false, data: series.value.map(item => item.blocked || 0), lineStyle: { color: '#e11d48', width: 2 } }, { name: '高风险', type: 'line', smooth: .32, showSymbol: false, data: series.value.map(item => item.high_risk || 0), lineStyle: { color: '#d97706', width: 1.5, type: 'dashed' } }] }, true)
}
async function loadAll() {
  if (loading.value) return
  loading.value = true
  try {
    const [summaryResult, seriesResult, systemResult] = await Promise.all([api.get('/dashboard/summary', { params: { range: range.value } }), api.get('/dashboard/timeseries', { params: { range: range.value } }), api.get('/system/status', { suppressError: true })])
    summary.value = unwrap(summaryResult); const nextSeries = unwrap(seriesResult); series.value = Array.isArray(nextSeries) ? nextSeries : []; system.value = unwrap(systemResult); showFirewallDetail.value = false; lastSynced.value = new Date().toLocaleTimeString('zh-CN', { hour12: false }); await nextTick(); initChart(); updateChart()
  } finally { loading.value = false }
}
onMounted(async () => { await loadAll(); await nextTick(); initChart(); resizeObserver = new ResizeObserver(() => trendChart?.resize()); if (trendRef.value) resizeObserver.observe(trendRef.value) })
onBeforeUnmount(() => { resizeObserver?.disconnect(); trendChart?.dispose() })
</script>

<style scoped>
.dashboard-page{gap:16px}.workspace-hero{display:flex;align-items:center;justify-content:space-between;gap:18px;padding:22px 24px;border:1px solid #dce5f0;border-radius:12px;background:radial-gradient(circle at 85% -75%,rgba(37,99,235,.12),transparent 43%),linear-gradient(135deg,#fff,#f8fbff);box-shadow:0 8px 24px rgba(15,23,42,.035)}.hero-kicker,.section-kicker{display:flex;align-items:center;gap:6px;color:#2563eb;font:800 10px ui-monospace,SFMono-Regular,Menlo,monospace;letter-spacing:.1em}.hero-kicker i{width:6px;height:6px;border-radius:50%;background:#2563eb;box-shadow:0 0 0 4px rgba(37,99,235,.1)}.hero-copy h1{margin:7px 0 4px;color:#0f172a;font-size:22px;letter-spacing:-.03em}.hero-copy p,.panel-header p{margin:0;color:#64748b;font-size:12.5px}.hero-actions{display:flex;align-items:center;gap:8px}.sync-meta{display:flex;flex-direction:column;align-items:flex-end;margin-right:2px}.sync-meta span{color:#94a3b8;font-size:10px}.sync-meta strong{margin-top:1px;color:#475569;font:700 11px ui-monospace,monospace}.range-select{height:36px;padding:0 28px 0 11px;border:1px solid #d8e2ee;border-radius:7px;background:#fff;color:#334155;font:600 12px inherit;outline:none}.range-select:focus{border-color:#2563eb;box-shadow:0 0 0 3px rgba(37,99,235,.1)}.refresh-button{height:36px;padding:0 11px}.refresh-button svg,.soc-link svg,.view-all svg{width:15px;height:15px;fill:none;stroke:currentColor;stroke-width:2;stroke-linecap:round;stroke-linejoin:round}.spinning{animation:spin .75s linear infinite}@keyframes spin{to{transform:rotate(360deg)}}.soc-link{height:36px;padding:0 13px;text-decoration:none}.runtime-banner{display:flex;align-items:center;gap:12px;padding:12px 15px;border:1px solid #bdebd8;border-radius:10px;background:linear-gradient(90deg,#f2fdf8,#fbfffd)}.runtime-banner.degraded{border-color:#f8d68e;background:linear-gradient(90deg,#fffbeb,#fffdf7)}.runtime-icon{display:grid;width:31px;height:31px;place-items:center;border-radius:8px;background:#dcfce7;color:#059669}.degraded .runtime-icon{background:#fef3c7;color:#b45309}.runtime-icon svg,.events-empty svg{width:17px;height:17px;fill:none;stroke:currentColor;stroke-width:1.9;stroke-linecap:round;stroke-linejoin:round}.runtime-copy{min-width:0}.runtime-copy strong,.runtime-copy span{display:block}.runtime-copy strong{color:#166534;font-size:12px}.degraded .runtime-copy strong{color:#92400e}.runtime-copy span{margin-top:2px;color:#64748b;font-size:11.5px}.runtime-status{display:flex;align-items:center;gap:5px;margin-left:auto;color:#15803d;font-size:11px;font-weight:800}.degraded .runtime-status{color:#b45309}.runtime-status i{width:6px;height:6px;border-radius:50%;background:currentColor;box-shadow:0 0 0 4px currentColor;opacity:.28}.detail-button{padding:4px 7px;border:0;border-radius:5px;background:transparent;color:#2563eb;font-size:11px;font-weight:700;cursor:pointer}.detail-button:hover{background:#eff6ff}.runtime-detail{padding:10px 14px;border:1px solid #f8d68e;border-radius:8px;background:#fffdf5;color:#92400e;font-size:12px}.runtime-detail code{word-break:break-word}.summary-grid{display:grid;grid-template-columns:repeat(4,1fr);gap:12px}.summary-card{position:relative;display:flex;min-height:104px;gap:12px;overflow:hidden;padding:16px;border:1px solid #e0e8f1;border-radius:10px;background:#fff;transition:border-color .2s,box-shadow .2s}.summary-card:before{content:"";position:absolute;top:0;left:0;width:3px;height:100%;background:var(--card-accent)}.summary-card:hover{border-color:#cbd8e8;box-shadow:0 10px 22px rgba(15,23,42,.06)}.summary-card.blue{--card-accent:#2563eb}.summary-card.rose{--card-accent:#e11d48}.summary-card.violet{--card-accent:#7c3aed}.summary-card.amber{--card-accent:#d97706}.summary-icon{display:grid;width:35px;height:35px;place-items:center;flex:0 0 auto;border-radius:9px}.blue .summary-icon{background:#eff6ff;color:#2563eb}.rose .summary-icon{background:#fff1f2;color:#e11d48}.violet .summary-icon{background:#f5f3ff;color:#7c3aed}.amber .summary-icon{background:#fffbeb;color:#d97706}.summary-icon svg{width:18px;height:18px;fill:none;stroke:currentColor;stroke-width:1.8;stroke-linecap:round;stroke-linejoin:round}.summary-card span,.summary-card small{display:block}.summary-card span{color:#64748b;font-size:11.5px;font-weight:700}.summary-card strong{display:block;margin:7px 0 3px;color:#0f172a;font-size:25px;line-height:1;letter-spacing:-.04em}.summary-card small{color:#94a3b8;font-size:10.5px}.panel{overflow:hidden;border:1px solid #dfe7f1;border-radius:10px;background:#fff;box-shadow:0 1px 2px rgba(15,23,42,.025)}.panel-header{display:flex;align-items:flex-start;justify-content:space-between;gap:16px;padding:16px 18px;border-bottom:1px solid #edf2f7;background:linear-gradient(180deg,#fff,#fbfdff)}.panel-header.compact{padding-top:14px;padding-bottom:13px}.section-kicker{font-size:9px}.panel-header h2{margin:5px 0 3px;color:#0f172a;font-size:15px;letter-spacing:-.01em}.chart-stage{position:relative;height:264px}.trend-chart{height:264px}.chart-legend{display:flex;gap:12px;padding-top:8px;color:#64748b;font-size:11px}.chart-legend span{display:flex;align-items:center;gap:5px}.chart-legend i{width:7px;height:7px;border-radius:50%;background:#2563eb}.chart-legend .blocked{background:#e11d48}.chart-legend .high-risk{background:#d97706}.chart-empty{position:absolute;inset:0;display:flex;flex-direction:column;align-items:center;justify-content:center;gap:5px;background:#fff;color:#64748b;font-size:12px}.chart-empty strong{margin-top:8px;color:#334155;font-size:13px}.empty-wave{display:flex;align-items:end;gap:3px;height:28px}.empty-wave i{width:4px;border-radius:9px;background:#cbd5e1}.empty-wave i:nth-child(1){height:11px}.empty-wave i:nth-child(2){height:25px}.empty-wave i:nth-child(3){height:16px}.insight-grid{display:grid;grid-template-columns:1.15fr .85fr;gap:16px}.panel-badge{margin-top:2px;padding:3px 6px;border-radius:4px;background:#eff6ff;color:#2563eb;font:800 9px ui-monospace,monospace}.category-list{padding:7px 18px 9px}.category-row{display:grid;grid-template-columns:25px minmax(92px,1fr) minmax(80px,1.6fr) 34px;align-items:center;gap:10px;min-height:40px;border-bottom:1px solid #f1f5f9}.category-row:last-child{border-bottom:0}.category-rank{color:#94a3b8;font:800 10px ui-monospace,monospace}.category-row strong{overflow:hidden;color:#334155;font-size:12px;text-overflow:ellipsis;white-space:nowrap}.category-track{height:5px;overflow:hidden;border-radius:99px;background:#eef2f7}.category-track i{display:block;height:100%;border-radius:inherit;background:linear-gradient(90deg,#2563eb,#60a5fa)}.category-row>b{text-align:right;color:#475569;font:700 11px ui-monospace,monospace}.compact-empty{display:grid;min-height:136px;place-items:center;color:#94a3b8;font-size:12px}.action-list{padding:3px 18px 7px}.action-list>div{display:grid;grid-template-columns:29px minmax(0,1fr) auto;align-items:center;gap:10px;min-height:45px;border-bottom:1px solid #f1f5f9}.action-list>div:last-child{border-bottom:0}.action-list>div>i{display:grid;width:27px;height:27px;place-items:center;border-radius:7px}.action-list>div>i svg{width:14px;height:14px;fill:none;stroke:currentColor;stroke-width:1.8;stroke-linecap:round;stroke-linejoin:round}.action-list .slate{background:#f1f5f9;color:#64748b}.action-list .blue{background:#eff6ff;color:#2563eb}.action-list .amber{background:#fffbeb;color:#b45309}.action-list .rose{background:#fff1f2;color:#e11d48}.action-list strong,.action-list span{display:block}.action-list strong{color:#475569;font:800 10px ui-monospace,monospace}.action-list span{margin-top:2px;color:#94a3b8;font-size:10.5px}.action-list>b{padding:3px 5px;border-radius:4px;font-size:9px}.action-list>b.slate{background:#f1f5f9;color:#64748b}.action-list>b.blue{background:#eff6ff;color:#2563eb}.action-list>b.amber{background:#fffbeb;color:#b45309}.action-list>b.rose{background:#fff1f2;color:#e11d48}.view-all{display:flex;align-items:center;gap:4px;margin-top:7px;color:#2563eb;font-size:12px;font-weight:800;text-decoration:none}.view-all svg{width:14px;height:14px}.table-wrap{overflow:auto}.events-table{width:100%;min-width:760px;border-collapse:collapse}.events-table th,.events-table td{padding:12px 17px;border-bottom:1px solid #edf2f7;text-align:left;font-size:12px}.events-table th{color:#64748b;background:#f8fafc;font-size:10px;font-weight:800;letter-spacing:.02em}.events-table tbody tr:hover{background:#fafcff}.events-table tbody tr:last-child td{border-bottom:0}.event-time{color:#64748b;white-space:nowrap}.events-table code{padding:3px 5px;border-radius:4px;background:#f1f5f9;color:#475569;font:10.5px ui-monospace,monospace}.request-cell{max-width:240px}.request-cell b{margin-right:6px;color:#2563eb;font:800 10px ui-monospace,monospace}.request-cell span{overflow:hidden;color:#475569;text-overflow:ellipsis;white-space:nowrap}.risk-score{display:inline-grid;min-width:28px;place-items:center;padding:3px 5px;border-radius:4px;font:800 10px ui-monospace,monospace}.risk-score.low{background:#ecfdf5;color:#059669}.risk-score.medium{background:#fffbeb;color:#a16207}.risk-score.critical{background:#fff1f2;color:#e11d48}.action-pill{display:inline-block;padding:3px 6px;border-radius:4px;background:#f1f5f9;color:#64748b;font:800 10px ui-monospace,monospace;text-transform:uppercase}.action-pill.block{background:#fff1f2;color:#e11d48}.action-pill.rate-limit{background:#fffbeb;color:#a16207}.action-pill.log{background:#eff6ff;color:#2563eb}.events-empty{display:flex;min-height:182px;flex-direction:column;align-items:center;justify-content:center;gap:6px;color:#94a3b8;font-size:12px}.events-empty svg{width:30px;height:30px;margin-bottom:4px;color:#86efac}.events-empty strong{color:#475569;font-size:13px}@media(max-width:1024px){.workspace-hero{align-items:flex-start;flex-direction:column}.hero-actions{width:100%}.summary-grid{grid-template-columns:repeat(2,1fr)}}@media(max-width:760px){.workspace-hero{padding:18px}.hero-actions{flex-wrap:wrap}.sync-meta{display:none}.range-select{flex:1}.soc-link{flex:1;justify-content:center}.runtime-banner{align-items:flex-start}.runtime-status{display:none}.summary-grid,.insight-grid{grid-template-columns:1fr}.chart-stage,.trend-chart{height:230px}.chart-legend{gap:8px;font-size:10px}.panel-header{padding:14px}.events-table th,.events-table td{padding:11px 14px}}@media(max-width:480px){.refresh-button{padding:0 9px}.refresh-button svg{margin:0}.refresh-button{font-size:0}.summary-card{min-height:91px}.category-row{grid-template-columns:23px minmax(80px,1fr) 1fr 28px;gap:7px}.action-list{padding-left:14px;padding-right:14px}}
</style>
