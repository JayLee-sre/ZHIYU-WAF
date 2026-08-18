<template>
  <div class="v2-sites">
    <header class="site-hero">
      <div class="hero-copy">
        <p class="eyebrow">V2 SITES · LOCAL ROUTING</p>
        <h1>站点与防护模式</h1>
        <p>按域名维护回源与风险动作。配置保存后，本地路由会即时刷新，无需任何授权或重启。</p>
      </div>
      <div class="hero-actions">
        <span class="local-badge"><i></i> 本地优先</span>
        <button class="primary" @click="openCreate">新增站点</button>
      </div>
    </header>

    <section class="mode-guide" aria-label="防护模式说明">
      <article class="mode-card monitor"><span class="mode-symbol">◌</span><div><b>MONITOR</b><p>保留完整风险证据，不打断业务访问。</p></div></article>
      <article class="mode-card protect"><span class="mode-symbol">✓</span><div><b>PROTECT</b><p>按统一风险动作执行限流与阻断。</p></div></article>
      <article class="mode-card emergency"><span class="mode-symbol">!</span><div><b>EMERGENCY</b><p>站点维护优先，用于应急处置。</p></div></article>
    </section>

    <section class="panel site-panel">
      <div class="panel-head">
        <div><h2>接入站点</h2><p>{{ sites.length ? `当前已接入 ${sites.length} 个站点，按 Host 精确或泛域名匹配。` : '从一个业务域名开始，逐步在观察模式下验证防护策略。' }}</p></div>
        <button class="ghost refresh" :disabled="loading" @click="loadSites">{{ loading ? '刷新中…' : '刷新' }}</button>
      </div>
      <div class="table-wrap" v-if="sites.length">
        <table>
          <thead><tr><th>域名</th><th>回源地址</th><th>防护模式</th><th>状态</th><th>更新时间</th><th><span class="sr-only">操作</span></th></tr></thead>
          <tbody>
            <tr v-for="site in sites" :key="site.id">
              <td><div class="domain-cell"><span class="domain-mark">↗</span><strong>{{ site.domain }}</strong></div></td>
              <td class="mono backend">{{ site.backend_url }}</td>
              <td><span class="mode-pill" :class="site.mode">{{ modeText(site.mode) }}</span></td>
              <td><span class="state-pill" :class="site.enabled ? 'enabled' : 'disabled'"><i></i>{{ site.enabled ? '已启用' : '已停用' }}</span></td>
              <td class="updated">{{ time(site.updated_at) }}</td>
              <td class="row-actions"><button type="button" title="编辑站点" @click="openEdit(site)">编辑</button><button type="button" :title="site.enabled ? '停用站点' : '启用站点'" @click="toggle(site)">{{ site.enabled ? '停用' : '启用' }}</button><button type="button" class="danger" title="删除站点" @click="remove(site)">删除</button></td>
            </tr>
          </tbody>
        </table>
      </div>
      <div v-else class="empty-state">
        <div class="empty-orbit"><span>＋</span></div><h3>还没有接入站点</h3><p>建议先添加一个非核心域名，并使用 <strong>MONITOR</strong> 模式观察风险命中，再切换至正式防护。</p><button class="primary" @click="openCreate">添加首个站点</button>
      </div>
    </section>

    <div v-if="dialog" class="overlay" @click.self="dialog=false">
      <form class="dialog" @submit.prevent="save">
        <div class="dialog-head"><div><p class="eyebrow">V2 SITE CONFIGURATION</p><h2>{{ editing ? '编辑站点' : '新增站点' }}</h2><p>请填写可从 WAF 所在网络访问的完整 HTTP/HTTPS 回源地址。</p></div><button type="button" aria-label="关闭" @click="dialog=false">×</button></div>
        <div class="form-grid">
          <label>域名<input v-model.trim="form.domain" placeholder="api.example.com" autocomplete="off" required><small>支持精确域名与 <code>*.example.com</code> 泛域名。</small></label>
          <label>回源地址<input v-model.trim="form.backend_url" placeholder="http://127.0.0.1:3000" autocomplete="off" required><small>建议使用内网地址，避免形成回源循环。</small></label>
        </div>
        <fieldset><legend>防护模式</legend><div class="mode-options"><label v-for="mode in modes" :key="mode.value" class="mode-option" :class="{ selected: form.mode === mode.value, [mode.value]: true }"><input v-model="form.mode" type="radio" :value="mode.value"><span><b>{{ mode.label }}</b><small>{{ mode.description }}</small></span></label></div></fieldset>
        <label class="enable-option"><input type="checkbox" v-model="form.enabled"><span><b>立即启用该站点</b><small>关闭后保留配置，但不会参与域名匹配和回源。</small></span></label>
        <p class="hint" v-if="form.mode==='emergency'">紧急模式会优先返回维护页。请只在发布故障、攻击应急或人工处置期间使用。</p>
        <div class="dialog-actions"><button type="button" class="ghost" @click="dialog=false">取消</button><button class="primary" :disabled="saving">{{ saving ? '保存中…' : editing ? '保存变更' : '创建站点' }}</button></div>
      </form>
    </div>
  </div>
