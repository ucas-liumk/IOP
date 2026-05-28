import { defineConfig } from "vite";
import vue from "@vitejs/plugin-vue";
import { fileURLToPath, URL } from "node:url";

export default defineConfig({
  plugins: [vue()],
  resolve: {
    alias: {
      "@": fileURLToPath(new URL("./src", import.meta.url)),
    },
  },
  server: {
    port: 5174,  // 5173 occupied by another local vite project
    proxy: {
      "/api": {
        target: "http://localhost:8080",
        changeOrigin: true,
      },
      "/livez": "http://localhost:8080",
      "/readyz": "http://localhost:8080",
      "/version": "http://localhost:8080",
      "/healthz": "http://localhost:8080",
      "/metrics": "http://localhost:8080",
    },
  },
});
