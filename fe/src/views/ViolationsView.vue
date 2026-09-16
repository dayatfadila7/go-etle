<script setup lang="ts">
import { ref, onMounted, computed } from "vue";
import { api } from "../api";

const violations = ref<any[]>([]);
const statusFilter = ref("");
const searchPlate = ref("");
const isLoading = ref(false);
const notice = ref("");

async function loadViolations() {
  isLoading.value = true;
  try {
    violations.value = await api.violations(statusFilter.value);
  } catch (e: any) {
    notice.value = "Gagal memuat data pelanggaran: " + e.message;
  } finally {
    isLoading.value = false;
  }
}

const filteredViolations = computed(() => {
  const q = searchPlate.value.trim().toUpperCase();
  if (!q) return violations.value;
  return violations.value.filter((v) => v.plate && v.plate.toUpperCase().includes(q));
});

onMounted(loadViolations);
</script>

<template>
  <div class="space-y-6 max-w-7xl mx-auto">
    <!-- Header -->
    <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
      <div>
        <h2 class="text-lg font-bold text-white tracking-wide">Data Pelanggaran (Violations)</h2>
        <p class="text-xs text-slate-400">Antrean rekaman pelanggaran kendaraan dan status pengiriman ke Korlantas Polri</p>
      </div>

      <button
        @click="loadViolations"
        class="bg-white hover:bg-slate-100 text-slate-700 border border-slate-300 dark:bg-slate-800 dark:hover:bg-slate-700 dark:text-slate-300 dark:border-slate-700 text-xs py-2 px-3.5 rounded-lg transition flex items-center gap-2 self-start sm:self-auto cursor-pointer shadow-2xs font-medium"
        title="Segarkan Data"
      >
        <i :class="['fa-solid fa-rotate', isLoading ? 'fa-spin' : '']"></i>
        <span>Segarkan</span>
      </button>
    </div>

    <!-- Alert -->
    <div v-if="notice" class="p-3 bg-rose-950/60 border border-rose-800 text-rose-300 rounded text-xs flex items-center justify-between">
      <div class="flex items-center gap-2">
        <i class="fa-solid fa-circle-exclamation"></i>
        <span>{{ notice }}</span>
      </div>
      <button @click="notice = ''" class="text-slate-400 hover:text-white">
        <i class="fa-solid fa-xmark"></i>
      </button>
    </div>

    <!-- Filter Bar -->
    <div class="bg-slate-900 border border-slate-800 p-4 rounded-xl flex flex-col sm:flex-row sm:items-center justify-between gap-3">
      <div class="flex flex-wrap items-center gap-3">
        <!-- Search Plate -->
        <div class="relative w-56">
          <span class="absolute inset-y-0 left-0 flex items-center pl-3 text-slate-500 text-xs">
            <i class="fa-solid fa-magnifying-glass"></i>
          </span>
          <input
            v-model="searchPlate"
            placeholder="Cari Plat Nomor..."
            class="w-full bg-slate-950 border border-slate-800 text-slate-100 text-xs pl-8 pr-3 py-2 rounded focus:outline-none focus:border-blue-500 uppercase"
          />
        </div>

        <!-- Filter Status Dropdown -->
        <div class="flex items-center gap-2 text-xs text-slate-300">
          <span>Status:</span>
          <select
            v-model="statusFilter"
            @change="loadViolations"
            class="bg-slate-950 border border-slate-800 text-slate-200 text-xs px-3 py-2 rounded focus:outline-none focus:border-blue-500"
          >
            <option value="">Semua Status</option>
            <option value="pending">Pending (Antrean)</option>
            <option value="processing">Processing (Sedang Dikirim)</option>
            <option value="sent">Sent (Terkirim Sukses)</option>
            <option value="failed">Failed (Gagal)</option>
          </select>
        </div>
      </div>

      <div class="text-xs text-slate-400 font-medium">
        Total: <span class="font-bold text-white font-mono">{{ filteredViolations.length }}</span> data
      </div>
    </div>

    <!-- Table Card -->
    <div class="bg-slate-900 border border-slate-800 rounded-xl overflow-hidden">
      <div class="overflow-x-auto">
        <table class="w-full text-left text-xs">
          <thead class="bg-slate-950/70 text-slate-400 border-b border-slate-800 font-medium">
            <tr>
              <th class="py-3 px-4">ID</th>
              <th class="py-3 px-4">Plat Nomor</th>
              <th class="py-3 px-4">Perangkat Kamera</th>
              <th class="py-3 px-4">Kode & Pelanggaran</th>
              <th class="py-3 px-4">Status Pengiriman</th>
              <th class="py-3 px-4">Respons ETLE</th>
              <th class="py-3 px-4 text-right">Percobaan</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-slate-800/60">
            <tr v-if="filteredViolations.length === 0 && !isLoading">
              <td colspan="7" class="py-10 text-center text-slate-500 italic">
                Belum ada data pelanggaran untuk filter ini
              </td>
            </tr>
            <tr v-for="v in filteredViolations" :key="v.id" class="hover:bg-slate-800/40 transition">
              <td class="py-3 px-4 font-mono text-slate-400">#{{ v.id }}</td>
              <td class="py-3 px-4 font-mono font-bold text-white text-sm tracking-wider">{{ v.plate }}</td>
              <td class="py-3 px-4 text-slate-300 font-mono">{{ v.device_name || "—" }}</td>
              <td class="py-3 px-4">
                <span class="font-mono font-semibold text-blue-400 block">{{ v.violation_code }}</span>
                <span class="text-slate-400 text-[11px] block truncate max-w-[220px]">{{ v.violation_name || "—" }}</span>
              </td>
              <td class="py-3 px-4 whitespace-nowrap">
                <span
                  :class="[
                    v.status === 'sent' ? 'bg-emerald-950/40 text-emerald-400 border-emerald-800/40' :
                    v.status === 'pending' ? 'bg-amber-950/40 text-amber-400 border-amber-800/40' :
                    v.status === 'processing' ? 'bg-blue-950/40 text-blue-400 border-blue-800/40' :
                    'bg-rose-950/40 text-rose-400 border-rose-800/40'
                  ]"
                  class="inline-flex items-center gap-1.5 px-2.5 py-0.5 rounded text-[11px] font-semibold border"
                >
                  <i
                    :class="[
                      v.status === 'sent' ? 'fa-solid fa-circle-check text-[10px]' :
                      v.status === 'pending' ? 'fa-solid fa-clock text-[10px]' :
                      v.status === 'processing' ? 'fa-solid fa-arrows-rotate fa-spin text-[10px]' :
                      'fa-solid fa-circle-xmark text-[10px]'
                    ]"
                  ></i>
                  <span class="capitalize">{{ v.status }}</span>
                </span>
              </td>
              <td class="py-3 px-4 font-mono text-slate-400">
                <span v-if="v.response_status" class="bg-slate-950 px-2 py-0.5 rounded border border-slate-800">
                  HTTP {{ v.response_status }}
                </span>
                <span v-else class="text-slate-600">—</span>
              </td>
              <td class="py-3 px-4 text-right font-mono text-slate-400">{{ v.attempts }}x</td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>
  </div>
</template>
