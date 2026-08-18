<template>
  <div class="soc">
    <header class="soc-header">
      <div class="brand-cluster">
        <router-link to="/dashboard" class="back-link" aria-label="返回控制台">
          <svg viewBox="0 0 24 24"><path d="M15 18l-6-6 6-6" /></svg><span>控制台</span>
        </router-link>
        <i class="divider"></i>
        <div class="shield-mark"><svg viewBox="0 0 24 24"><path d="M12 3l7 3.2v5.1c0 4.4-2.9 7.7-7 9.7-4.1-2-7-5.3-7-9.7V6.2L12 3z"/><path d="M8.4 12.2l2.2 2.2 4.9-5"/></svg></div>
        <div class="brand-copy">
          <div><h1>智域 WAF <small>SECURITY OPERATIONS</small></h1><b class="live"><i></i>LIVE</b></div>
          <p>本地风险决策与实时防护态势</p>
        </div>
      </div>
      <div class="command-bar">
        <span class="status-chip" :class="healthData.status === 'ok' ? 'healthy' : 'warning'"><i></i>{{ healthData.status === 'ok' ? '防护链路正常' : '状态待确认' }}</span>
        <span class="status-chip ai-chip" :class="{ enabled: healthData.ai_enabled }"><i></i>AI {{ healthData.ai_enabled ? '已启用' : '未启用' }}</span>
        <div class="clock"><small>最后同步 {{ lastUpdated }}</small><strong>{{ currentTime }}</strong></div>
        <button class="refresh" type="button" :disabled="isLoading" @click="refreshDashboard(true)"><svg viewBox="0 0 24 24" :class="{ spinning: isLoading }"><path d="M20 11a8 8 0 10-2.34 5.66"/><path d="M20 4v7h-7"/></svg>{{ isLoading ? '同步中' : '刷新' }}</button>
      </div>
    </header>

    <main class="soc-content">
      <section class="metric-ribbon" aria-label="24 小时安全指标">
        <article v-for="metric in metrics" :key="metric.key" class="metric-card" :style="{ '--accent': metric.color, '--accent-rgb': metric.rgb }">
          <div><span>{{ metric.label }}</span><svg viewBox="0 0 24 24"><path :d="metric.icon"/></svg></div>
          <strong>{{ metric.value }}</strong>
          <small><i></i>{{ metric.note }}</small>
        </article>
      </section>

      <section class="soc-grid">
        <aside class="side-column left-column">
          <section class="panel threat-panel">
            <header class="panel-head"><div><span>RISK DISTRIBUTION</span><h2>威胁等级分布</h2></div><b>{{ severityTotal }} 事件</b></header>
            <div ref="severityRef" class="chart pie-chart"></div>
            <div class="severity-list">
              <div v-for="item in severityLegend" :key="item.key"><span><i :style="{ background: item.color }"></i>{{ item.label }}</span><b>{{ item.count }}</b><small>{{ item.percent }}%</small></div>
            </div>
          </section>
          <section class="panel source-panel">
            <header class="panel-head compact"><div><span>DETECTION PATH</span><h2>检测来源</h2></div></header>
            <div ref="sourceRef" class="chart source-chart"></div>
            <div class="source-foot"><span><i class="blue"></i>规则 {{ formatNumber(ruleCountFromStats) }}</span><span><i class="purple"></i>AI {{ formatNumber(aiCount) }}</span></div>
          </section>
        </aside>

        <section class="center-column">
          <section class="panel map-panel world-panel">
            <header class="panel-head"><div><span>GLOBAL THREAT RADAR</span><h2>全球攻击来源</h2></div><b class="map-count"><i></i>{{ worldRegions.length }} 个境外地区</b></header>
            <div ref="worldRef" class="chart map-chart"></div>
            <div v-if="!mapsReady" class="map-loading">正在加载地理安全数据…</div>
          </section>
          <div class="bottom-grid">
            <section class="panel map-panel china-panel">
              <header class="panel-head compact"><div><span>DOMESTIC VISIBILITY</span><h2>国内攻击来源</h2></div><b>{{ chinaRegions.length }} 省/区</b></header>
              <div ref="chinaRef" class="chart mini-map"></div>
            </section>
            <section class="panel trend-panel">
              <header class="panel-head compact"><div><span>24H TELEMETRY</span><h2>风险趋势</h2></div><div class="trend-key"><span><i></i>请求</span><span><i></i>拦截</span></div></header>
              <div ref="trendRef" class="chart trend-chart"></div>
              <div v-if="!trendPoints.length" class="empty-chart">暂未收到攻击事件</div>
            </section>
          </div>
        </section>

        <aside class="side-column right-column">
          <section class="panel rank-panel">
            <header class="panel-head"><div><span>THREAT ORIGIN</span><h2>攻击来源排行</h2></div><b>TOP {{ Math.min(topRegions.length, 8) }}</b></header>
            <ol v-if="topRegions.length" class="ranking">
              <li v-for="(item, index) in topRegions.slice(0, 8)" :key="`${item.region}-${index}`">
                <em :class="{ hot: index < 3 }">{{ String(index + 1).padStart(2, '0') }}</em>
                <div><p><strong>{{ item.region || '未知地区' }}</strong><span>{{ formatNumber(item.count) }} 次</span></p><i><b :style="{ width: regionWidth(item.count) }"></b></i></div>
              </li>
            </ol>
            <div v-else class="empty-panel">暂无攻击来源数据</div>
          </section>
          <section class="panel defense-panel">
            <header class="panel-head compact"><div><span>DEFENSE POSTURE</span><h2>防护状态</h2></div><b class="coverage">{{ healthData.status === 'ok' ? '已覆盖' : '待检查' }}</b></header>
            <div class="defense-list">
              <div class="defense-row" :class="healthData.status === 'ok' ? 'active' : 'warning'"><i class="defense-icon green"><svg viewBox="0 0 24 24"><path d="M12 3l7 3.2v5.1c0 4.4-2.9 7.7-7 9.7-4.1-2-7-5.3-7-9.7V6.2L12 3z"/></svg></i><p><strong>风险决策引擎</strong><span>{{ healthData.status === 'ok' ? '持续评估中' : '等待状态确认' }}</span></p><b></b></div>
              <div class="defense-row active"><i class="defense-icon purple"><svg viewBox="0 0 24 24"><path d="M12 4v16M5 9l7-5 7 5M5 15l7 5 7-5"/></svg></i><p><strong>规则策略</strong><span>{{ ruleCount }} 条策略已加载</span></p><b></b></div>
              <div class="defense-row active"><i class="defense-icon amber"><svg viewBox="0 0 24 24"><path d="M12 3v7m0 4v7m-6.4-3.3l5.1-3m2.6-1.5l5.1-3M5.6 6.3l5.1 3m2.6 1.5l5.1 3"/></svg></i><p><strong>SSH 防护</strong><span>{{ formatNumber(sshStats.blocked || 0) }} 次封禁动作</span></p><b></b></div>
              <div class="defense-row active"><i class="defense-icon rose"><svg viewBox="0 0 24 24"><path d="M12 3v18M5 7.5l7-4.5 7 4.5v9L12 21l-7-4.5v-9z"/></svg></i><p><strong>威胁情报</strong><span>{{ formatNumber(threatCount) }} 个风险 IP</span></p><b :class="{ muted: !threatCount }"></b></div>
            </div>
            <div class="defense-summary"><div><span>SSH 失败</span><b>{{ formatNumber(sshStats.failed || 0) }}</b></div><div><span>拦截率</span><b>{{ blockRate }}%</b></div><div><span>同步周期</span><b>30s</b></div></div>
          </section>
        </aside>
      </section>
    </main>
    <footer class="soc-footer"><span><i></i>LOCAL-FIRST PROTECTION</span><span>数据每 30 秒自动刷新</span><span>智域 WAF V2</span></footer>
  </div>
