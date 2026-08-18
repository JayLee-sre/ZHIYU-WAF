<template>
  <div class="v2-sites">
    <header><div><p class="eyebrow">V2 SITES</p><h1>站点与防护模式</h1><p>按域名配置回源地址与风险动作模式。无需授权，变更会即时刷新本地路由。</p></div><button class="primary" @click="openCreate">新增站点</button></header>
    <div class="mode-guide"><span><b>MONITOR</b> 记录风险但不阻断</span><span><b>PROTECT</b> 按风险动作执行防护</span><span><b>EMERGENCY</b> 站点维护优先</span></div>
    <section class="panel"><div class="panel-head"><div><h2>站点列表</h2><p>{{ sites.length }} 个站点，由 Host 精确或泛域名匹配。</p></div><button class="ghost" @click="loadSites">刷新</button></div>
      <div class="table-wrap" v-if="sites.length"><table><thead><tr><th>域名</th><th>回源地址</th><th>模式</th><th>状态</th><th>更新时间</th><th></th></tr></thead><tbody>
        <tr v-for="site in sites" :key="site.id"><td><strong>{{ site.domain }}</strong></td><td class="mono">{{ site.backend_url }}</td><td><span class="mode" :class="site.mode">{{ site.mode }}</span></td><td><span :class="site.enabled ? 'enabled' : 'disabled'">{{ site.enabled ? '已启用' : '已停用' }}</span></td><td>{{ time(site.updated_at) }}</td><td class="row-actions"><button @click="openEdit(site)">编辑</button><button @click="toggle(site)">{{ site.enabled ? '停用' : '启用' }}</button><button class="danger" @click="remove(site)">删除</button></td></tr>
      </tbody></table></div><div v-else class="empty">暂无站点。添加一个域名即可开始以 V2 模式接入保护。</div>
    </section>
    <div v-if="dialog" class="overlay" @click.self="dialog=false"><form class="dialog" @submit.prevent="save"><div class="dialog-head"><div><h2>{{ editing ? '编辑站点' : '新增站点' }}</h2><p>使用完整 HTTP/HTTPS 回源地址，例如 http://127.0.0.1:3000。</p></div><button type="button" @click="dialog=false">×</button></div>
      <label>域名<input v-model.trim="form.domain" placeholder="api.example.com" required></label><label>回源地址<input v-model.trim="form.backend_url" placeholder="http://127.0.0.1:3000" required></label>
      <div class="split"><label>防护模式<select v-model="form.mode"><option value="monitor">monitor</option><option value="protect">protect</option><option value="emergency">emergency</option></select></label><label class="check"><input type="checkbox" v-model="form.enabled"> 启用站点</label></div>
      <p class="hint" v-if="form.mode==='emergency'">紧急模式会同时使站点进入维护状态，应仅在应急处置期间使用。</p><div class="dialog-actions"><button type="button" class="ghost" @click="dialog=false">取消</button><button class="primary" :disabled="saving">{{ saving ? '保存中…' : '保存' }}</button></div>
    </form></div>
  </div>
