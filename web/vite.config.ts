import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import vuetify from 'vite-plugin-vuetify'

export default defineConfig({
  plugins: [vue(), vuetify({ autoImport: true })],
  server: {
    // In development the API is served by the Go binary (go run . → :8080).
    proxy: { '/api': 'http://localhost:8080' },
  },
  build: {
    outDir: 'dist', // embedded into the Go binary by web/embed.go
    emptyOutDir: true,
  },
})
