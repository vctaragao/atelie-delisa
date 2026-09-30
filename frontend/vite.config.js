import { defineConfig } from 'vite'
import { svelte } from '@sveltejs/vite-plugin-svelte'

// O alvo do proxy vem do ambiente: dentro do Docker é o nome do serviço
// (http://backend:8080); rodando fora, localhost.
const apiTarget = process.env.VITE_API_TARGET || 'http://localhost:8080'

export default defineConfig({
  plugins: [svelte()],
  server: {
    port: 5173,
    host: '0.0.0.0',
    // No Windows, o bind mount do Docker não propaga eventos de arquivo,
    // então o watcher precisa fazer polling para o hot reload funcionar.
    watch: { usePolling: true, interval: 300 },
    proxy: {
      '/api': { target: apiTarget, changeOrigin: true },
    },
  },
  build: { outDir: 'dist' },
})
