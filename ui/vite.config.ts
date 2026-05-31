import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

export default defineConfig({
  plugins: [vue()],
  build: {
    outDir: '../internal/webui/static',
    emptyOutDir: true,
  },
  server: {
    proxy: { '/api': 'http://localhost:8085' },
  },
})
