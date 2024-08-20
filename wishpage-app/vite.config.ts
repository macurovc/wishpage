import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react-swc'

const localServer = 'http://localhost:8080'

// https://vitejs.dev/config/
export default defineConfig({
  plugins: [react()],
  server: {
    proxy: {
      '/items': localServer,
      '/reserve/': localServer,
      '/login': localServer,
      '/admin/': localServer,
    }
  }
})
