<script setup lang="ts">
import { ref, onMounted, computed } from "vue";
import { useRouter, useRoute } from "vue-router";
import { signOut } from "../auth";
import { checkAuth, clearCachedSession } from "../router";
import { currentTheme, setTheme } from "../theme";

const router = useRouter();
const route = useRoute();
const session = ref<any>(null);
const showLogoutModal = ref(false);
const isLoggingOut = ref(false);

const navItems = [
  { path: "/dashboard", label: "Dashboard", icon: "fa-solid fa-gauge-high" },
  { path: "/clients", label: "Klien (Polres/Polda)", icon: "fa-solid fa-building-shield" },
  { path: "/master-pelanggaran", label: "Master Pelanggaran", icon: "fa-solid fa-book-bookmark" },
  { path: "/cameras", label: "Kamera", icon: "fa-solid fa-video" },
  { path: "/violations", label: "Pelanggaran", icon: "fa-solid fa-triangle-exclamation" },
];

onMounted(async () => {
  session.value = await checkAuth();
});

const userEmail = computed(() => {
  return session.value?.user?.email || "operator@etle.local";
});

const currentRouteName = computed(() => {
  const current = navItems.find((item) => item.path === route.path);
  return current ? current.label : "Portal ETLE";
});

function openLogoutModal() {
  showLogoutModal.value = true;
}

function cancelLogout() {
  showLogoutModal.value = false;
}

async function confirmLogout() {
  isLoggingOut.value = true;
  try {
    await signOut();
    clearCachedSession();
    showLogoutModal.value = false;
    router.push("/login");
  } finally {
    isLoggingOut.value = false;
  }
}
</script>

