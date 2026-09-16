<script setup lang="ts">
import { ref, onMounted, onBeforeUnmount, computed } from "vue";
import { api } from "../api";

const tab = ref<"delivery" | "system" | "db">("delivery");
const data = ref<any>(null);
const isLoading = ref(false);
const notice = ref("");
let timer: number | undefined;

const labels: Record<string, string> = {
  time: "Waktu",
  result: "Hasil",
  id: "ID",
  plate: "Plat",
  device: "Perangkat",
  attempt: "Coba",
  violation_code: "Kode",
  capture_time_ms: "Capture(ms)",
  http_status: "HTTP",
  etle_status: "Status ETLE",
  error: "Keterangan",
  duration_ms: "Durasi(ms)",
  load1: "Load 1m",
  load5: "Load 5m",
  cpu_pct_goetle: "CPU go-etle %",
  rss_mb_goetle: "RAM go-etle (MB)",
  mem_used_pct: "RAM %",
  disk_used_pct: "Disk %",
  queue_pending: "Antrean",
  queue_failed: "Gagal",
  sent_total: "Terkirim",
  total_rows: "Total Data",
  conns: "Koneksi",
  xact_rollback: "Rollback TX",
  tup_inserted: "Insert",
  tup_updated: "Update",
  tup_deleted: "Delete",
  deadlocks: "Deadlock",
  cache_hit_pct: "Cache Hit %",
  db_size_mb: "Ukuran DB (MB)",
};

const tabs = [
  { key: "delivery" as const, label: "Log Pengiriman", icon: "fa-solid fa-paper-plane" },
  { key: "system" as const, label: "Kinerja Sistem", icon: "fa-solid fa-microchip" },
  { key: "db" as const, label: "Kinerja Database", icon: "fa-solid fa-database" },
];

function colHeader(k: string): string {
  return labels[k] || k;
}

function displayVal(row: any, k: string): string {
  const v = row[k];
  if (v === undefined || v === null || v === "") return "—";
  if (k === "capture_time_ms" && v !== "—" && v !== "0") {
    const d = new Date(Number(v));
    if (!isNaN(d.getTime())) {
      return d.toLocaleString("id-ID", {
        day: "2-digit",
        month: "2-digit",
        hour: "2-digit",
        minute: "2-digit",
        second: "2-digit",
      });
    }
  }
  return v;
}

function rowsFor(list: any[]): any[] {
  return (list || []).slice().reverse();
}

function resultBadgeClass(r: string): string {
  switch (r) {
    case "sent":
      return "bg-emerald-500/10 text-emerald-400 border-emerald-500/30";
    case "failed":
      return "bg-rose-500/10 text-rose-400 border-rose-500/30";
    case "expired":
    case "skipped_unknown":
      return "bg-amber-500/10 text-amber-400 border-amber-500/30";
    default:
      return "bg-slate-500/10 text-slate-400 border-slate-500/30";
  }
}

function currentList(): any[] {
  if (tab.value === "delivery") return rowsFor(data.value?.delivery);
  if (tab.value === "system") return rowsFor(data.value?.system);
  return rowsFor(data.value?.db);
}

function currentKeys(): string[] {
  const list = currentList();
  return list.length ? Object.keys(list[0]) : [];
}

function fileOf(): string {
  if (tab.value === "delivery") return data.value?.delivery_file || "";
  if (tab.value === "system") return data.value?.system_file || "";
  return data.value?.db_file || "";
}

async function load() {
  isLoading.value = true;
  try {
    data.value = await api.monitoring();
    notice.value = "";
  } catch (e: any) {
    notice.value = "Gagal memuat monitoring: " + e.message;
  } finally {
    isLoading.value = false;
  }
}

onMounted(() => {
  load();
  timer = window.setInterval(load, 60000);
});
onBeforeUnmount(() => {
  if (timer) window.clearInterval(timer);
});
</script>

