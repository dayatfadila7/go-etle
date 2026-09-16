<script setup lang="ts">
import { ref } from "vue";
import { useRouter, useRoute } from "vue-router";
import { authClient } from "../auth";
import { clearCachedSession } from "../router";
import { currentTheme, setTheme } from "../theme";

const router = useRouter();
const route = useRoute();

const email = ref("");
const password = ref("");
const errorMessage = ref("");
const isLoading = ref(false);

async function submitLogin() {
  isLoading.value = true;
  errorMessage.value = "";
  try {
    const res: any = await authClient.signIn.email({
      email: email.value,
      password: password.value,
    });
    if (res?.error) {
      errorMessage.value = res.error.message || "Email atau password tidak valid";
      return;
    }
    clearCachedSession();
    const redirectPath = (route.query.redirect as string) || "/dashboard";
    router.push(redirectPath);
  } catch (err: any) {
    errorMessage.value = err?.message || "Koneksi ke server gagal";
  } finally {
    isLoading.value = false;
  }
}
</script>

<template>
  <div class="min-h-screen bg-slate-950 flex flex-col items-center justify-center p-4 font-sans antialiased relative">
    <!-- Top Theme Switcher -->
    <div class="absolute top-4 right-4 flex items-center bg-slate-900 border border-slate-800 rounded-lg p-1 text-xs gap-1 shadow-sm">
      <button
        type="button"
        @click="setTheme('light')"
        :class="[
          'px-2.5 py-1 rounded transition flex items-center gap-1.5 cursor-pointer font-medium text-xs',
          currentTheme === 'light' ? 'bg-blue-600 text-white shadow-xs font-semibold' : 'text-slate-400 hover:text-slate-200'
        ]"
        title="Mode Terang (Light)"
      >
        <i class="fa-solid fa-sun text-xs"></i>
      </button>

      <button
        type="button"
        @click="setTheme('dark')"
        :class="[
          'px-2.5 py-1 rounded transition flex items-center gap-1.5 cursor-pointer font-medium text-xs',
          currentTheme === 'dark' ? 'bg-blue-600 text-white shadow-xs font-semibold' : 'text-slate-400 hover:text-slate-200'
        ]"
        title="Mode Gelap (Dark)"
      >
        <i class="fa-solid fa-moon text-xs"></i>
      </button>

      <button
        type="button"
        @click="setTheme('system')"
        :class="[
          'px-2.5 py-1 rounded transition flex items-center gap-1.5 cursor-pointer font-medium text-xs',
          currentTheme === 'system' ? 'bg-blue-600 text-white shadow-xs font-semibold' : 'text-slate-400 hover:text-slate-200'
        ]"
        title="Ikuti Sistem OS"
      >
        <i class="fa-solid fa-display text-xs"></i>
      </button>
    </div>

    <div class="w-full max-w-md bg-slate-900 border border-slate-800 rounded-xl shadow-xl overflow-hidden">
      <!-- Header -->
      <div class="p-8 pb-6 text-center border-b border-slate-800/80 bg-slate-900/50">
        <div class="w-14 h-14 mx-auto mb-4 rounded-xl bg-blue-600/10 border border-blue-500/30 flex items-center justify-center text-blue-400 text-2xl">
          <i class="fa-solid fa-shield-halved"></i>
        </div>
        <h2 class="text-xl font-bold text-white tracking-wide">SATRIA</h2>
        <p class="text-xs text-slate-400 mt-1">Sistem Antar-sistem Transfer Rekam Informasi Aplikasi Tilang Elektronik</p>
      </div>

      <!-- Form -->
      <form @submit.prevent="submitLogin" class="p-8 space-y-4">
        <div v-if="errorMessage" class="p-3 bg-rose-950/50 border border-rose-800/60 rounded text-rose-300 text-xs flex items-center gap-2.5">
          <i class="fa-solid fa-circle-exclamation text-sm shrink-0"></i>
          <span>{{ errorMessage }}</span>
        </div>

        <div class="space-y-1.5">
          <label class="block text-xs font-medium text-slate-300">Alamat Email</label>
          <div class="relative">
            <span class="absolute inset-y-0 left-0 flex items-center pl-3 text-slate-500 text-xs">
              <i class="fa-solid fa-envelope"></i>
            </span>
            <input
              v-model="email"
              type="email"
              required
              placeholder="nama@etle.local"
              class="w-full bg-slate-950 border border-slate-800 text-slate-100 text-xs pl-8 pr-3 py-2.5 rounded focus:outline-none focus:border-blue-500 focus:ring-1 focus:ring-blue-500 transition"
            />
          </div>
        </div>

        <div class="space-y-1.5">
          <label class="block text-xs font-medium text-slate-300">Kata Sandi</label>
          <div class="relative">
            <span class="absolute inset-y-0 left-0 flex items-center pl-3 text-slate-500 text-xs">
              <i class="fa-solid fa-lock"></i>
            </span>
            <input
              v-model="password"
              type="password"
              required
              placeholder="••••••••"
              class="w-full bg-slate-950 border border-slate-800 text-slate-100 text-xs pl-8 pr-3 py-2.5 rounded focus:outline-none focus:border-blue-500 focus:ring-1 focus:ring-blue-500 transition"
            />
          </div>
        </div>

        <button
          type="submit"
          :disabled="isLoading"
          class="w-full mt-2 bg-blue-600 hover:bg-blue-500 disabled:opacity-50 text-white text-xs font-semibold py-2.5 px-4 rounded transition flex items-center justify-center gap-2 cursor-pointer shadow-sm"
        >
          <i v-if="isLoading" class="fa-solid fa-circle-notch fa-spin text-sm"></i>
          <i v-else class="fa-solid fa-right-to-bracket text-xs"></i>
          <span>{{ isLoading ? "Memverifikasi Kredensial..." : "Masuk Sistem" }}</span>
        </button>
      </form>

      <!-- Footer Info -->
      <div class="px-8 py-4 bg-slate-950/60 border-t border-slate-800 text-center text-[11px] text-slate-500">
        Akses dibatasi hanya untuk personel yang terdaftar di Korlantas Polri
      </div>
    </div>
  </div>
</template>
