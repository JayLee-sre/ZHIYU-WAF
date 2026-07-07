import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

export default defineConfig({
  plugins: [vue()],
  build: {
    rolldownOptions: {
      output: {
        manualChunks(id) {
          if (!id.includes('node_modules')) return
          if (id.includes('/vue/') || id.includes('/vue-router/')) return 'vue'
          if (id.includes('/element-plus/') || id.includes('/@element-plus/')) return 'element'
        },
      },
    },
  },
  server: {
    port: 3000,
    proxy: {
      '/api': {
        target: 'http://localhost:9090',
        changeOrigin: true,
        ws: true,
      },
      '/health': {
        target: 'http://localhost:9090',
        changeOrigin: true,
      },
    },
  },
})
