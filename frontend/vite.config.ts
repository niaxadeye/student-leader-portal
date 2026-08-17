import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import path from 'node:path'

export default defineConfig({
  plugins: [react()],
  build: {
    // iOS Safari older than the Vite 6 default browser baseline still occurs on
    // event devices. Keep syntax within Safari 14 capabilities.
    target: ['es2019', 'safari14'],
    // A user may keep the SPA open while a new release is built. Preserve the
    // previous hashed chunks so that an already loaded entry module can finish
    // its lazy imports instead of receiving a 404.
    emptyOutDir: false,
  },
  resolve: {
    alias: {
      '@': path.resolve(__dirname, './src'),
    },
  },
  server: {
    host: true,
    port: 5173,
    // Локальная разработка: проксируем API на нативный Go-бэкенд (в проде это делает nginx).
    proxy: {
      '/api': 'http://127.0.0.1:8080',
    },
  },
})
