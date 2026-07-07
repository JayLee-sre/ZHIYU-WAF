<template>
  <section class="pro-feature-gate" :class="{ fullscreen, compact }">
    <div class="gate-shell">
      <div class="gate-icon">
        <el-icon :size="28"><Lock /></el-icon>
      </div>

      <div class="gate-copy">
        <span class="gate-kicker">专业版能力</span>
        <h2>{{ title }}</h2>
        <p>{{ description }}</p>
      </div>

      <div class="gate-features" v-if="features.length">
        <div class="gate-feature" v-for="feature in features" :key="feature">
          <el-icon :size="14"><CircleCheck /></el-icon>
          <span>{{ feature }}</span>
        </div>
      </div>

      <div class="gate-actions">
        <router-link class="btn-primary gate-primary" :to="settingsTo">
          前往授权
        </router-link>
        <router-link class="gate-secondary" :to="backTo">
          返回管理面板
        </router-link>
      </div>
    </div>
  </section>
</template>

<script setup>
import { computed } from 'vue'
import { CircleCheck, Lock } from '@element-plus/icons-vue'

const props = defineProps({
  title: { type: String, required: true },
  description: { type: String, required: true },
  features: { type: Array, default: () => [] },
  featureKey: { type: String, default: '' },
  backTo: { type: String, default: '/dashboard' },
  fullscreen: { type: Boolean, default: false },
  compact: { type: Boolean, default: false },
})

const settingsTo = computed(() => {
  if (!props.featureKey) return '/settings'
  return { path: '/settings', query: { upgrade: 'pro_required', feature: props.featureKey } }
})
</script>

<style scoped>
.pro-feature-gate {
  min-height: 420px;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 28px;
}

.pro-feature-gate.fullscreen {
  min-height: 100vh;
  background:
    radial-gradient(circle at 18% -12%, rgba(37,99,235,.09), transparent 30%),
    linear-gradient(180deg, #f7f9fd 0%, #f5f7fb 48%, #f8fafc 100%);
}

.pro-feature-gate.compact {
  min-height: 0;
  padding: 0;
  justify-content: stretch;
}

.gate-shell {
  width: min(720px, 100%);
  border: 1px solid var(--border);
  border-radius: var(--radius-card);
  background: linear-gradient(180deg, #fff 0%, #fbfdff 100%);
  box-shadow: var(--shadow-card);
  padding: 28px;
  text-align: center;
}

.compact .gate-shell {
  width: 100%;
  display: grid;
  grid-template-columns: auto minmax(0, 1fr) auto;
  align-items: center;
  gap: 18px;
  text-align: left;
}

.gate-icon {
  width: 56px;
  height: 56px;
  margin: 0 auto 16px;
  border-radius: 8px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #4f46e5;
  background: #eef2ff;
  box-shadow: inset 0 0 0 1px #dbe3ff;
}

.compact .gate-icon {
  margin: 0;
}

.gate-kicker {
  display: inline-flex;
  align-items: center;
  height: 24px;
  padding: 0 9px;
  border-radius: 999px;
  background: #f8fafc;
  border: 1px solid var(--border);
  color: #64748b;
  font-size: 12px;
  font-weight: 800;
}

.gate-copy h2 {
  margin: 12px 0 8px;
  color: var(--text-primary);
  font-size: 22px;
  font-weight: 800;
  letter-spacing: 0;
}

.compact .gate-copy h2 {
  margin: 8px 0 4px;
  font-size: 18px;
}

.gate-copy p {
  max-width: 560px;
  margin: 0 auto;
  color: var(--text-secondary);
  font-size: 14px;
  line-height: 1.7;
}

.compact .gate-copy p {
  margin: 0;
}

.gate-features {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 10px;
  margin-top: 22px;
}

.compact .gate-features {
  grid-column: 1 / -1;
  margin-top: 0;
}

.gate-feature {
  min-height: 42px;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 7px;
  padding: 9px 10px;
  border: 1px solid var(--border-light);
  border-radius: 8px;
  background: #f8fafc;
  color: #475569;
  font-size: 13px;
  font-weight: 700;
}

.gate-feature :deep(.el-icon) {
  color: #059669;
  flex-shrink: 0;
}

.gate-actions {
  display: flex;
  justify-content: center;
  align-items: center;
  gap: 12px;
  margin-top: 24px;
}

.compact .gate-actions {
  margin-top: 0;
  flex-direction: column;
  align-items: flex-end;
}

.gate-primary {
  text-decoration: none;
  justify-content: center;
}

.gate-secondary {
  color: var(--text-muted);
  font-size: 13px;
  font-weight: 700;
  text-decoration: none;
}

.gate-secondary:hover {
  color: var(--primary);
}

@media (max-width: 768px) {
  .pro-feature-gate {
    padding: 16px;
    align-items: flex-start;
  }

  .gate-shell,
  .compact .gate-shell {
    display: block;
    padding: 22px;
    text-align: center;
  }

  .compact .gate-icon,
  .gate-icon {
    margin: 0 auto 14px;
  }

  .gate-features {
    grid-template-columns: 1fr;
  }

  .compact .gate-actions,
  .gate-actions {
    align-items: stretch;
    flex-direction: column;
  }

  .gate-primary {
    width: 100%;
  }
}
</style>
