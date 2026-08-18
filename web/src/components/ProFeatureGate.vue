<template>
  <section class="feature-notice" :class="{ fullscreen, compact }">
    <div class="notice-shell">
      <el-icon :size="26"><CircleCheck /></el-icon>
      <div><span>V2 免费功能</span><h2>{{ title }}</h2><p>{{ description }}</p></div>
      <router-link :to="backTo">返回管理面板</router-link>
    </div>
  </section>
</template>
<script setup>
import { CircleCheck } from '@element-plus/icons-vue'
defineProps({ title:{type:String,required:true}, description:{type:String,required:true}, backTo:{type:String,default:'/dashboard'}, fullscreen:{type:Boolean,default:false}, compact:{type:Boolean,default:false} })
</script>
<style scoped>
.feature-notice{display:flex;min-height:360px;align-items:center;justify-content:center;padding:24px}.feature-notice.compact{min-height:0;padding:0}.notice-shell{display:flex;align-items:center;gap:16px;width:min(680px,100%);padding:22px;border:1px solid #b9eadb;border-radius:14px;background:#f0fdf8;color:#166534}.notice-shell h2{margin:4px 0;color:#14532d;font-size:18px}.notice-shell p{margin:0;color:#3f6b5a;font-size:13px;line-height:1.6}.notice-shell span{font-size:11px;font-weight:800;letter-spacing:.08em}.notice-shell>a{margin-left:auto;color:#166534;font-size:13px;font-weight:800;text-decoration:none;white-space:nowrap}@media(max-width:600px){.notice-shell{align-items:flex-start;flex-direction:column}.notice-shell>a{margin-left:0}}
</style>
