<template>
  <div class="v2-dashboard">
    <header class="hero">
      <div>
        <p class="eyebrow">ZHIYU-WAF V2 · LOCAL-FIRST</p>
        <h1>安全态势</h1>
        <p>以统一风险决策为中心，所有功能免费、所有关键动作可追溯。</p>
      </div>
      <div class="actions">
        <select v-model="range" @change="loadAll">
          <option value="1h">最近 1 小时</option>
          <option value="6h">最近 6 小时</option>
          <option value="24h">最近 24 小时</option>
          <option value="7d">最近 7 天</option>
          <option value="30d">最近 30 天</option>
        </select>
        <button class="primary" :disabled="loading" @click="loadAll">{{ loading ? '刷新中…' : '刷新数据' }}</button>
      </div>
    </header>

    <div class="system-strip" :class="{ degraded: system.firewall?.degraded }">
      <span class="status-dot"></span>
      <span class="system-copy"><strong>{{ system.firewall?.available ? 'nftables 防火墙已连接' : '应用层防护运行中' }}</strong><small>{{ firewallSummary }}</small></span>
      <button v-if="system.firewall?.degraded" class="detail-toggle" type="button" @click="showFirewallDetail = !showFirewallDetail">{{ showFirewallDetail ? '收起详情' : '查看详情' }}</button>
      <span class="free-badge">全功能免费</span>
    </div>
    <div v-if="showFirewallDetail && system.firewall?.message" class="firewall-detail"><span>内核防火墙详情</span><code>{{ system.firewall.message }}</code></div>

    <section class="metric-grid">
      <article v-for="card in cards" :key="card.label" class="metric-card" :class="card.tone">
        <span>{{ card.label }}</span>
        <strong>{{ formatNumber(card.value) }}</strong>
        <small>{{ card.note }}</small>
      </article>
    </section>

    <section class="panel trend-panel">
      <div class="panel-head"><div><h2>风险趋势</h2><p>请求、拦截与高风险事件按时间聚合。</p></div><span>{{ range }}</span></div>
      <div v-if="series.length" class="trend-bars" :class="{ sparse: series.length <= 4 }">
        <div v-for="point in series" :key="point.timestamp" class="trend-point" :title="`${point.timestamp}: ${point.requests} 请求 / ${point.blocked} 拦截`">
          <div class="bars">
            <i class="request" :style="{ height: barHeight(point.requests) }"></i>
            <i class="blocked" :style="{ height: barHeight(point.blocked) }"></i>
            <i class="risk" :style="{ height: barHeight(point.high_risk) }"></i>
          </div>
          <small>{{ shortTime(point.timestamp) }}</small>
        </div>
      </div>
      <div v-else class="empty chart-empty"><span>暂无趋势数据</span><p>当请求通过防护链后，这里会按时间展示请求、拦截与高风险事件。</p></div>
      <div class="legend"><span><i class="request"></i>请求</span><span><i class="blocked"></i>拦截</span><span><i class="risk"></i>高风险</span></div>
    </section>

    <div class="two-col">
      <section class="panel">
        <div class="panel-head"><div><h2>威胁类别</h2><p>按主要检测证据聚合。</p></div></div>
        <div v-if="summary.top_categories?.length" class="category-list">
          <div v-for="item in summary.top_categories" :key="item.category" class="category-row">
            <span>{{ item.category }}</span><div><i :style="{ width: categoryWidth(item.count) }"></i></div><strong>{{ item.count }}</strong>
          </div>
        </div>
        <div v-else class="empty">暂无分类命中。</div>
      </section>
      <section class="panel action-panel">
        <div class="panel-head"><div><h2>风险动作说明</h2><p>同一请求只由风险引擎输出一次最终动作。</p></div></div>
        <dl>
          <div><dt class="allow">ALLOW</dt><dd>正常回源，保留最小可观测记录。</dd></div>
          <div><dt class="log">LOG</dt><dd>记录风险证据，不打断访问。</dd></div>
          <div><dt class="limit">RATE LIMIT</dt><dd>返回 429，限制异常频率。</dd></div>
          <div><dt class="block">BLOCK</dt><dd>应用层阻断；高风险来源可同步 nftables。</dd></div>
        </dl>
      </section>
    </div>

    <section class="panel events-panel">
      <div class="panel-head"><div><h2>最近安全事件</h2><p>事件不存储完整请求体；仅保留定位与审计所需字段。</p></div><router-link to="/logs">查看全部事件</router-link></div>
      <div class="table-wrap" v-if="summary.recent_events?.length">
        <table><thead><tr><th>时间</th><th>来源</th><th>请求</th><th>证据</th><th>风险</th><th>动作</th></tr></thead>
          <tbody><tr v-for="event in summary.recent_events" :key="event.id"><td>{{ formatTime(event.created_at) }}</td><td class="mono">{{ event.client_ip }}</td><td><b>{{ event.method }}</b> <span class="path">{{ event.path }}</span></td><td>{{ event.rule_id || event.category || '-' }}</td><td><span class="risk-score">{{ event.risk_score }}</span></td><td><span class="action" :class="String(event.action || 'allow').replace('_', '-')">{{ event.action || 'allow' }}</span></td></tr></tbody>
        </table>
      </div>
      <div v-else class="empty">暂无安全事件。</div>
    </section>
  </div>