</template>

<script setup>
import { computed, nextTick, onBeforeUnmount, onMounted, ref } from 'vue'
import * as echarts from 'echarts'
import api from '../api'

const stats = ref({})
const healthData = ref({})
const sshStats = ref({})
const threatInfo = ref({})
const ruleCount = ref(0)
const trendPoints = ref([])
const currentTime = ref('--:--:--')
const lastUpdated = ref('等待同步')
const isLoading = ref(false)
const mapsReady = ref(false)

const severityRef = ref(null)
const sourceRef = ref(null)
const worldRef = ref(null)
const chinaRef = ref(null)
const trendRef = ref(null)
let severityChart, sourceChart, worldChart, chinaChart, trendChart, resizeObserver, clockTimer, refreshTimer
let mapsRegistered = false

const ChinaRegions = ['北京','上海','广东','深圳','浙江','江苏','四川','湖北','湖南','福建','山东','河南','河北','安徽','辽宁','陕西','重庆','云南','广西','山西','贵州','江西','黑龙江','吉林','甘肃','内蒙古','新疆','海南','宁夏','青海','西藏','天津','中国台湾','中国香港','中国澳门']
const topRegions = computed(() => stats.value.top_regions || [])
const totalRequests = computed(() => Number(stats.value.total_requests || 0))
const blockedCount = computed(() => Number(stats.value.blocked_count || 0))
const aiCount = computed(() => Number(stats.value.ai_count || stats.value.by_source?.ai || 0))
const ruleCountFromStats = computed(() => Number(stats.value.by_source?.rule_engine || stats.value.by_source?.rule || 0))
const blockRate = computed(() => totalRequests.value ? Math.round(blockedCount.value / totalRequests.value * 100) : 0)
const threatCount = computed(() => Number((threatInfo.value.threat_ips || []).length || threatInfo.value.ip_count || 0))
const chinaRegions = computed(() => topRegions.value.filter(item => ChinaRegions.some(region => item.region?.includes(region) || region.includes(item.region))))
const worldRegions = computed(() => topRegions.value.filter(item => !ChinaRegions.some(region => item.region?.includes(region) || region.includes(item.region))))
const severityTotal = computed(() => Object.values(stats.value.by_severity || {}).reduce((total, value) => total + Number(value || 0), 0))
const severityLegend = computed(() => {
  const values = stats.value.by_severity || {}, total = severityTotal.value || 1
  return [
    { key: 'critical', label: '严重', color: '#ff5872', count: Number(values.critical || 0) },
    { key: 'high', label: '高危', color: '#ff9e53', count: Number(values.high || 0) },
    { key: 'medium', label: '中危', color: '#f7c74d', count: Number(values.medium || 0) },
    { key: 'low', label: '低危', color: '#37d7a2', count: Number(values.low || 0) },
  ].map(item => ({ ...item, percent: Math.round(item.count / total * 100) }))
})
const metrics = computed(() => [
  { key: 'requests', label: '风险检测量', value: formatNumber(totalRequests.value), note: '最近 24 小时', color: '#6694ff', rgb: '102,148,255', icon: 'M4 18V6m5 12V10m5 8V4m5 14V8' },
  { key: 'blocked', label: '已拦截攻击', value: formatNumber(blockedCount.value), note: `拦截率 ${blockRate.value}%`, color: '#ff647d', rgb: '255,100,125', icon: 'M12 3l8 4v5c0 4.8-3.2 7.9-8 9-4.8-1.1-8-4.2-8-9V7l8-4zM9 12h6' },
  { key: 'ai', label: 'AI 识别', value: formatNumber(aiCount.value), note: '智能检测结果', color: '#aa7dff', rgb: '170,125,255', icon: 'M8 4h8M8 20h8M4 8v8m16-8v8M9 9h6v6H9z' },
  { key: 'regions', label: '攻击来源', value: formatNumber(topRegions.value.length), note: '已归类地区', color: '#31d9c1', rgb: '49,217,193', icon: 'M20 10c0 5-8 11-8 11S4 15 4 10a8 8 0 1116 0zM12 12.5a2.5 2.5 0 100-5 2.5 2.5 0 000 5z' },
  { key: 'rules', label: '生效策略', value: formatNumber(ruleCount.value), note: '本地规则引擎', color: '#50b2ff', rgb: '80,178,255', icon: 'M5 4h14v16H5zM8 8h8m-8 4h8m-8 4h5' },
])
const tooltip = { backgroundColor: 'rgba(8,18,37,.96)', borderColor: 'rgba(120,157,230,.25)', borderWidth: 1, padding: [9, 11], textStyle: { color: '#e5efff', fontSize: 12 }, extraCssText: 'border-radius:8px;box-shadow:0 16px 32px rgba(0,0,0,.28)' }

