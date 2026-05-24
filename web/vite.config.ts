import { defineConfig } from "vite";
import vue from "@vitejs/plugin-vue";

export default defineConfig({
  base: "/admin/",
  plugins: [vue()],
  server: {
    proxy: {
      "/admin/api": "http://127.0.0.1:8090"
    }
  },
  build: {
    outDir: "../internal/infrastructure/http/admin_dist",
    emptyOutDir: true
  }
});