<template>
  <div class="space-y-4">
    <div class="flex items-center justify-between gap-3">
      <div class="flex items-center gap-2 bg-slate-950 border border-slate-800 rounded-lg p-1 text-xs">
        <button
          v-for="t in tabs"
          :key="t.key"
          type="button"
          @click="tab = t.key"
          class="px-3 py-1.5 rounded flex items-center gap-1.5 font-medium transition cursor-pointer"
          :class="
            tab === t.key
              ? 'bg-blue-600 text-white shadow-xs font-semibold'
              : 'text-slate-400 hover:text-slate-200'
          "
        >
          <i :class="[t.icon, 'text-xs']"></i>
          <span>{{ t.label }}</span>
        </button>
      </div>
      <button
        type="button"
        @click="load"
        :disabled="isLoading"
        class="px-3 py-1.5 rounded text-xs font-medium bg-slate-800 hover:bg-slate-700 text-slate-300 border border-slate-700 transition flex items-center gap-1.5 cursor-pointer disabled:opacity-50"
      >
        <i :class="isLoading ? 'fa-solid fa-circle-notch fa-spin' : 'fa-solid fa-rotate'" class="text-xs"></i>
        <span>{{ isLoading ? "Muat..." : "Segarkan (60 dtk)" }}</span>
      </button>
    </div>

    <p v-if="notice" class="text-xs text-rose-400 bg-rose-500/10 border border-rose-500/30 rounded-lg px-3 py-2">{{ notice }}</p>

    <div class="bg-slate-900 border border-slate-800 rounded-xl overflow-hidden">
      <div class="border-b border-slate-800 px-4 py-3 flex items-center justify-between">
        <div class="flex items-center gap-2 text-xs">
          <i class="fa-solid fa-file-lines text-blue-400"></i>
          <span class="text-slate-200 font-semibold">{{ tabs.find((t) => t.key === tab)?.label }}</span>
          <span v-if="currentList().length" class="text-slate-500 font-mono">{{ currentList().length }} baris</span>
        </div>
        <span class="text-[10px] text-slate-500 font-mono truncate max-w-[45%]" :title="fileOf()">{{ fileOf() }}</span>
      </div>

      <div class="overflow-x-auto max-h-[620px] overflow-y-auto">
        <table class="w-full text-left text-xs border-collapse min-w-[820px]">
          <thead class="sticky top-0 bg-slate-950 z-10">
            <tr class="text-slate-400">
              <th
                v-for="k in currentKeys()"
                :key="k"
                class="px-3 py-2 font-semibold whitespace-nowrap text-[11px] uppercase tracking-wide border-b border-slate-800"
              >
                {{ colHeader(k) }}
              </th>
            </tr>
          </thead>
          <tbody>
            <tr v-if="!currentList().length" class="text-center text-slate-500">
              <td :colspan="Math.max(1, currentKeys().length)" class="px-3 py-10">
                <i class="fa-solid fa-hourglass-half mr-2"></i>
                Belum ada baris data{{ tab === "delivery" ? " — log pengisian terisi otomatis saat data terkirim" : "" }}.
              </td>
            </tr>
            <tr
              v-for="(row, i) in currentList()"
              :key="tab + '-' + i"
              class="border-b border-slate-800/70 hover:bg-slate-800/40"
            >
              <td
                v-for="k in currentKeys()"
                :key="k"
                class="px-3 py-2 whitespace-nowrap text-slate-300 font-mono align-top"
              >
                <span
                  v-if="k === 'result'"
                  class="px-2 py-0.5 rounded-full border text-[10px] font-semibold inline-block"
                  :class="resultBadgeClass(row[k])"
                >
                  {{ displayVal(row, k) }}
                </span>
                <template v-else-if="k === 'error'">
                  <span class="text-rose-300/90">{{ displayVal(row, k) }}</span>
                </template>
                <template v-else>{{ displayVal(row, k) }}</template>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>
  </div>
</template>