function formatNumber(value) { return Number(value || 0).toLocaleString('zh-CN') }
function unwrap(response) { return response?.data ?? response ?? {} }
function regionWidth(value) { return `${Math.max(7, Math.round(Number(value || 0) / Number(topRegions.value[0]?.count || 1) * 100))}%` }
function tickClock() { currentTime.value = new Date().toLocaleTimeString('zh-CN', { hour12: false }) }
function formatBucket(timestamp) { return String(timestamp || '').match(/(\d{2}:\d{2})/)?.[1] || String(timestamp || '').slice(-5) }
function worldCoordinate(region) { const items = {'美国':[-95,38],'日本':[138,36],'韩国':[127,37],'印度':[78,21],'俄罗斯':[100,60],'德国':[10,51],'英国':[-3,55],'法国':[2,46],'巴西':[-51,-14],'加拿大':[-106,56],'澳大利亚':[133,-25],'荷兰':[5,52],'新加坡':[103,1],'印度尼西亚':[118,-1],'泰国':[100,15],'越南':[108,14],'菲律宾':[122,13],'伊朗':[53,32],'土耳其':[35,39],'意大利':[12,42],'西班牙':[-4,40],'波兰':[20,52],'乌克兰':[32,49],'墨西哥':[-102,23],'南非':[23,-30],'尼日利亚':[8,10],'埃及':[30,27],'沙特阿拉伯':[45,24],'以色列':[35,31],'马来西亚':[102,4],'中国台湾':[121,24],'中国香港':[114,22],'朝鲜':[127,40],'阿根廷':[-63,-34]}; return Object.entries(items).find(([name]) => region?.includes(name) || name.includes(region))?.[1] }
function chinaCoordinate(region) { const items = {'北京':[116.4,39.9],'上海':[121.5,31.2],'广东':[113.3,23.1],'深圳':[114.1,22.5],'浙江':[120.2,30.3],'江苏':[118.8,32.1],'四川':[104.1,30.6],'湖北':[114.3,30.6],'湖南':[112.9,28.2],'福建':[119.3,26.1],'山东':[117,36.7],'河南':[113.7,34.8],'河北':[114.5,38],'安徽':[117.3,31.8],'辽宁':[123.4,41.8],'陕西':[108.9,34.3],'重庆':[106.6,29.6],'云南':[102.7,25],'广西':[108.3,22.8],'山西':[112.9,37.9],'贵州':[106.7,26.6],'江西':[115.7,28.7],'黑龙江':[126.7,45.8],'吉林':[125.3,43.8],'甘肃':[103.8,36.1],'内蒙古':[111.7,40.8],'新疆':[87.6,43.8],'海南':[110.3,20],'宁夏':[106.3,38.5],'青海':[101.8,36.6],'西藏':[91.1,29.7],'天津':[117.2,39.1],'中国台湾':[121.5,25],'中国香港':[114.2,22.3],'中国澳门':[113.5,22.2]}; return Object.entries(items).find(([name]) => region?.includes(name) || name.includes(region))?.[1] }

