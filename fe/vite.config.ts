import { defineConfig } from "vite";
import vue from "@vitejs/plugin-vue";
import tailwindcss from "@tailwindcss/vite";

export default defineConfig({
  plugins: [vue(), tailwindcss()],
  server: {
    proxy: {
      // FE (5173) -> BFF (3000) satu origin, hindari CORS
      "/api": "http://localhost:3000",
    },
  },
});
