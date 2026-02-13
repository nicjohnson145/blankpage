import {defineConfig} from 'vite'
import react from '@vitejs/plugin-react'

// https://vite.dev/config/
export default defineConfig({
    build: {
        chunkSizeWarningLimit: 600,
    },
    plugins: [react()],
    server: {
        proxy: {
            "/pauth.v1beta1.PAuthService": "http://localhost:8080",
            "/blankpage.v1.BlankPageService": "http://localhost:8080",
        },
    },
})