</template>
<script setup>
import { onMounted, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import api from '../api'
const sites=ref([]),dialog=ref(false),editing=ref(false),saving=ref(false)
const form=reactive(blank())
function blank(){return {id:null,domain:'',backend_url:'',mode:'protect',enabled:true}}
function unwrap(r){return r?.data??r??[]}
async function loadSites(){try{const res=await api.get('/sites');sites.value=unwrap(res)}catch{}}
function openCreate(){Object.assign(form,blank());editing.value=false;dialog.value=true}
function openEdit(site){Object.assign(form,JSON.parse(JSON.stringify(site)));editing.value=true;dialog.value=true}
async function save(){saving.value=true;try{if(editing.value)await api.put(`/sites/${form.id}`,form);else await api.post('/sites',form);ElMessage.success('V2 站点配置已保存');dialog.value=false;loadSites()}finally{saving.value=false}}
async function toggle(site){await api.put(`/sites/${site.id}`,{...site,enabled:!site.enabled});loadSites()}
async function remove(site){try{await ElMessageBox.confirm(`确认删除 ${site.domain}？`,`删除站点`,{type:'warning'});await api.delete(`/sites/${site.id}`);ElMessage.success('站点已删除');loadSites()}catch{}}
function time(value){return value?new Date(value).toLocaleString('zh-CN',{hour12:false}):'-'}
onMounted(loadSites)
</script>
<style scoped>
.v2-sites{display:flex;flex-direction:column;gap:16px}.v2-sites header{display:flex;justify-content:space-between;align-items:center;gap:20px;padding:24px;border:1px solid #d9e3f0;border-radius:16px;background:linear-gradient(120deg,#f8fbff,#eef6ff)}.eyebrow{margin:0 0 6px;color:#0f766e;font:800 11px ui-monospace,monospace;letter-spacing:.12em}.v2-sites h1,.v2-sites h2{margin:0;color:#14213a}.v2-sites header p,.panel-head p{margin:6px 0 0;font-size:13px;color:#63748b}.primary,.ghost,.row-actions button{border-radius:8px;padding:8px 12px;font-weight:700;cursor:pointer}.primary{border:0;background:#0f766e;color:#fff}.ghost,.row-actions button{border:1px solid #cbd7e6;background:#fff;color:#41536d}.mode-guide{display:grid;grid-template-columns:repeat(3,1fr);gap:10px}.mode-guide span{padding:12px;border:1px solid #dbe4f1;border-radius:10px;background:#fff;color:#63748b;font-size:12px}.mode-guide b{color:#172554}.panel{border:1px solid #dbe4f1;border-radius:14px;background:#fff;overflow:hidden}.panel-head{display:flex;justify-content:space-between;padding:16px 18px;border-bottom:1px solid #e6edf5}.table-wrap{overflow:auto}table{width:100%;border-collapse:collapse;min-width:740px}th,td{text-align:left;padding:13px 16px;border-bottom:1px solid #edf1f6;font-size:13px}th{font-size:11px;text-transform:uppercase;color:#74869f;background:#f8fafc}.mono{font-family:ui-monospace,monospace;color:#334155}.mode{font:800 11px ui-monospace,monospace;padding:4px 7px;border-radius:6px}.mode.monitor{background:#e0f2fe;color:#0369a1}.mode.protect{background:#dcfce7;color:#15803d}.mode.emergency{background:#fff1f2;color:#be123c}.enabled{color:#15803d;font-weight:700}.disabled{color:#94a3b8}.row-actions{display:flex;gap:7px}.row-actions .danger{color:#be123c;border-color:#fecdd3}.empty{text-align:center;padding:40px;color:#7b8ca4}.overlay{position:fixed;z-index:20;inset:0;background:rgba(15,23,42,.48);display:grid;place-items:center;padding:16px}.dialog{width:min(540px,100%);padding:20px;border-radius:15px;background:#fff;box-shadow:0 24px 60px rgba(15,23,42,.25)}.dialog-head{display:flex;justify-content:space-between;gap:16px;margin-bottom:16px}.dialog-head button{border:0;background:none;font-size:24px;color:#64748b}.dialog label{display:flex;flex-direction:column;gap:6px;margin:12px 0;color:#475569;font-size:12px;font-weight:700}.dialog input,.dialog select{border:1px solid #cbd7e6;border-radius:8px;padding:10px;font:inherit;color:#172554}.split{display:grid;grid-template-columns:1fr 1fr;gap:12px}.dialog .check{flex-direction:row;align-items:center;justify-content:center}.hint{padding:10px;border-radius:8px;background:#fff7ed;color:#9a3412;font-size:12px}.dialog-actions{display:flex;justify-content:end;gap:9px;margin-top:18px}@media(max-width:700px){.v2-sites header{align-items:start;flex-direction:column}.mode-guide{grid-template-columns:1fr}.split{grid-template-columns:1fr}}
</style>
