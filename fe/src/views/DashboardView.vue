<script setup lang="ts">
import { ref, onMounted } from "vue";
import { api } from "../api";

const stats = ref<any>({
  total_violations: 0,
  pending: 0,
  processing: 0,
  sent: 0,
  failed: 0,
  total_clients: 0,
  total_cameras: 0,
  master_count: 0,
  retention_days: 3,
});

const syncLogs = ref<any[]>([]);
const recentViolations = ref<any[]>([]);
const isSyncing = ref(false);
const isCleaning = ref(false);
const showCleanupConfirm = ref(false);
const alertMessage = ref("");
const alertType = ref<"success" | "error" | "info">("info");

function notify(text: string, type: "success" | "error" | "info" = "info") {
  alertMessage.value = text;
  alertType.value = type;
  if (type === "success") {
    setTimeout(() => {
      if (alertMessage.value === text) alertMessage.value = "";
    }, 5000);
  }
}

async function loadDashboardData() {
  try {
    const [st, sl, vs] = await Promise.allSettled([
      api.dashboardStats(),
      api.syncLogs(15),
      api.violations(""),
    ]);
    if (st.status === "fulfilled") stats.value = st.value;
    if (sl.status === "fulfilled") syncLogs.value = sl.value;
    if (vs.status === "fulfilled") recentViolations.value = ((vs.value?.items || []) as any[]).slice(0, 8);
  } catch (e: any) {
    notify("Gagal memuat data dashboard: " + e.message, "error");
  }
}

async function handleSyncAll() {
  isSyncing.value = true;
  try {
    const res = await api.syncAll();
    notify(res.message || `Sinkronisasi berhasil: ${res.synced} data master diperbarui`, "success");
    await loadDashboardData();
  } catch (e: any) {
    notify("Sinkronisasi gagal: " + e.message, "error");
    await loadDashboardData();
  } finally {
    isSyncing.value = false;
  }
}

async function handleCleanup() {
  showCleanupConfirm.value = false;
  isCleaning.value = true;
  try {
    const res = await api.cleanup(2);
    notify(res.message || `${res.deleted} data pelanggaran berhasil dibersihkan`, "success");
    await loadDashboardData();
  } catch (e: any) {
    notify("Pembersihan data gagal: " + e.message, "error");
  } finally {
    isCleaning.value = false;
  }
}

onMounted(loadDashboardData);
</script>

