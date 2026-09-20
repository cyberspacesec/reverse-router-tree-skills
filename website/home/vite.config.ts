import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

// GitHub Pages 项目站点：部署在 /reverse-router-tree-skills/ 子路径下，
// 资源引用必须带此前缀，否则线上白屏。
const base = '/reverse-router-tree-skills/'

export default defineConfig({
  base,
  plugins: [react()],
  build: {
    outDir: 'dist',
    sourcemap: false,
  },
})