<template>
  <div class="min-h-screen bg-slate-950 text-slate-100 flex font-sans antialiased">
    <!-- Sidebar -->
    <aside class="w-64 bg-slate-900 border-r border-slate-800 flex flex-col shrink-0">
      <!-- Sidebar Brand Header -->
      <div class="h-16 border-b border-slate-800 px-5 flex items-center gap-3">
        <div class="w-9 h-9 rounded bg-blue-600 border border-blue-400 flex items-center justify-center text-white text-base shadow-sm">
          <i class="fa-solid fa-shield-halved"></i>
        </div>
        <div class="leading-tight">
          <span class="font-bold text-sm text-white tracking-wider block">ETLE POLRI</span>
          <span class="text-[11px] text-slate-400 block font-medium">Gateway Integrasi</span>
        </div>
      </div>

      <!-- Navigation Links -->
      <div class="px-3 py-4 flex-1 space-y-1">
        <div class="px-3 pb-2 text-[10px] font-semibold text-slate-300 uppercase tracking-wider">
          Menu Navigasi
        </div>
        <router-link
          v-for="item in navItems"
          :key="item.path"
          :to="item.path"
          class="flex items-center gap-3 px-3 py-2.5 rounded text-xs font-medium transition-colors duration-150"
          :class="[
            route.path === item.path
              ? 'bg-blue-600 text-white font-semibold shadow-sm'
              : 'text-slate-300 hover:text-white hover:bg-slate-800'
          ]"
        >
          <i :class="[item.icon, 'w-4 text-center text-sm', route.path === item.path ? 'text-white' : 'text-slate-300']"></i>
          <span>{{ item.label }}</span>
        </router-link>
      </div>

      <!-- Sidebar Footer / User Profile -->
      <div class="border-t border-slate-800 p-4 bg-slate-900/80">
        <div class="flex items-center gap-3 mb-3">
          <div class="w-8 h-8 rounded-full bg-slate-800 border border-slate-700 flex items-center justify-center text-slate-300 text-xs">
            <i class="fa-solid fa-user-shield"></i>
          </div>
          <div class="flex-1 min-w-0">
            <span class="text-xs font-medium text-slate-200 truncate block">{{ userEmail }}</span>
            <span class="text-[10px] text-emerald-400 flex items-center gap-1.5 font-medium">
              <span class="w-1.5 h-1.5 rounded-full bg-emerald-400 animate-pulse"></span>
              Aktif
            </span>
          </div>
        </div>
        <button
          @click="openLogoutModal"
          class="w-full bg-slate-800 hover:bg-rose-950/60 hover:text-rose-300 hover:border-rose-700/50 text-slate-300 border border-slate-700 text-xs py-2 px-3 rounded flex items-center justify-center gap-2 transition cursor-pointer"
        >
          <i class="fa-solid fa-arrow-right-from-bracket text-xs"></i>
          <span>Keluar Sesi</span>
        </button>
      </div>
    </aside>

    <!-- Main Viewport -->
    <div class="flex-1 flex flex-col min-w-0 overflow-x-hidden">
      <!-- Top Bar -->
      <header class="h-16 bg-slate-900 border-b border-slate-800 px-8 flex items-center justify-between sticky top-0 z-20">
        <div class="flex items-center gap-3">
          <h1 class="text-base font-semibold text-white tracking-wide">
            {{ currentRouteName }}
          </h1>
          <span class="text-slate-600">/</span>
          <span class="text-xs text-slate-400 font-mono">{{ route.path }}</span>
        </div>

        <div class="flex items-center gap-3 text-xs">
          <!-- Theme Switcher (Terang / Gelap / Sistem) -->
          <div class="flex items-center bg-slate-950 border border-slate-800 rounded-lg p-1 text-xs gap-1 shadow-inner">
            <button
              type="button"
              @click="setTheme('light')"
              :class="[
                'px-2.5 py-1 rounded transition flex items-center gap-1.5 cursor-pointer font-medium text-xs',
                currentTheme === 'light'
                  ? 'bg-blue-600 text-white shadow-xs font-semibold'
                  : 'text-slate-400 hover:text-slate-200'
              ]"
              title="Mode Terang (Light Mode)"
            >
              <i class="fa-solid fa-sun text-xs"></i>
              <span class="hidden md:inline">Terang</span>
            </button>

            <button
              type="button"
              @click="setTheme('dark')"
              :class="[
                'px-2.5 py-1 rounded transition flex items-center gap-1.5 cursor-pointer font-medium text-xs',
                currentTheme === 'dark'
                  ? 'bg-blue-600 text-white shadow-xs font-semibold'
                  : 'text-slate-400 hover:text-slate-200'
              ]"
              title="Mode Gelap (Dark Mode)"
            >
              <i class="fa-solid fa-moon text-xs"></i>
              <span class="hidden md:inline">Gelap</span>
            </button>

            <button
              type="button"
              @click="setTheme('system')"
              :class="[
                'px-2.5 py-1 rounded transition flex items-center gap-1.5 cursor-pointer font-medium text-xs',
                currentTheme === 'system'
                  ? 'bg-blue-600 text-white shadow-xs font-semibold'
                  : 'text-slate-400 hover:text-slate-200'
              ]"
              title="Ikuti Tema Sistem OS"
            >
              <i class="fa-solid fa-display text-xs"></i>
              <span class="hidden md:inline">Sistem</span>
            </button>
          </div>

          <div class="flex items-center gap-2 bg-slate-950 border border-slate-800 px-3 py-1.5 rounded text-slate-300 font-mono">
            <i class="fa-solid fa-server text-blue-400"></i>
            <span>ETLE API: Online</span>
          </div>
        </div>
      </header>

      <!-- Page Content View -->
      <main class="flex-1 p-8 overflow-y-auto">
        <router-view />
      </main>
    </div>

    <!-- Modal Alert Konfirmasi Logout -->
    <Teleport to="body">
      <div
        v-if="showLogoutModal"
        class="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/60 backdrop-blur-xs transition-opacity animate-in fade-in"
      >
        <div
          class="bg-slate-900 border border-slate-800 rounded-2xl shadow-2xl max-w-sm w-full p-6 text-slate-100 animate-in zoom-in-95 duration-150"
        >
          <div class="flex items-center gap-3.5 mb-4">
            <div class="w-11 h-11 rounded-xl bg-rose-500/10 border border-rose-500/30 flex items-center justify-center text-rose-500 shrink-0">
              <i class="fa-solid fa-arrow-right-from-bracket text-lg"></i>
            </div>
            <div>
              <h3 class="text-sm font-bold text-white tracking-wide">Konfirmasi Keluar Sesi</h3>
              <p class="text-xs text-slate-400">Portal Integrasi ETLE Korlantas</p>
            </div>
          </div>

          <p class="text-xs text-slate-300 mb-6 leading-relaxed">
            Apakah Anda yakin ingin keluar dari sesi portal ETLE ini? Anda harus memasukkan kredensial login kembali untuk mengakses data.
          </p>

          <div class="flex items-center justify-end gap-2.5">
            <button
              @click="cancelLogout"
              :disabled="isLoggingOut"
              class="px-4 py-2 text-xs font-medium text-slate-300 hover:text-white bg-slate-800 hover:bg-slate-700 rounded-lg transition cursor-pointer"
            >
              Batal
            </button>
            <button
              @click="confirmLogout"
              :disabled="isLoggingOut"
              class="px-4 py-2 text-xs font-semibold text-white bg-rose-600 hover:bg-rose-500 rounded-lg transition flex items-center gap-2 cursor-pointer shadow-sm disabled:opacity-50"
            >
              <i v-if="isLoggingOut" class="fa-solid fa-circle-notch fa-spin text-xs"></i>
              <i v-else class="fa-solid fa-check text-xs"></i>
              <span>{{ isLoggingOut ? "Keluar..." : "Ya, Keluar" }}</span>
            </button>
          </div>
        </div>
      </div>
    </Teleport>
  </div>
</template>