async function loadMaps() {
  if (mapsRegistered) { mapsReady.value = true; return }
  try {
    const [world, china] = await Promise.all([fetch('/world.json').then(response => response.json()), fetch('/china.json').then(response => response.json())])
    echarts.registerMap('zhiyu-world', world); echarts.registerMap('zhiyu-china', china); mapsRegistered = true; mapsReady.value = true
  } catch (error) { console.warn('Unable to load map assets', error) }
}
function initCharts() {
  if (!severityChart && severityRef.value) severityChart = echarts.init(severityRef.value)
  if (!sourceChart && sourceRef.value) sourceChart = echarts.init(sourceRef.value)
  if (!worldChart && worldRef.value) worldChart = echarts.init(worldRef.value)
  if (!chinaChart && chinaRef.value) chinaChart = echarts.init(chinaRef.value)
  if (!trendChart && trendRef.value) trendChart = echarts.init(trendRef.value)
  updateCharts()
}
function updateCharts() { updateSeverity(); updateSource(); updateWorld(); updateChina(); updateTrend() }
function updateSeverity() {
  if (!severityChart) return
  const data = severityLegend.value.filter(item => item.count).map(item => ({ name: item.label, value: item.count, itemStyle: { color: item.color } }))
  severityChart.setOption({ tooltip: { ...tooltip, trigger: 'item', formatter: '{b}<br/><b>{c}</b> 次 · {d}%' }, series: [{ type: 'pie', radius: ['58%', '78%'], center: ['50%', '50%'], label: { show: false }, labelLine: { show: false }, itemStyle: { borderColor: '#101c36', borderWidth: 4, borderRadius: 5 }, emphasis: { scaleSize: 7 }, data: data.length ? data : [{ name: '暂无事件', value: 1, itemStyle: { color: '#263653' } }] }], graphic: severityTotal.value ? [{ type: 'group', left: 'center', top: 'center', children: [{ type: 'text', left: 'center', top: -18, style: { text: formatNumber(severityTotal.value), fill: '#f5f8ff', font: '800 26px Inter', textAlign: 'center' } }, { type: 'text', left: 'center', top: 15, style: { text: '风险事件', fill: '#7690b8', font: '600 11px Inter', textAlign: 'center' } }] }] : [] }, true)
}
function updateSource() {
  if (!sourceChart) return
  const rule = ruleCountFromStats.value, ai = aiCount.value
  sourceChart.setOption({ tooltip: { ...tooltip, trigger: 'item', formatter: '{b}<br/><b>{c}</b> 次 · {d}%' }, series: [{ type: 'pie', radius: ['46%', '70%'], center: ['50%', '50%'], label: { show: false }, itemStyle: { borderColor: '#101c36', borderWidth: 4, borderRadius: 4 }, data: rule || ai ? [{ name: '规则引擎', value: rule, itemStyle: { color: '#6694ff' } }, { name: 'AI 分析', value: ai, itemStyle: { color: '#aa7dff' } }] : [{ name: '暂无数据', value: 1, itemStyle: { color: '#263653' } }] }] }, true)
}
function updateWorld() {
  if (!worldChart || !mapsRegistered) return
  const data = worldRegions.value.map(item => { const coordinate = worldCoordinate(item.region); return coordinate ? { name: item.region, value: [...coordinate, Number(item.count || 0)] } : null }).filter(Boolean)
  const lines = data.map(item => ({ coords: [item.value.slice(0, 2), [104.2, 35.9]] }))
  worldChart.setOption({ tooltip: { ...tooltip, trigger: 'item', formatter: item => item.value?.[2] ? `<b>${item.name}</b><br/>风险事件 ${item.value[2]} 次` : item.name }, geo: { map: 'zhiyu-world', roam: true, zoom: 1.16, center: [17,26], scaleLimit: { min: 1, max: 5 }, itemStyle: { areaColor: '#172947', borderColor: '#3a5681', borderWidth: .65 }, emphasis: { itemStyle: { areaColor: '#284976', borderColor: '#6e9df0' }, label: { show: false } }, label: { show: false } }, series: [{ type: 'lines', coordinateSystem: 'geo', data: lines, effect: { show: true, period: 4.5, trailLength: .25, symbol: 'circle', symbolSize: 3, color: '#85afff' }, lineStyle: { color: 'rgba(120,164,255,.25)', width: 1, curveness: .18 } }, { type: 'effectScatter', coordinateSystem: 'geo', data, rippleEffect: { scale: 3, brushType: 'stroke' }, symbolSize: value => Math.max(7, Math.min(22, Math.sqrt(value[2]) * 3.4)), itemStyle: { color: '#ff5d78', shadowBlur: 14, shadowColor: 'rgba(255,93,120,.7)' } }] }, true)
}
function updateChina() {
  if (!chinaChart || !mapsRegistered) return
  const data = chinaRegions.value.map(item => { const coordinate = chinaCoordinate(item.region); return coordinate ? { name: item.region, value: [...coordinate, Number(item.count || 0)] } : null }).filter(Boolean)
  chinaChart.setOption({ tooltip: { ...tooltip, trigger: 'item', formatter: item => item.value?.[2] ? `<b>${item.name}</b><br/>风险事件 ${item.value[2]} 次` : item.name }, geo: { map: 'zhiyu-china', roam: true, zoom: 1.08, scaleLimit: { min: 1, max: 5 }, itemStyle: { areaColor: '#172947', borderColor: '#3a5681', borderWidth: .7 }, emphasis: { itemStyle: { areaColor: '#284976', borderColor: '#6e9df0' }, label: { show: false } }, label: { show: false } }, series: [{ type: 'effectScatter', coordinateSystem: 'geo', data, rippleEffect: { scale: 2.6, brushType: 'stroke' }, symbolSize: value => Math.max(7, Math.min(20, Math.sqrt(value[2]) * 3.8)), itemStyle: { color: '#74a1ff', shadowBlur: 12, shadowColor: 'rgba(116,161,255,.75)' } }] }, true)
}
function updateTrend() {
  if (!trendChart) return
  const labels = trendPoints.value.map(item => formatBucket(item.timestamp))
  trendChart.setOption({ tooltip: { ...tooltip, trigger: 'axis', axisPointer: { type: 'line', lineStyle: { color: 'rgba(120,153,220,.35)' } }, formatter: values => { const item = trendPoints.value[values[0]?.dataIndex] || {}; return `<b>${item.timestamp || ''}</b><br/>请求 <b>${formatNumber(item.requests)}</b><br/>拦截 <b>${formatNumber(item.blocked)}</b><br/>高风险 <b>${formatNumber(item.high_risk)}</b>` } }, grid: { top: 14, right: 13, bottom: 24, left: 31 }, xAxis: { type: 'category', boundaryGap: false, data: labels, axisTick: { show: false }, axisLine: { lineStyle: { color: '#2d4568' } }, axisLabel: { color: '#738aaf', fontSize: 9, interval: Math.max(0, Math.floor(labels.length / 4) - 1) } }, yAxis: { type: 'value', minInterval: 1, splitNumber: 3, axisLine: { show: false }, axisTick: { show: false }, axisLabel: { color: '#738aaf', fontSize: 9 }, splitLine: { lineStyle: { color: 'rgba(67,96,142,.35)', type: 'dashed' } } }, series: [{ name: '请求', type: 'line', smooth: .34, showSymbol: false, data: trendPoints.value.map(item => item.requests || 0), lineStyle: { color: '#6694ff', width: 2 }, areaStyle: { color: new echarts.graphic.LinearGradient(0,0,0,1,[{ offset: 0, color: 'rgba(102,148,255,.36)' }, { offset: 1, color: 'rgba(102,148,255,.02)' }]) } }, { name: '拦截', type: 'line', smooth: .34, showSymbol: false, data: trendPoints.value.map(item => item.blocked || 0), lineStyle: { color: '#ff647d', width: 2 } }, { name: '高风险', type: 'line', smooth: .34, showSymbol: false, data: trendPoints.value.map(item => item.high_risk || 0), lineStyle: { color: '#f7c74d', width: 1.5, type: 'dashed' } }] }, true)
}
async function refreshDashboard(showLoading = false) {
  if (isLoading.value) return
  isLoading.value = true
  const results = await Promise.allSettled([api.get('/stats', { suppressError: !showLoading }), api.get('/dashboard/timeseries', { params: { range: '24h' }, suppressError: !showLoading }), api.get('/health', { suppressError: !showLoading }), api.get('/ssh/stats', { suppressError: true }), api.get('/threatintel/status', { suppressError: true }), api.get('/rules', { suppressError: !showLoading })])
  const [statsResult, trendResult, healthResult, sshResult, threatResult, rulesResult] = results
  if (statsResult.status === 'fulfilled') stats.value = unwrap(statsResult.value)
  if (trendResult.status === 'fulfilled') { const result = unwrap(trendResult.value); trendPoints.value = Array.isArray(result) ? result : [] }
  if (healthResult.status === 'fulfilled') healthData.value = unwrap(healthResult.value)
  if (sshResult.status === 'fulfilled') sshStats.value = unwrap(sshResult.value)
  if (threatResult.status === 'fulfilled') threatInfo.value = unwrap(threatResult.value)
  if (rulesResult.status === 'fulfilled') { const result = unwrap(rulesResult.value); ruleCount.value = Array.isArray(result) ? result.length : 0 }
  lastUpdated.value = new Date().toLocaleTimeString('zh-CN', { hour12: false })
  await nextTick(); updateCharts(); isLoading.value = false
}
onMounted(async () => {
  tickClock(); clockTimer = window.setInterval(tickClock, 1000)
  await Promise.all([refreshDashboard(), loadMaps()]); await nextTick(); initCharts()
  resizeObserver = new ResizeObserver(() => [severityChart, sourceChart, worldChart, chinaChart, trendChart].forEach(chart => chart?.resize()))
  ;[severityRef, sourceRef, worldRef, chinaRef, trendRef].forEach(ref => ref.value && resizeObserver.observe(ref.value))
  refreshTimer = window.setInterval(() => refreshDashboard(), 30000)
})
onBeforeUnmount(() => { window.clearInterval(clockTimer); window.clearInterval(refreshTimer); resizeObserver?.disconnect(); [severityChart, sourceChart, worldChart, chinaChart, trendChart].forEach(chart => chart?.dispose()) })
</script>

