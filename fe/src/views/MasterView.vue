<script setup lang="ts">
import { ref, onMounted, computed } from "vue";
import { api } from "../api";

const master = ref<any[]>([]);
const searchQuery = ref("");
const isLoading = ref(false);
const isSyncing = ref(false);
const notice = ref("");

async function loadMaster() {
  isLoading.value = true;
  try {
    master.value = await api.master();
  } catch (e: any) {
    notice.value = "Gagal memuat master pelanggaran: " + e.message;
  } finally {
    isLoading.value = false;
  }
}

async function handleSync() {
  isSyncing.value = true;
  try {
    const res = await api.syncAll();
    notice.value = res.message || `Berhasil sinkronisasi ${res.synced} data dari Polantas`;
    await loadMaster();
  } catch (e: any) {
    notice.value = "Sinkronisasi gagal: " + e.message;
  } finally {
    isSyncing.value = false;
  }
}

const filteredMaster = computed(() => {
  const q = searchQuery.value.trim().toLowerCase();
  if (!q) return master.value;
  return master.value.filter((m) => {
    const code = (m.code || m.Code || "").toLowerCase();
    const name = (m.name || m.Name || "").toLowerCase();
    return code.includes(q) || name.includes(q);
  });
});

onMounted(loadMaster);
</script>

<template>
  <div class="space-y-6 max-w-7xl mx-auto">
    <!-- Header -->
    <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
      <div>
        <h2 class="text-lg font-bold text-white tracking-wide">Master Pelanggaran Korlantas</h2>
        <p class="text-xs text-slate-400">Daftar klasifikasi kode pelanggaran dan pasal lalu lintas resmi Polri</p>
      </div>

      <div class="flex items-center gap-2.5">
        <button
          @click="handleSync"
          :disabled="isSyncing"
          class="bg-blue-600 hover:bg-blue-500 disabled:opacity-50 text-white text-xs font-semibold py-2 px-3.5 rounded transition flex items-center gap-2 cursor-pointer shadow-sm"
        >
          <i :class="['fa-solid', isSyncing ? 'fa-circle-notch fa-spin' : 'fa-arrows-rotate']"></i>
          <span>{{ isSyncing ? "Menyinkronkan..." : "Sinkronisasi Polantas" }}</span>
        </button>

        <button
          @click="loadMaster"
          class="bg-white hover:bg-slate-100 text-slate-700 border border-slate-300 dark:bg-slate-800 dark:hover:bg-slate-700 dark:text-slate-300 dark:border-slate-700 text-xs p-2 rounded-lg transition cursor-pointer shadow-2xs flex items-center justify-center"
          title="Segarkan Data"
        >
          <i class="fa-solid fa-rotate"></i>
        </button>
      </div>
    </div>

    <!-- Alert -->
    <div v-if="notice" class="p-3 bg-blue-950/60 border border-blue-800 text-blue-300 rounded text-xs flex items-center justify-between">
      <div class="flex items-center gap-2">
        <i class="fa-solid fa-circle-info"></i>
        <span>{{ notice }}</span>
      </div>
      <button @click="notice = ''" class="text-slate-400 hover:text-white">
        <i class="fa-solid fa-xmark"></i>
      </button>
    </div>

    <!-- Filter & Statistics Bar -->
    <div class="bg-slate-900 border border-slate-800 p-4 rounded-xl flex flex-col sm:flex-row sm:items-center justify-between gap-3">
      <div class="relative w-full sm:w-80">
        <span class="absolute inset-y-0 left-0 flex items-center pl-3 text-slate-500 text-xs">
          <i class="fa-solid fa-magnifying-glass"></i>
        </span>
        <input
          v-model="searchQuery"
          type="text"
          placeholder="Cari kode atau nama pelanggaran..."
          class="w-full bg-slate-950 border border-slate-800 text-slate-100 text-xs pl-8 pr-3 py-2 rounded focus:outline-none focus:border-blue-500"
        />
      </div>

      <div class="text-xs text-slate-400 font-medium">
        Menampilkan <span class="font-bold text-white">{{ filteredMaster.length }}</span> dari {{ master.length }} kode terdaftar
      </div>
    </div>

    <!-- Table Card -->
    <div class="bg-slate-900 border border-slate-800 rounded-xl overflow-hidden">
      <div class="overflow-x-auto max-h-[650px]">
        <table class="w-full text-left text-xs">
          <thead class="bg-slate-950/80 sticky top-0 text-slate-400 border-b border-slate-800 font-medium z-10">
            <tr>
              <th class="py-3 px-4 w-32">Kode Pelanggaran</th>
              <th class="py-3 px-4">Deskripsi / Jenis Pelanggaran</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-slate-800/60">
            <tr v-if="filteredMaster.length === 0 && !isLoading">
              <td colspan="2" class="py-10 text-center text-slate-500 italic">
                {{ searchQuery ? 'Tidak ada kode yang cocok dengan pencarian' : 'Belum ada data master pelanggaran' }}
              </td>
            </tr>
            <tr v-for="m in filteredMaster" :key="m.code || m.Code" class="hover:bg-slate-800/40 transition">
              <td class="py-3 px-4 font-mono font-bold text-blue-400 whitespace-nowrap">{{ m.code || m.Code }}</td>
              <td class="py-3 px-4 text-slate-200 leading-relaxed">{{ m.name || m.Name }}</td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>
  </div>
</template>