</template>
<script setup>
import { onMounted, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import api from '../api'
const sites = ref([]), dialog = ref(false), editing = ref(false), saving = ref(false), loading = ref(false)
const form = reactive(blank())
const modes = [
  { value: 'monitor', label: 'MONITOR', description: '记录风险，不主动阻断。' },
  { value: 'protect', label: 'PROTECT', description: '执行统一风险动作。' },
  { value: 'emergency', label: 'EMERGENCY', description: '立即进入维护优先。' },
]
function blank() { return { id: null, domain: '', backend_url: '', mode: 'protect', enabled: true } }
function unwrap(r) { return r?.data ?? r ?? [] }
async function loadSites() { loading.value = true; try { const res = await api.get('/sites'); sites.value = unwrap(res) } catch {} finally { loading.value = false } }
function openCreate() { Object.assign(form, blank()); editing.value = false; dialog.value = true }
function openEdit(site) { Object.assign(form, JSON.parse(JSON.stringify(site))); editing.value = true; dialog.value = true }
async function save() { saving.value = true; try { if (editing.value) await api.put(`/sites/${form.id}`, form); else await api.post('/sites', form); ElMessage.success('V2 站点配置已保存并刷新本地路由'); dialog.value = false; await loadSites() } finally { saving.value = false } }
async function toggle(site) { await api.put(`/sites/${site.id}`, { ...site, enabled: !site.enabled }); ElMessage.success(site.enabled ? '站点已停用' : '站点已启用'); await loadSites() }
async function remove(site) { try { await ElMessageBox.confirm(`确认删除站点「${site.domain}」？此操作会立即停止该域名的本地匹配。`, '删除站点', { type: 'warning', confirmButtonText: '删除', cancelButtonText: '取消' }); await api.delete(`/sites/${site.id}`); ElMessage.success('站点已删除'); await loadSites() } catch {} }
function modeText(mode) { return { monitor: '观察模式', protect: '防护模式', emergency: '紧急维护' }[mode] || mode }
function time(value) { return value ? new Date(value).toLocaleString('zh-CN', { hour12: false }) : '-' }
onMounted(loadSites)
</script>
<style scoped>
.v2-sites { display: flex; flex-direction: column; gap: 18px; color: #15243b; }
.site-hero { position: relative; display: flex; justify-content: space-between; align-items: center; gap: 24px; overflow: hidden; padding: 27px 28px; border: 1px solid #d9e6f5; border-radius: 18px; background: radial-gradient(circle at 92% 5%, rgba(45, 212, 191, .18), transparent 23%), linear-gradient(120deg, #fafdff, #eef6ff 62%, #eaf8f6); box-shadow: 0 12px 28px rgba(30, 64, 175, .06); }
.site-hero::after { content: ''; position: absolute; right: 160px; bottom: -68px; width: 180px; height: 180px; border: 1px solid rgba(37, 99, 235, .12); border-radius: 50%; }
.hero-copy,.hero-actions { position: relative; z-index: 1; }.eyebrow { margin: 0 0 7px; color: #0f766e; font: 800 10px/1.3 ui-monospace, monospace; letter-spacing: .13em; }.site-hero h1,.panel h2,.dialog h2 { margin: 0; color: #13233b; }.site-hero h1 { font-size: 27px; letter-spacing: -.03em; }.site-hero p,.panel-head p,.dialog-head p { margin: 7px 0 0; color: #64748b; font-size: 13px; line-height: 1.6; }.hero-actions { display: flex; align-items: center; gap: 10px; }.primary,.ghost,.row-actions button { border-radius: 9px; padding: 9px 13px; font-size: 12px; font-weight: 800; cursor: pointer; transition: transform .16s ease, box-shadow .16s ease, border-color .16s ease; }.primary { border: 0; background: linear-gradient(135deg, #0f766e, #0891b2); color: #fff; box-shadow: 0 8px 16px rgba(15, 118, 110, .16); }.primary:hover:not(:disabled) { transform: translateY(-1px); box-shadow: 0 11px 21px rgba(15, 118, 110, .23); }.primary:disabled { opacity: .62; cursor: not-allowed; }.ghost,.row-actions button { border: 1px solid #cfdae8; background: #fff; color: #41536d; }.ghost:hover,.row-actions button:hover { border-color: #93c5fd; color: #1d4ed8; background: #f8fbff; }.local-badge { display: inline-flex; align-items: center; gap: 6px; padding: 6px 9px; border: 1px solid #a7f3d0; border-radius: 999px; background: rgba(255,255,255,.72); color: #047857; font-size: 11px; font-weight: 800; }.local-badge i,.state-pill i { width: 6px; height: 6px; border-radius: 50%; background: currentColor; box-shadow: 0 0 0 3px rgba(5, 150, 105, .11); }.mode-guide { display: grid; grid-template-columns: repeat(3, 1fr); gap: 12px; }.mode-card { display: flex; align-items: center; gap: 11px; padding: 14px; border: 1px solid #dbe6f2; border-radius: 13px; background: #fff; box-shadow: 0 6px 16px rgba(15, 23, 42, .035); }.mode-symbol { display: grid; place-items: center; width: 29px; height: 29px; flex: 0 0 auto; border-radius: 9px; font-size: 16px; font-weight: 900; }.mode-card b { font: 800 11px ui-monospace, monospace; letter-spacing: .03em; }.mode-card p { margin: 3px 0 0; color: #71829a; font-size: 12px; }.mode-card.monitor .mode-symbol { background: #e0f2fe; color: #0369a1; }.mode-card.protect .mode-symbol { background: #dcfce7; color: #15803d; }.mode-card.emergency .mode-symbol { background: #fff1f2; color: #be123c; }.panel { border: 1px solid #dbe6f2; border-radius: 15px; background: #fff; overflow: hidden; box-shadow: 0 10px 24px rgba(15, 23, 42, .045); }.panel-head { display: flex; justify-content: space-between; gap: 18px; align-items: center; padding: 18px 20px; border-bottom: 1px solid #eaf0f7; }.panel-head h2 { font-size: 16px; }.refresh { min-width: 64px; }.table-wrap { overflow: auto; }table { width: 100%; min-width: 820px; border-collapse: collapse; }th,td { padding: 14px 18px; border-bottom: 1px solid #edf2f7; text-align: left; font-size: 13px; }th { background: #f8fbfe; color: #75879f; font-size: 10px; font-weight: 800; letter-spacing: .08em; text-transform: uppercase; }tbody tr { transition: background .16s ease; }tbody tr:hover { background: #f8fbff; }tbody tr:last-child td { border-bottom: 0; }.domain-cell { display: flex; gap: 9px; align-items: center; }.domain-mark { display: grid; place-items: center; width: 24px; height: 24px; border-radius: 7px; background: #eff6ff; color: #2563eb; font-size: 14px; }.domain-cell strong { color: #1e3a5f; }.mono { font-family: ui-monospace, SFMono-Regular, Menlo, monospace; color: #475569; font-size: 12px; }.mode-pill { display: inline-flex; padding: 4px 8px; border-radius: 7px; font: 800 10px ui-monospace, monospace; }.mode-pill.monitor { background: #e0f2fe; color: #0369a1; }.mode-pill.protect { background: #dcfce7; color: #15803d; }.mode-pill.emergency { background: #fff1f2; color: #be123c; }.state-pill { display: inline-flex; align-items: center; gap: 7px; font-size: 12px; font-weight: 800; }.state-pill.enabled { color: #15803d; }.state-pill.disabled { color: #94a3b8; }.state-pill.disabled i { box-shadow: none; }.updated { color: #71829a; font-size: 12px; }.row-actions { display: flex; justify-content: flex-end; gap: 7px; }.row-actions button { padding: 6px 9px; font-size: 11px; }.row-actions .danger { color: #be123c; border-color: #fecdd3; }.row-actions .danger:hover { background: #fff1f2; border-color: #fda4af; }.empty-state { display: flex; flex-direction: column; align-items: center; padding: 54px 24px; text-align: center; }.empty-orbit { display: grid; place-items: center; width: 58px; height: 58px; margin-bottom: 12px; border: 1px solid #bfdbfe; border-radius: 50%; background: radial-gradient(circle at 35% 30%, #fff, #eff6ff); color: #2563eb; box-shadow: 0 0 0 10px #f8fbff; }.empty-orbit span { display: grid; place-items: center; width: 30px; height: 30px; border-radius: 10px; background: #2563eb; color: #fff; font-size: 22px; line-height: 1; }.empty-state h3 { margin: 0; color: #1e3a5f; font-size: 16px; }.empty-state p { max-width: 430px; margin: 7px 0 16px; color: #71829a; font-size: 13px; line-height: 1.7; }.overlay { position: fixed; z-index: 200; inset: 0; display: grid; place-items: center; padding: 18px; background: rgba(15,23,42,.48); backdrop-filter: blur(5px); }.dialog { width: min(650px,100%); max-height: calc(100vh - 36px); overflow: auto; padding: 23px; border: 1px solid rgba(255,255,255,.7); border-radius: 18px; background: #fff; box-shadow: 0 28px 70px rgba(15,23,42,.28); }.dialog-head { display: flex; justify-content: space-between; gap: 18px; margin-bottom: 19px; }.dialog-head h2 { font-size: 20px; }.dialog-head button { width: 30px; height: 30px; border: 0; border-radius: 8px; background: transparent; color: #64748b; font-size: 24px; cursor: pointer; }.dialog-head button:hover { background: #f1f5f9; color: #1e293b; }.form-grid { display: grid; grid-template-columns: 1fr 1fr; gap: 14px; }.dialog label { display: flex; flex-direction: column; gap: 6px; color: #334155; font-size: 12px; font-weight: 800; }.dialog input:not([type="radio"]),.dialog select { width: 100%; border: 1px solid #cbd8e6; border-radius: 9px; padding: 10px 11px; background: #fff; color: #172554; font: inherit; outline: none; transition: border-color .16s ease, box-shadow .16s ease; }.dialog input:focus,.dialog select:focus { border-color: #2563eb; box-shadow: 0 0 0 3px rgba(37,99,235,.10); }.dialog small { color: #8090a5; font-size: 11px; font-weight: 500; line-height: 1.5; }.dialog code { color: #2563eb; }fieldset { margin: 18px 0 14px; padding: 0; border: 0; }legend { margin-bottom: 9px; color: #334155; font-size: 12px; font-weight: 800; }.mode-options { display: grid; grid-template-columns: repeat(3,1fr); gap: 9px; }.mode-option { position: relative; min-height: 80px; padding: 11px; border: 1px solid #d8e3ef; border-radius: 11px; cursor: pointer; transition: border-color .16s ease, background .16s ease, transform .16s ease; }.mode-option:hover { transform: translateY(-1px); }.mode-option input { position: absolute; opacity: 0; pointer-events: none; }.mode-option.selected.monitor { border-color: #7dd3fc; background: #f0f9ff; }.mode-option.selected.protect { border-color: #86efac; background: #f0fdf4; }.mode-option.selected.emergency { border-color: #fda4af; background: #fff5f5; }.mode-option b { display: block; font: 800 10px ui-monospace, monospace; }.mode-option small { display: block; margin-top: 5px; }.enable-option { flex-direction: row !important; align-items: flex-start; gap: 10px !important; padding: 11px 12px; border: 1px solid #dbe6f2; border-radius: 10px; background: #f8fbff; cursor: pointer; }.enable-option input { width: 16px; height: 16px; margin-top: 1px; accent-color: #0f766e; }.enable-option b,.enable-option small { display: block; }.hint { margin: 14px 0 0; padding: 10px 12px; border: 1px solid #fed7aa; border-radius: 10px; background: #fff7ed; color: #9a3412; font-size: 12px; line-height: 1.6; }.dialog-actions { display: flex; justify-content: flex-end; gap: 9px; margin-top: 20px; }.sr-only { position: absolute; width: 1px; height: 1px; padding: 0; overflow: hidden; clip: rect(0,0,0,0); white-space: nowrap; border: 0; }
@media(max-width:800px) { .site-hero { align-items: flex-start; flex-direction: column; }.hero-actions { width: 100%; justify-content: space-between; }.mode-guide { grid-template-columns: 1fr; }.mode-card { padding: 13px 14px; } }
@media(max-width:560px) { .site-hero { padding: 22px 20px; }.site-hero h1 { font-size: 24px; }.hero-actions .primary { flex: 1; }.form-grid,.mode-options { grid-template-columns: 1fr; }.dialog { padding: 19px; }.dialog-actions > * { flex: 1; }.local-badge { display: none; } }
</style>