<style scoped>
.soc{--bg:#07101f;--surface:#101c36;--line:rgba(111,147,211,.2);--muted:#8096b9;min-height:100vh;display:flex;flex-direction:column;overflow:hidden;color:#edf4ff;font-family:Inter,-apple-system,BlinkMacSystemFont,"Segoe UI","PingFang SC","Microsoft YaHei",sans-serif;background:radial-gradient(circle at 50% -20%,rgba(50,92,184,.28),transparent 39%),radial-gradient(circle at 8% 100%,rgba(39,106,169,.12),transparent 25%),var(--bg)}
.soc:before{content:"";position:fixed;inset:0;z-index:0;pointer-events:none;opacity:.25;background-image:linear-gradient(rgba(103,139,199,.08) 1px,transparent 1px),linear-gradient(90deg,rgba(103,139,199,.08) 1px,transparent 1px);background-size:48px 48px;mask-image:linear-gradient(to bottom,#000,transparent 76%)}.soc-header,.soc-content,.soc-footer{position:relative;z-index:1}
.soc-header{min-height:70px;display:flex;align-items:center;justify-content:space-between;gap:22px;padding:0 28px;border-bottom:1px solid var(--line);background:rgba(7,16,31,.88);backdrop-filter:blur(16px)}.brand-cluster,.command-bar,.back-link,.status-chip,.clock,.metric-card>div,.metric-card small,.map-count,.trend-key,.soc-footer{display:flex;align-items:center}.brand-cluster{min-width:0;gap:13px}.back-link{gap:5px;color:#91a6c8;text-decoration:none;font-size:12px;font-weight:700;transition:color .18s}.back-link:hover{color:#d7e5ff}.back-link svg{width:16px;height:16px;fill:none;stroke:currentColor;stroke-width:2;stroke-linecap:round;stroke-linejoin:round}.divider{width:1px;height:24px;background:var(--line)}.shield-mark{display:grid;width:34px;height:34px;place-items:center;border:1px solid rgba(122,164,255,.4);border-radius:10px;background:linear-gradient(145deg,rgba(95,141,255,.28),rgba(91,76,211,.25));box-shadow:inset 0 1px rgba(255,255,255,.11),0 0 24px rgba(95,141,255,.18)}.shield-mark svg{width:21px;fill:none;stroke:#cfe0ff;stroke-width:1.75;stroke-linecap:round;stroke-linejoin:round}.brand-copy>div{display:flex;align-items:center;gap:9px}.brand-copy h1{margin:0;font-size:14px;font-weight:800;line-height:1.25;letter-spacing:.01em}.brand-copy h1 small{margin-left:7px;color:#7187ab;font-family:ui-monospace,monospace;font-size:8px;font-weight:700;letter-spacing:.12em;vertical-align:1px}.brand-copy p{margin:3px 0 0;color:#7187aa;font-size:10px;letter-spacing:.03em}.live{display:inline-flex;align-items:center;gap:4px;padding:2px 5px;border:1px solid rgba(52,214,154,.25);border-radius:4px;background:rgba(25,183,126,.1);color:#62e4b5;font:800 8px ui-monospace,monospace;letter-spacing:.08em}.live i{width:5px;height:5px;border-radius:50%;background:currentColor;box-shadow:0 0 8px currentColor}.command-bar{gap:9px}.status-chip{gap:6px;height:29px;padding:0 9px;border:1px solid rgba(105,139,202,.18);border-radius:6px;background:rgba(20,34,64,.58);color:#96aaca;font-size:10.5px;font-weight:700}.status-chip i{width:6px;height:6px;border-radius:50%;background:#7589a9}.status-chip.healthy{color:#75e3b5;border-color:rgba(72,210,152,.18);background:rgba(33,155,114,.1)}.status-chip.healthy i,.status-chip.enabled i{background:currentColor;box-shadow:0 0 8px currentColor}.status-chip.warning{color:#ffbd61}.status-chip.warning i{background:currentColor}.ai-chip.enabled{color:#c4a6ff;border-color:rgba(170,123,255,.19);background:rgba(129,83,220,.1)}.clock{min-width:91px;flex-direction:column;align-items:flex-end;padding-left:6px}.clock small{color:#667da2;font-size:9px;font-variant-numeric:tabular-nums}.clock strong{color:#e1ebfb;font:700 13px ui-monospace,monospace;letter-spacing:.045em;font-variant-numeric:tabular-nums}.refresh{display:inline-flex;align-items:center;justify-content:center;gap:5px;height:30px;padding:0 10px;border:1px solid rgba(95,141,255,.36);border-radius:6px;background:linear-gradient(180deg,rgba(75,121,229,.27),rgba(49,75,142,.27));color:#d7e4ff;font-size:10.5px;font-weight:800;cursor:pointer}.refresh:hover:not(:disabled){border-color:rgba(130,166,255,.8);background:rgba(75,121,229,.42)}.refresh:disabled{cursor:wait;opacity:.72}.refresh svg{width:14px;height:14px;fill:none;stroke:currentColor;stroke-width:2;stroke-linecap:round;stroke-linejoin:round}.spinning{animation:spin .75s linear infinite}@keyframes spin{to{transform:rotate(360deg)}}
.soc-content{width:min(1680px,100%);flex:1;margin:0 auto;padding:17px 24px 15px}.metric-ribbon{display:grid;grid-template-columns:repeat(5,minmax(0,1fr));gap:11px}.metric-card{position:relative;min-height:107px;overflow:hidden;padding:14px 16px 12px;border:1px solid var(--line);border-radius:10px;background:linear-gradient(135deg,rgba(23,38,68,.92),rgba(12,24,47,.92));box-shadow:inset 0 1px rgba(255,255,255,.025),0 12px 26px rgba(0,0,0,.1)}.metric-card:after{content:"";position:absolute;top:0;right:0;width:58px;height:58px;background:radial-gradient(circle at 100% 0,rgba(var(--accent-rgb),.25),transparent 68%)}.metric-card:before{content:"";position:absolute;bottom:0;left:0;width:100%;height:2px;background:linear-gradient(90deg,var(--accent),transparent 68%)}.metric-card>div{position:relative;z-index:1;justify-content:space-between;color:#8ea2c1;font-size:11px;font-weight:700}.metric-card svg{width:18px;height:18px;fill:none;stroke:var(--accent);stroke-width:1.7;stroke-linecap:round;stroke-linejoin:round}.metric-card>strong{position:relative;z-index:1;display:block;margin-top:10px;color:#f5f8ff;font-size:clamp(21px,2vw,28px);font-weight:800;line-height:1;letter-spacing:-.045em;font-variant-numeric:tabular-nums}.metric-card small{position:relative;z-index:1;gap:5px;margin-top:10px;color:#7288aa;font-size:10px}.metric-card small i{width:5px;height:5px;border-radius:50%;background:var(--accent);box-shadow:0 0 8px var(--accent)}
.soc-grid{display:grid;grid-template-columns:minmax(220px,.88fr) minmax(450px,1.66fr) minmax(245px,1fr);gap:11px;min-height:min(660px,calc(100vh - 225px));margin-top:11px}.side-column,.center-column{display:flex;min-width:0;flex-direction:column;gap:11px}.panel{position:relative;display:flex;min-height:0;flex-direction:column;overflow:hidden;border:1px solid var(--line);border-radius:10px;background:linear-gradient(150deg,rgba(17,31,58,.96),rgba(11,23,45,.96));box-shadow:inset 0 1px rgba(255,255,255,.025),0 12px 28px rgba(0,0,0,.1)}.panel:before{content:"";position:absolute;top:0;left:18px;right:18px;height:1px;background:linear-gradient(90deg,transparent,rgba(138,169,238,.35),transparent)}.panel-head{display:flex;align-items:flex-start;justify-content:space-between;gap:10px;min-height:56px;padding:14px 15px 10px;border-bottom:1px solid rgba(100,134,191,.12)}.panel-head.compact{min-height:52px;padding-top:12px;padding-bottom:9px}.panel-head>div>span{display:block;color:#6087c7;font:800 8px ui-monospace,monospace;letter-spacing:.12em;line-height:1}.panel-head h2{margin:5px 0 0;color:#eaf1ff;font-size:13px;font-weight:800;letter-spacing:.005em}.panel-head>b{margin-top:2px;padding:3px 6px;border:1px solid rgba(104,145,218,.18);border-radius:4px;background:rgba(58,93,154,.12);color:#86a5dc;font:700 9px ui-monospace,monospace;white-space:nowrap}.chart{min-height:0;flex:1}.threat-panel{flex:1.1}.source-panel{flex:.9}.pie-chart{min-height:150px}.source-chart{min-height:108px}.severity-list{display:grid;grid-template-columns:1fr 1fr;gap:0 10px;padding:3px 15px 12px}.severity-list>div{display:grid;grid-template-columns:1fr auto auto;align-items:center;gap:5px;min-height:26px;border-top:1px solid rgba(98,130,183,.11);color:#99acc9;font-size:10px}.severity-list span{display:flex;align-items:center;gap:5px}.severity-list span i{width:6px;height:6px;border-radius:50%;box-shadow:0 0 7px currentColor}.severity-list b{color:#e4edfc;font:700 10px ui-monospace,monospace}.severity-list small{color:#6680aa;font-size:9px}.source-foot{display:grid;grid-template-columns:1fr 1fr;gap:4px;padding:0 14px 13px;color:#8298bb;font-size:9.5px}.source-foot span{overflow:hidden;text-overflow:ellipsis;white-space:nowrap}.source-foot i{display:inline-block;width:5px;height:5px;margin:0 4px 1px 0;border-radius:50%}.source-foot .blue{background:#6694ff;box-shadow:0 0 6px #6694ff}.source-foot .purple{background:#aa7dff;box-shadow:0 0 6px #aa7dff}
.world-panel{flex:1.16}.map-chart{min-height:280px}.map-count{gap:5px;color:#92a7c7;font-size:9.5px}.map-count i{width:5px;height:5px;border-radius:50%;background:#6694ff;box-shadow:0 0 8px #6694ff}.map-loading{position:absolute;top:50%;left:50%;color:#7992ba;font-size:11px;transform:translate(-50%,-50%)}.bottom-grid{display:grid;min-height:0;flex:.84;grid-template-columns:.95fr 1.15fr;gap:11px}.china-panel,.trend-panel{min-height:200px}.mini-map,.trend-chart{min-height:0;flex:1}.trend-key{gap:9px;padding-top:3px;color:#8fa3c1;font-size:9px}.trend-key span{display:flex;align-items:center;gap:4px}.trend-key i{width:5px;height:5px;border-radius:50%;background:#6694ff}.trend-key span:nth-child(2) i{background:#ff647d}.empty-chart{position:absolute;top:61%;left:50%;color:#7890b4;font-size:10px;transform:translate(-50%,-50%)}
.rank-panel{flex:1.15}.ranking{display:flex;flex:1;flex-direction:column;gap:1px;margin:0;padding:7px 14px 12px;list-style:none;overflow:auto}.ranking li{display:grid;grid-template-columns:25px minmax(0,1fr);align-items:center;gap:8px;min-height:38px;border-bottom:1px solid rgba(98,130,183,.1)}.ranking li:last-child{border-bottom:0}.ranking em{display:grid;width:21px;height:18px;place-items:center;border-radius:4px;background:rgba(78,102,144,.24);color:#8095b7;font:800 9px ui-monospace,monospace;font-style:normal}.ranking em.hot{background:rgba(255,97,122,.14);color:#ff8598}.ranking li>div{min-width:0}.ranking p{display:flex;justify-content:space-between;gap:8px;margin:0}.ranking strong{overflow:hidden;color:#dce6f8;font-size:10.5px;font-weight:700;text-overflow:ellipsis;white-space:nowrap}.ranking p span{color:#778eae;font:9px ui-monospace,monospace;white-space:nowrap}.ranking p+i{display:block;height:3px;margin-top:5px;overflow:hidden;border-radius:99px;background:rgba(84,112,158,.22)}.ranking p+i b{display:block;height:100%;border-radius:inherit;background:linear-gradient(90deg,#678fff,#ff6681);box-shadow:0 0 8px rgba(111,143,255,.36);transition:width .5s}.empty-panel{display:grid;min-height:170px;flex:1;place-items:center;color:#748caf;font-size:11px}.defense-panel{flex:.85}.coverage{color:#67e4b5!important;border-color:rgba(59,205,146,.21)!important;background:rgba(33,173,118,.1)!important}.defense-list{padding:4px 14px 1px}.defense-row{display:grid;grid-template-columns:28px minmax(0,1fr) 7px;align-items:center;gap:8px;min-height:47px;border-bottom:1px solid rgba(98,130,183,.1)}.defense-row:last-child{border-bottom:0}.defense-icon{display:grid;width:27px;height:27px;place-items:center;border:1px solid rgba(68,210,152,.25);border-radius:7px;background:rgba(46,197,138,.1);color:#6ee4b8}.defense-icon.purple{border-color:rgba(170,123,255,.25);background:rgba(137,88,229,.1);color:#c6a7ff}.defense-icon.amber{border-color:rgba(248,190,74,.24);background:rgba(227,161,32,.1);color:#fbc45e}.defense-icon.rose{border-color:rgba(255,97,122,.24);background:rgba(224,66,97,.1);color:#ff93a5}.defense-icon svg{width:15px;height:15px;fill:none;stroke:currentColor;stroke-width:1.8;stroke-linecap:round;stroke-linejoin:round}.defense-row p{min-width:0;margin:0}.defense-row strong,.defense-row span{display:block;overflow:hidden;text-overflow:ellipsis;white-space:nowrap}.defense-row strong{color:#dce7f8;font-size:10.5px}.defense-row span{margin-top:2px;color:#738aaf;font-size:9.5px}.defense-row>b{width:6px;height:6px;border-radius:50%;background:#4ae0a6;box-shadow:0 0 8px rgba(74,224,166,.78)}.defense-row.warning>b{background:#f5bd5b;box-shadow:0 0 8px rgba(245,189,91,.78)}.defense-row>b.muted{background:#8094b3;box-shadow:none}.defense-summary{display:grid;grid-template-columns:repeat(3,1fr);gap:1px;margin-top:auto;border-top:1px solid rgba(98,130,183,.12);background:rgba(5,14,30,.24)}.defense-summary div{padding:9px 6px 10px;text-align:center;border-right:1px solid rgba(98,130,183,.09)}.defense-summary div:last-child{border-right:0}.defense-summary span,.defense-summary b{display:block}.defense-summary span{color:#7189ad;font-size:8.5px}.defense-summary b{margin-top:4px;color:#e3ecfa;font:800 12px ui-monospace,monospace}.soc-footer{justify-content:space-between;min-height:28px;padding:0 28px;border-top:1px solid var(--line);background:rgba(6,15,29,.76);color:#536b92;font:8.5px ui-monospace,monospace;letter-spacing:.055em}.soc-footer span:first-child{color:#7597ce}.soc-footer i{display:inline-block;width:5px;height:5px;margin:0 5px 1px 0;border-radius:50%;background:#56dba8;box-shadow:0 0 7px #56dba8}
@media(max-width:1320px){.soc-grid{grid-template-columns:minmax(210px,.84fr) minmax(430px,1.55fr) minmax(225px,.94fr)}.soc-header{padding:0 18px}.soc-content{padding-left:18px;padding-right:18px}.ai-chip{display:none}}@media(max-width:1080px){.soc{overflow:auto}.soc-header{min-height:66px}.clock{display:none}.metric-ribbon{grid-template-columns:repeat(3,1fr)}.soc-grid{min-height:auto;grid-template-columns:minmax(0,1.2fr) minmax(0,1fr)}.center-column{grid-column:1/-1;min-height:660px;order:-1}.world-panel{min-height:370px}.side-column{min-height:510px}}@media(max-width:720px){.soc-header{align-items:flex-start;min-height:auto;flex-direction:column;gap:10px;padding:13px 14px}.command-bar{width:100%;justify-content:space-between}.ai-chip{display:flex}.back-link span,.divider,.brand-copy h1 small,.live{display:none}.soc-content{padding:11px}.metric-ribbon{grid-template-columns:repeat(2,1fr);gap:8px}.metric-card{min-height:91px;padding:12px}.metric-card:last-child{grid-column:1/-1}.soc-grid{display:flex;flex-direction:column;min-height:0;gap:8px;margin-top:8px}.center-column,.side-column{min-height:0}.center-column{order:0}.world-panel{min-height:320px}.map-chart{min-height:265px}.bottom-grid{min-height:470px;grid-template-columns:1fr}.china-panel,.trend-panel{min-height:220px}.threat-panel{min-height:360px}.source-panel{min-height:245px}.rank-panel{min-height:355px}.defense-panel{min-height:330px}.soc-footer{padding:0 14px}.soc-footer span:nth-child(2){display:none}}@media(prefers-reduced-motion:reduce){.spinning{animation:none}.ranking p+i b{transition:none}}
</style>
