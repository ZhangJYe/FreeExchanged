import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import process from 'node:process'

const appBase = process.env.VITE_APP_BASE || '/'

// https://vitejs.dev/config/
export default defineConfig({
  base: appBase,
  plugins: [react()],
  server: {
    proxy: {
      '/v1': {
        target: 'http://localhost:8888',
        changeOrigin: true,
        secure: false,
      },
      '/ws': {
        target: 'ws://localhost:8888',
        ws: true,
        changeOrigin: true,
      },
      '/free/v1': {
        target: 'http://localhost:8888',
        changeOrigin: true,
        secure: false,
        rewrite: (path) => path.replace(/^\/free/, ''),
      },
      '/free/ws': {
        target: 'ws://localhost:8888',
        ws: true,
        changeOrigin: true,
        rewrite: (path) => path.replace(/^\/free/, ''),
      }
    }
  }
})
