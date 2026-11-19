import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import vuetify from 'vite-plugin-vuetify'

export default defineConfig({
  plugins: [vue(), vuetify({ autoImport: true })],
  server: {
    // En développement, l'API est servie par le binaire Go (go run . → :8080).
    proxy: { '/api': 'http://localhost:8080' },
  },
  build: {
    outDir: 'dist', // embarqué dans le binaire Go par web/embed.go
    emptyOutDir: true,
  },
})