</template>

<script setup>
import { computed, onMounted, ref } from 'vue'
import api from '../api'

const range = ref('24h')
const loading = ref(false)
const summary = ref({ top_categories: [], recent_events: [] })
const series = ref([])
const system = ref({ firewall: {} })
const showFirewallDetail = ref(false)
const firewallSummary = computed(() => {
  if (system.value.firewall?.available) return '内核封禁集合已就绪，短期恶意来源可同步至 nftables。'
  if (system.value.firewall?.degraded) return '内核封禁暂不可用，系统已自动降级为应用层阻断。'
  return '本地规则、风险引擎与审计存储均在应用层持续运行。'
})

const cards = computed(() => [
  { label: '请求总数', value: summary.value.request_count || 0, note: '进入 V2 防护链', tone: 'blue' },
  { label: '已拦截', value: summary.value.blocked_count || 0, note: '最终动作：BLOCK', tone: 'red' },
  { label: '攻击来源', value: summary.value.attack_ip_count || 0, note: '风险分数 ≥ 30 的来源', tone: 'amber' },
  { label: '高风险事件', value: summary.value.high_risk_count || 0, note: '风险分数 ≥ 60', tone: 'purple' },
])

function unwrap(response) { return response?.data ?? response ?? {} }
async function loadAll() {
  loading.value = true
  try {
    const [nextSummary, nextSeries, nextSystem] = await Promise.all([
      api.get('/dashboard/summary', { params: { range: range.value } }),
      api.get('/dashboard/timeseries', { params: { range: range.value } }),
      api.get('/system/status', { suppressError: true }),
    ])
    summary.value = unwrap(nextSummary)
    series.value = unwrap(nextSeries)
    system.value = unwrap(nextSystem)
    showFirewallDetail.value = false
  } finally { loading.value = false }
}
function formatNumber(value) { return Number(value || 0).toLocaleString('zh-CN') }
function formatTime(value) { return value ? new Date(value).toLocaleString('zh-CN', { hour12: false }) : '-' }
function shortTime(value) { return value ? String(value).slice(5, 16) : '' }
function barHeight(value) { const max = Math.max(...series.value.flatMap(p => [p.requests || 0, p.blocked || 0, p.high_risk || 0]), 1); return `${Math.max(4, Math.round((Number(value || 0) / max) * 100))}%` }
function categoryWidth(value) { const max = Math.max(...(summary.value.top_categories || []).map(item => item.count), 1); return `${Math.max(5, Math.round((value / max) * 100))}%` }
onMounted(loadAll)
</script>

