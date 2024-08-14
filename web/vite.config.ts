import { fileURLToPath, URL } from 'node:url'

import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import VueDevTools from 'vite-plugin-vue-devtools'

// https://vitejs.dev/config/
export default defineConfig({
  plugins: [
    vue(),
    VueDevTools(),
  ],
  resolve: {
    alias: {
      '@': fileURLToPath(new URL('./src', import.meta.url))
    }
  },
  build: {
    outDir: fileURLToPath(new URL('../internal/webui/assets', import.meta.url)),
    emptyOutDir: true,
  },
  server: {
    proxy: {
      // Provide access to the Admin API during development.
      // Uses the CCP service managed with Docker Compose.
      "/api": {
        target: process.env.CCP_ADMIN_API_URL
          ? process.env.ENDURO_API_URL
          : "http://127.0.0.1:63030",
        changeOrigin: true,
        rewrite: (path) => path.replace(/^\/api/, ""),
      },
    },
  },
})
