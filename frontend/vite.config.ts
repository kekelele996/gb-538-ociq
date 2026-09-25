import { defineConfig } from 'vitest/config'
import vue from '@vitejs/plugin-vue'

export default defineConfig({
  plugins: [vue()],
  server: {
    port: 18538,
    proxy: { '/api': 'http://127.0.0.1:19538' },
  },
  build: { chunkSizeWarningLimit: 800 },
  test: { environment: 'jsdom' },
})
