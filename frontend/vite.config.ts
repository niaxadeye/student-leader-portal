import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import path from 'node:path'

export default defineConfig({
  plugins: [react()],
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