<style scoped>
.v2-dashboard{display:flex;flex-direction:column;gap:16px;color:#e5edf8}.hero{display:flex;justify-content:space-between;gap:24px;align-items:center;padding:26px 28px;border:1px solid #20324d;border-radius:18px;background:radial-gradient(circle at 80% 0,#183f61 0,#101c2e 42%,#0b1320 100%)}.eyebrow{font:700 11px/1.3 ui-monospace,monospace;letter-spacing:.14em;color:#67e8f9;margin:0 0 8px}.hero h1{margin:0;font-size:28px}.hero p{margin:8px 0 0;color:#a8b8cf}.actions{display:flex;gap:10px}.actions select,.actions button{height:38px;border-radius:9px;padding:0 12px;border:1px solid #355170;background:#122239;color:#e5edf8}.actions .primary{border:0;background:#22c7a0;color:#05251f;font-weight:800;cursor:pointer}.actions button:disabled{opacity:.6}.system-strip{display:flex;align-items:center;gap:9px;padding:12px 16px;background:#0d2c2a;border:1px solid #1d5b53;border-radius:11px;color:#bdf5e7;font-size:13px}.system-strip.degraded{background:#352518;border-color:#6b4926;color:#ffdc9a}.status-dot{width:8px;height:8px;border-radius:99px;background:currentColor;box-shadow:0 0 12px currentColor}.free-badge{margin-left:auto;padding:3px 8px;border:1px solid currentColor;border-radius:99px;font-size:11px;font-weight:800}.metric-grid{display:grid;grid-template-columns:repeat(4,1fr);gap:14px}.metric-card{padding:18px;border-radius:14px;border:1px solid #253956;background:#101c2e;display:flex;flex-direction:column;gap:8px}.metric-card span,.metric-card small{color:#93a7c0;font-size:12px}.metric-card strong{font-size:28px;line-height:1}.metric-card.blue strong{color:#67e8f9}.metric-card.red strong{color:#fb7185}.metric-card.amber strong{color:#fbbf24}.metric-card.purple strong{color:#c4b5fd}.panel{border:1px solid #243852;border-radius:14px;background:#101c2e;overflow:hidden}.panel-head{display:flex;justify-content:space-between;align-items:start;padding:17px 18px;border-bottom:1px solid #20324b}.panel-head h2{margin:0;font-size:15px}.panel-head p{margin:5px 0 0;color:#8ca1bc;font-size:12px}.panel-head>a{color:#67e8f9;font-size:12px;text-decoration:none}.trend-bars{height:215px;display:flex;gap:6px;padding:20px 18px 4px;align-items:end;overflow-x:auto}.trend-point{min-width:22px;flex:1;height:100%;display:flex;flex-direction:column;justify-content:end;gap:7px}.bars{height:170px;display:flex;align-items:end;gap:2px}.bars i{display:block;flex:1;min-height:3px;border-radius:3px 3px 0 0}.request,.legend .request{background:#38bdf8}.blocked,.legend .blocked{background:#fb7185}.risk,.legend .risk{background:#fbbf24}.trend-point small{color:#7186a1;font-size:9px;white-space:nowrap;transform:rotate(-35deg);transform-origin:left}.legend{display:flex;gap:16px;padding:10px 18px 14px;color:#9bb0ca;font-size:12px}.legend span{display:flex;gap:6px;align-items:center}.legend i{width:8px;height:8px;border-radius:2px}.two-col{display:grid;grid-template-columns:1fr 1fr;gap:16px}.category-list{padding:8px 18px}.category-row{display:grid;grid-template-columns:120px 1fr 42px;gap:10px;align-items:center;padding:10px 0;color:#cbd8e9;font-size:13px}.category-row>div{height:7px;border-radius:99px;background:#1d2c42;overflow:hidden}.category-row i{display:block;height:100%;border-radius:inherit;background:linear-gradient(90deg,#38bdf8,#8b5cf6)}.category-row strong{text-align:right}.action-panel dl{margin:0;padding:8px 18px}.action-panel dl>div{display:grid;grid-template-columns:96px 1fr;gap:12px;padding:10px 0;border-bottom:1px solid #1e3048}.action-panel dl>div:last-child{border-bottom:0}.action-panel dt{font:800 11px ui-monospace,monospace}.action-panel dd{margin:0;color:#9db0c9;font-size:12px}.allow{color:#5eead4}.log{color:#67e8f9}.limit{color:#fbbf24}.block{color:#fb7185}.table-wrap{overflow:auto}table{width:100%;border-collapse:collapse;min-width:720px}th,td{text-align:left;padding:12px 16px;border-bottom:1px solid #1d2e45;font-size:12px}th{color:#8da1ba;background:#0d1829;font-weight:700}td{color:#c7d5e8}.mono{font-family:ui-monospace,SFMono-Regular,Menlo,monospace}.path{color:#8fa4bd}.risk-score{color:#fbbf24;font-weight:800}.action{padding:3px 7px;border-radius:6px;font:700 10px ui-monospace,monospace;text-transform:uppercase}.action.block{background:#4a1e2b;color:#fda4af}.action.rate-limit{background:#4b3a16;color:#fde68a}.action.log{background:#12384b;color:#7dd3fc}.action.allow{background:#113a35;color:#99f6e4}.empty{padding:36px 18px;text-align:center;color:#7086a2;font-size:13px}@media(max-width:900px){.metric-grid{grid-template-columns:repeat(2,1fr)}.two-col{grid-template-columns:1fr}.hero{align-items:flex-start;flex-direction:column}}@media(max-width:560px){.metric-grid{grid-template-columns:1fr}.actions{width:100%}.actions select,.actions button{flex:1}.system-strip{flex-wrap:wrap}.free-badge{margin-left:0}}
.v2-dashboard { gap: 18px; }
.hero { position: relative; overflow: hidden; min-height: 126px; box-shadow: 0 18px 38px rgba(15, 23, 42, .14); }
.hero::after { content: ''; position: absolute; width: 260px; height: 260px; right: -70px; top: -180px; border-radius: 50%; background: radial-gradient(circle, rgba(45, 212, 191, .22), transparent 68%); pointer-events: none; }
.hero > * { position: relative; z-index: 1; }
.actions select, .actions button { transition: transform .18s ease, box-shadow .18s ease, border-color .18s ease; }
.actions .primary:hover:not(:disabled) { transform: translateY(-1px); box-shadow: 0 8px 20px rgba(34, 199, 160, .22); }
.system-strip { min-height: 54px; padding: 10px 16px; border-radius: 13px; box-shadow: 0 8px 18px rgba(6, 78, 59, .08); }
.system-copy { display: flex; flex-direction: column; min-width: 0; gap: 2px; }
.system-copy strong { font-size: 13px; }
.system-copy small { overflow: hidden; color: inherit; opacity: .78; font-size: 11px; text-overflow: ellipsis; white-space: nowrap; }
.detail-toggle { margin-left: auto; border: 1px solid currentColor; border-radius: 8px; background: transparent; color: inherit; cursor: pointer; font-size: 11px; font-weight: 800; padding: 5px 8px; }
.free-badge { margin-left: 0; }
.firewall-detail { display: flex; align-items: flex-start; gap: 10px; padding: 10px 14px; margin-top: -10px; border: 1px solid #76552c; border-radius: 11px; background: #261c13; color: #fed7aa; font-size: 12px; }
.firewall-detail span { flex-shrink: 0; font-weight: 800; }
.firewall-detail code { color: #fde68a; line-height: 1.5; word-break: break-word; }
.metric-card { min-height: 116px; position: relative; overflow: hidden; transition: transform .18s ease, border-color .18s ease, box-shadow .18s ease; }
.metric-card::after { content: ''; position: absolute; width: 100px; height: 100px; border-radius: 50%; right: -44px; bottom: -58px; opacity: .12; background: currentColor; }
.metric-card:hover { transform: translateY(-2px); border-color: #3c5a7b; box-shadow: 0 14px 28px rgba(2, 8, 23, .22); }
.metric-card strong { position: relative; z-index: 1; letter-spacing: -.04em; }
.trend-panel { box-shadow: 0 16px 34px rgba(2, 8, 23, .12); }
.trend-bars { min-height: 230px; padding-top: 24px; background-image: linear-gradient(to bottom, rgba(148, 163, 184, .09) 1px, transparent 1px); background-size: 100% 42px; }
.trend-bars.sparse { justify-content: center; gap: 18px; }
.trend-bars.sparse .trend-point { flex: 0 0 72px; }
.trend-point { min-width: 28px; }
.bars { border-bottom: 1px solid rgba(148, 163, 184, .28); }
.bars i { box-shadow: 0 -4px 10px rgba(56, 189, 248, .10); }
.chart-empty { min-height: 210px; display: flex; flex-direction: column; justify-content: center; gap: 6px; }
.chart-empty span { color: #cbd8e9; font-weight: 800; font-size: 14px; }
.chart-empty p { margin: 0; color: #7086a2; }
.panel { box-shadow: 0 12px 28px rgba(2, 8, 23, .10); }
.panel-head { min-height: 68px; }
.category-row { min-height: 46px; }
.events-panel { box-shadow: 0 16px 34px rgba(2, 8, 23, .12); }
.events-panel tbody tr { transition: background .16s ease; }
.events-panel tbody tr:hover { background: rgba(30, 50, 75, .42); }
@media(max-width: 700px) { .hero { min-height: auto; padding: 22px 20px; } .hero h1 { font-size: 25px; } .system-strip { align-items: flex-start; } .system-copy small { white-space: normal; } .detail-toggle { margin-left: 0; } .firewall-detail { flex-direction: column; } .trend-bars.sparse { justify-content: flex-start; } }
</style>