<template>
  <div class="space-y-6 max-w-7xl mx-auto">
    <!-- Notice / Alert Banner -->
    <div
      v-if="alertMessage"
      :class="[
        alertType === 'success' ? 'bg-emerald-950/60 border-emerald-800 text-emerald-300' :
        alertType === 'error' ? 'bg-rose-950/60 border-rose-800 text-rose-300' :
        'bg-blue-950/60 border-blue-800 text-blue-300'
      ]"
      class="p-3.5 border rounded-lg text-xs flex items-center justify-between"
    >
      <div class="flex items-center gap-2.5">
        <i
          :class="[
            alertType === 'success' ? 'fa-solid fa-circle-check text-emerald-400' :
            alertType === 'error' ? 'fa-solid fa-circle-exclamation text-rose-400' :
            'fa-solid fa-circle-info text-blue-400'
          ]"
        ></i>
        <span>{{ alertMessage }}</span>
      </div>
      <button @click="alertMessage = ''" class="text-slate-400 hover:text-white">
        <i class="fa-solid fa-xmark"></i>
      </button>
    </div>

    <!-- Retention Policy & Control Panel -->
    <div class="bg-slate-900 border border-slate-800 rounded-xl p-5 flex flex-col md:flex-row md:items-center justify-between gap-4">
      <div class="space-y-1">
        <div class="flex items-center gap-2.5">
          <span class="inline-flex items-center gap-1.5 px-2.5 py-1 rounded bg-amber-500/10 text-amber-400 border border-amber-500/20 text-xs font-semibold">
            <i class="fa-solid fa-database"></i>
            <span>Retensi Data: {{ stats.retention_days }} Hari</span>
          </span>
          <span class="text-xs text-slate-400">Data violations disimpan maksimal 3 hari</span>
        </div>
        <p class="text-xs text-slate-400">
          Gunakan perintah CLI <code class="bg-slate-950 px-1.5 py-0.5 rounded border border-slate-800 text-slate-300 font-mono">go run . cleanup --days 2</code> atau tombol pembersihan manual di samping.
        </p>
      </div>

      <div class="flex items-center gap-2.5 shrink-0">
        <!-- 1-Click Sync Polantas -->
        <button
          @click="handleSyncAll"
          :disabled="isSyncing"
          class="bg-blue-600 hover:bg-blue-500 disabled:opacity-50 text-white text-xs font-medium py-2 px-3.5 rounded transition flex items-center gap-2 cursor-pointer shadow-sm"
        >
          <i :class="['fa-solid', isSyncing ? 'fa-circle-notch fa-spin' : 'fa-arrows-rotate']"></i>
          <span>{{ isSyncing ? "Menyinkronkan..." : "Sinkronisasi Polantas" }}</span>
        </button>

        <!-- Cleanup Button -->
        <button
          @click="showCleanupConfirm = true"
          :disabled="isCleaning"
          class="bg-slate-800 hover:bg-rose-950/50 hover:text-rose-300 hover:border-rose-700/50 text-slate-300 border border-slate-700 text-xs font-medium py-2 px-3 rounded transition flex items-center gap-2 cursor-pointer"
        >
          <i :class="['fa-solid', isCleaning ? 'fa-spinner fa-spin' : 'fa-trash-can']"></i>
          <span>{{ isCleaning ? "Membersihkan..." : "Hapus Data > 2 Hari" }}</span>
        </button>

        <!-- Refresh Button -->
        <button
          @click="loadDashboardData"
          class="bg-slate-800 hover:bg-slate-700 text-slate-300 border border-slate-700 text-xs p-2 rounded transition cursor-pointer"
          title="Segarkan Data"
        >
          <i class="fa-solid fa-rotate"></i>
        </button>
      </div>
    </div>

    <!-- KPI Metric Cards Grid -->
    <div class="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-6 gap-3.5">
      <!-- Total Violations -->
      <div class="bg-slate-900 border border-slate-800 p-4 rounded-xl">
        <div class="flex items-center justify-between text-slate-400 mb-2">
          <span class="text-[11px] font-medium uppercase tracking-wider">Total Data</span>
          <i class="fa-solid fa-list-check text-slate-500 text-xs"></i>
        </div>
        <div class="text-2xl font-bold text-white font-mono">{{ stats.total_violations }}</div>
        <span class="text-[10px] text-slate-400 mt-1 block">Semua rekaman aktif</span>
      </div>

      <!-- Pending -->
      <div class="bg-slate-900 border border-slate-800 p-4 rounded-xl">
        <div class="flex items-center justify-between text-amber-400 mb-2">
          <span class="text-[11px] font-medium uppercase tracking-wider">Antrean</span>
          <i class="fa-solid fa-clock text-amber-500/70 text-xs"></i>
        </div>
        <div class="text-2xl font-bold text-amber-300 font-mono">{{ stats.pending }}</div>
        <span class="text-[10px] text-slate-400 mt-1 block">Menunggu pengiriman</span>
      </div>

      <!-- Processing -->
      <div class="bg-slate-900 border border-slate-800 p-4 rounded-xl">
        <div class="flex items-center justify-between text-blue-400 mb-2">
          <span class="text-[11px] font-medium uppercase tracking-wider">Diproses</span>
          <i class="fa-solid fa-arrows-split-up-and-left text-blue-500/70 text-xs"></i>
        </div>
        <div class="text-2xl font-bold text-blue-300 font-mono">{{ stats.processing }}</div>
        <span class="text-[10px] text-slate-400 mt-1 block">Sedang dikirim worker</span>
      </div>

      <!-- Sent -->
      <div class="bg-slate-900 border border-slate-800 p-4 rounded-xl">
        <div class="flex items-center justify-between text-emerald-400 mb-2">
          <span class="text-[11px] font-medium uppercase tracking-wider">Terkirim</span>
          <i class="fa-solid fa-circle-check text-emerald-500/70 text-xs"></i>
        </div>
        <div class="text-2xl font-bold text-emerald-300 font-mono">{{ stats.sent }}</div>
        <span class="text-[10px] text-slate-400 mt-1 block">Diterima server Polri</span>
      </div>

      <!-- Failed -->
      <div class="bg-slate-900 border border-slate-800 p-4 rounded-xl">
        <div class="flex items-center justify-between text-rose-400 mb-2">
          <span class="text-[11px] font-medium uppercase tracking-wider">Gagal</span>
          <i class="fa-solid fa-circle-xmark text-rose-500/70 text-xs"></i>
        </div>
        <div class="text-2xl font-bold text-rose-300 font-mono">{{ stats.failed }}</div>
        <span class="text-[10px] text-slate-400 mt-1 block">Melebihi max retry</span>
      </div>

      <!-- Master Count -->
      <div class="bg-slate-900 border border-slate-800 p-4 rounded-xl">
        <div class="flex items-center justify-between text-slate-300 mb-2">
          <span class="text-[11px] font-medium uppercase tracking-wider">Master Kode</span>
          <i class="fa-solid fa-book-bookmark text-slate-500 text-xs"></i>
        </div>
        <div class="text-2xl font-bold text-slate-100 font-mono">{{ stats.master_count }}</div>
        <span class="text-[10px] text-slate-400 mt-1 block">Kode pelanggaran</span>
      </div>
    </div>

    <!-- Dual Column Section: Sync Logs & Recent Violations -->
    <div class="grid grid-cols-1 lg:grid-cols-2 gap-6">
      <!-- Sync Logs Card -->
      <div class="bg-slate-900 border border-slate-800 rounded-xl overflow-hidden">
        <div class="px-5 py-4 border-b border-slate-800 flex items-center justify-between">
          <div class="flex items-center gap-2">
            <i class="fa-solid fa-clock-rotate-left text-blue-400 text-sm"></i>
            <h3 class="text-sm font-semibold text-white">Log Sinkronisasi Polantas</h3>
          </div>
        </div>

        <div class="overflow-x-auto">
          <table class="w-full text-left text-xs">
            <thead class="bg-slate-950/60 text-slate-400 border-b border-slate-800 font-medium">
              <tr>
                <th class="py-2.5 px-4">Waktu</th>
                <th class="py-2.5 px-3">Sumber / Klien</th>
                <th class="py-2.5 px-3">Status</th>
                <th class="py-2.5 px-3 text-right">Data</th>
                <th class="py-2.5 px-4">Keterangan</th>
              </tr>
            </thead>
            <tbody class="divide-y divide-slate-800/60">
              <tr v-if="syncLogs.length === 0">
                <td colspan="5" class="py-8 text-center text-slate-500 italic">
                  Belum ada catatan aktivitas sinkronisasi
                </td>
              </tr>
              <tr v-for="item in syncLogs" :key="item.id" class="hover:bg-slate-800/40 transition">
                <td class="py-2.5 px-4 text-slate-400 font-mono whitespace-nowrap">
                  {{ new Date(item.created_at).toLocaleTimeString() }}
                </td>
                <td class="py-2.5 px-3 font-medium text-slate-200">{{ item.client_name }}</td>
                <td class="py-2.5 px-3 whitespace-nowrap">
                  <span
                    v-if="item.status === 'success'"
                    class="inline-flex items-center gap-1 text-[11px] font-medium text-emerald-400 bg-emerald-950/40 px-2 py-0.5 rounded border border-emerald-800/40"
                  >
                    <i class="fa-solid fa-check text-[10px]"></i>
                    <span>Sukses</span>
                  </span>
                  <span
                    v-else
                    class="inline-flex items-center gap-1 text-[11px] font-medium text-rose-400 bg-rose-950/40 px-2 py-0.5 rounded border border-rose-800/40"
                  >
                    <i class="fa-solid fa-xmark text-[10px]"></i>
                    <span>Gagal</span>
                  </span>
                </td>
                <td class="py-2.5 px-3 text-right font-mono text-slate-200">{{ item.records_synced }}</td>
                <td class="py-2.5 px-4 text-slate-400 truncate max-w-[160px]" :title="item.message">
                  {{ item.message }}
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>

      <!-- Recent Violations Card -->
      <div class="bg-slate-900 border border-slate-800 rounded-xl overflow-hidden">
        <div class="px-5 py-4 border-b border-slate-800 flex items-center justify-between">
          <div class="flex items-center gap-2">
            <i class="fa-solid fa-triangle-exclamation text-amber-400 text-sm"></i>
            <h3 class="text-sm font-semibold text-white">Pelanggaran Terkini</h3>
          </div>
          <router-link to="/violations" class="text-xs text-blue-400 hover:text-blue-300 flex items-center gap-1">
            <span>Lihat Semua</span>
            <i class="fa-solid fa-angle-right text-[10px]"></i>
          </router-link>
        </div>

        <div class="overflow-x-auto">
          <table class="w-full text-left text-xs">
            <thead class="bg-slate-950/60 text-slate-400 border-b border-slate-800 font-medium">
              <tr>
                <th class="py-2.5 px-4">ID</th>
                <th class="py-2.5 px-3">Plat Nomor</th>
                <th class="py-2.5 px-3">Kode Pelanggaran</th>
                <th class="py-2.5 px-3">Status</th>
                <th class="py-2.5 px-4 text-right">Percobaan</th>
              </tr>
            </thead>
            <tbody class="divide-y divide-slate-800/60">
              <tr v-if="recentViolations.length === 0">
                <td colspan="5" class="py-8 text-center text-slate-500 italic">
                  Belum ada data pelanggaran tercatat
                </td>
              </tr>
              <tr v-for="v in recentViolations" :key="v.id" class="hover:bg-slate-800/40 transition">
                <td class="py-2.5 px-4 font-mono text-slate-400">#{{ v.id }}</td>
                <td class="py-2.5 px-3 font-mono font-bold text-white">{{ v.plate }}</td>
                <td class="py-2.5 px-3 text-slate-300">{{ v.violation_code }}</td>
                <td class="py-2.5 px-3 whitespace-nowrap">
                  <span
                    :class="[
                      v.status === 'sent' ? 'bg-emerald-950/40 text-emerald-400 border-emerald-800/40' :
                      v.status === 'pending' ? 'bg-amber-950/40 text-amber-400 border-amber-800/40' :
                      v.status === 'processing' ? 'bg-blue-950/40 text-blue-400 border-blue-800/40' :
                      'bg-rose-950/40 text-rose-400 border-rose-800/40'
                    ]"
                    class="px-2 py-0.5 rounded text-[10px] font-semibold border inline-block"
                  >
                    {{ v.status }}
                  </span>
                </td>
                <td class="py-2.5 px-4 text-right font-mono text-slate-400">{{ v.attempts }}x</td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>
    </div>

    <!-- Modal Konfirmasi Hapus Data Lama -->
    <Teleport to="body">
      <div
        v-if="showCleanupConfirm"
        class="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/60 backdrop-blur-xs transition-opacity"
      >
        <div class="bg-slate-900 border border-slate-800 rounded-2xl shadow-2xl max-w-sm w-full p-6 text-slate-100">
          <div class="flex items-center gap-3.5 mb-4">
            <div class="w-11 h-11 rounded-xl bg-rose-500/10 border border-rose-500/30 flex items-center justify-center text-rose-500 shrink-0">
              <i class="fa-solid fa-trash-can text-lg"></i>
            </div>
            <div>
              <h3 class="text-sm font-bold text-white tracking-wide">Hapus Data Lama</h3>
              <p class="text-xs text-slate-400">Konfirmasi Pembersihan Data</p>
            </div>
          </div>

          <p class="text-xs text-slate-300 mb-3 leading-relaxed">
            Apakah Anda yakin ingin menghapus semua data pelanggaran yang berumur lebih dari
            <strong class="text-white">2 hari</strong>?
          </p>
          <div class="p-2.5 rounded-lg bg-amber-500/10 border border-amber-500/20 text-amber-500 text-[11px] mb-5">
            <i class="fa-solid fa-circle-info mr-1"></i>
            <span>Data beserta file media (XML/gambar) yang lewat masa retensi akan dihapus permanen.</span>
          </div>

          <div class="flex items-center justify-end gap-2.5">
            <button
              @click="showCleanupConfirm = false"
              :disabled="isCleaning"
              class="px-4 py-2 text-xs font-medium text-slate-700 dark:text-slate-300 bg-slate-100 dark:bg-slate-800 hover:bg-slate-200 dark:hover:bg-slate-700 rounded-lg transition cursor-pointer"
            >
              Batal
            </button>
            <button
              @click="handleCleanup"
              :disabled="isCleaning"
              class="px-4 py-2 text-xs font-semibold text-white bg-rose-600 hover:bg-rose-500 rounded-lg transition flex items-center gap-2 cursor-pointer shadow-sm disabled:opacity-50"
            >
              <i v-if="isCleaning" class="fa-solid fa-circle-notch fa-spin text-xs"></i>
              <i v-else class="fa-solid fa-trash text-xs"></i>
              <span>{{ isCleaning ? "Menghapus..." : "Ya, Hapus Data" }}</span>
            </button>
          </div>
        </div>
      </div>
    </Teleport>
  </div>
</template>
