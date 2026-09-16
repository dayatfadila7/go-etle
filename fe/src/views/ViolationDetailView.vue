<script setup lang="ts">
import { ref, onMounted, computed, onBeforeUnmount } from "vue";
import { useRoute } from "vue-router";
import { api } from "../api";

const route = useRoute();
const id = computed(() => String(route.params.id));
const v = ref<any>(null);
const isLoading = ref(true);
const notice = ref("");

function fmtDT(val: any): string {
  if (!val) return "—";
  const d = new Date(val);
  if (isNaN(d.getTime())) return "—";
  return d.toLocaleString("id-ID", {
    day: "2-digit",
    month: "short",
    year: "numeric",
    hour: "2-digit",
    minute: "2-digit",
    second: "2-digit",
  });
}

function fmtEpoch(ms: any): string {
  if (!ms || Number(ms) <= 0) return "—";
  return fmtDT(ms);
}

function trType(t: any): string {
  const raw = String(t || "");
  switch (raw.toLowerCase()) {
    case "two wheel":
    case "two-wheel":
    case "twowheel":
    case "twowheelvehicle":
      return "Roda Dua";
    case "buggy":
      return "Mobil";
    case "suvmpv":
      return "SUV / MPV";
    case "sedan":
      return "Sedan";
    case "pickup":
      return "Pick Up";
    case "truck":
      return "Truk";
    case "bus":
      return "Bus";
    case "van":
      return "Van";
    default:
      return raw || "—";
  }
}

function trColor(c: any): string {
  const raw = String(c || "");
  switch (raw.toLowerCase()) {
    case "unknown":
    case "":
      return "Tidak Diketahui";
    case "black":
      return "Hitam";
    case "white":
      return "Putih";
    case "silver":
      return "Silver";
    case "red":
      return "Merah";
    default:
      return raw || "—";
  }
}

function trViolation(n: any): string {
  const raw = String(n || "");
  switch (raw.toLowerCase()) {
    case "no_seatbelt_fastened":
    case "no seatbelt":
    case "no-seatbelt":
      return "Tidak Menggunakan Sabuk Pengaman";
    case "front_passenger_not_buckled_up":
      return "Penumpang Depan Tidak Mengenakan Sabuk";
    case "riding_without_helmet":
    case "no_helmet":
    case "non-helmet":
      return "Tidak Menggunakan Helm";
    case "driving_while_using_phone":
    case "using_phone_while_driving":
      return "Menggunakan Telepon Saat Berkendara";
    case "running_red_light":
    case "red_light_running":
      return "Menerobos Lampu Merah";
    default:
      return raw || "—";
  }
}

const statusMeta = computed(() => {
  const s = v.value?.status;
  switch (s) {
    case "sent":
      return { label: "Terkirim", cls: "bg-emerald-50 text-emerald-600 border-emerald-200 dark:bg-emerald-500/10 dark:text-emerald-400 dark:border-emerald-500/40", icon: "fa-solid fa-circle-check" };
    case "pending":
      return { label: "Antrean", cls: "bg-amber-50 text-amber-600 border-amber-200 dark:bg-amber-500/10 dark:text-amber-400 dark:border-amber-500/40", icon: "fa-solid fa-clock" };
    case "processing":
      return { label: "Sedang Dikirim", cls: "bg-blue-50 text-blue-600 border-blue-200 dark:bg-blue-500/10 dark:text-blue-400 dark:border-blue-500/40", icon: "fa-solid fa-arrows-rotate fa-spin" };
    default:
      return { label: "Gagal", cls: "bg-rose-50 text-rose-600 border-rose-200 dark:bg-rose-500/10 dark:text-rose-400 dark:border-rose-500/40", icon: "fa-solid fa-circle-xmark" };
  }
});

const cameraLabel = computed(() => v.value?.camera_name || v.value?.device_name || "—");
const imageUrl = computed(() => v.value?.vehicle_image_url || v.value?.plate_image_url || "");
const hasVideo = computed(() => !!v.value?.video_url);

const metaList = computed(() => [
  { k: "No. Plat", val: () => v.value?.plate || "—", mono: true, hl: true, plate: true },
  {
    k: "Status",
    val: () => statusMeta.value.label,
    chip: true,
    chipCls: statusMeta.value.cls,
  },
  { k: "Respons ETLE", val: () => v.value?.response_status || "—", mono: true },
  { k: "Polres", val: () => v.value?.client_name || "—" },
  { k: "ID Klien", val: () => v.value?.client_code || "—", mono: true },
  { k: "Perangkat Kamera", val: () => v.value?.device_name || "—", mono: true },
  { k: "Kode Kamera", val: () => v.value?.camera_code || "—", mono: true },
  { k: "Nama Kamera", val: () => v.value?.camera_name || "—" },
  { k: "Lokasi", val: () => v.value?.location_name || "—" },
  { k: "Tipe Kendaraan", val: () => trType(v.value?.vehicle_type), tooltip: () => v.value?.vehicle_type },
  { k: "Warna Kendaraan", val: () => trColor(v.value?.vehicle_color), tooltip: () => v.value?.vehicle_color },
  { k: "Warna Plat", val: () => trColor(v.value?.plate_color), tooltip: () => v.value?.plate_color },
  { k: "Kode Pelanggaran", val: () => v.value?.violation_code || "—", mono: true, hl: true },
  { k: "Pelanggaran", val: () => trViolation(v.value?.violation_name), hl: true, tooltip: () => v.value?.violation_name },
  { k: "Percobaan Kirim", val: () => (v.value?.attempts ?? 0) + "x" },
]);

const steps = computed(() => [
  {
    no: 1,
    label: "Waktu Tangkap",
    note: "Terekam oleh kamera CCTV ANPR",
    val: `${fmtEpoch(v.value?.capture_time)}`,
    cls: "bg-amber-400 text-amber-950 border-amber-300 shadow-[0_0_10px_rgba(251,191,36,0.45)]",
  },
  {
    no: 2,
    label: "Diterima Sistem",
    note: "Masuk ke gateway integrasi",
    val: fmtDT(v.value?.created_at),
    cls: "bg-blue-600 text-white border-blue-400 shadow-[0_0_10px_rgba(59,130,246,0.5)]",
  },
  {
    no: 3,
    label: "Respons ETLE",
    note: "Balasan dari server Korlantas",
    val: fmtDT(v.value?.sent_at),
    cls: "bg-emerald-500 text-white border-emerald-400 shadow-[0_0_10px_rgba(16,185,129,0.5)]",
  },
]);

async function load() {
  isLoading.value = true;
  try {
    v.value = await api.violationDetail(id.value);
    notice.value = "";
  } catch (e: any) {
    notice.value = "Gagal memuat detail pelanggaran: " + e.message;
  } finally {
    isLoading.value = false;
  }
}

onMounted(load);

// ---------- viewer gambar: zoom & geser (drag-pan) ----------
const previewSrc = ref("");
const scale = ref(1);
const tx = ref(0);
const ty = ref(0);
const dragging = ref(false);
let startX = 0, startY = 0, sTX = 0, sTY = 0;

function openPreview(src: string) {
  if (!src) return;
  previewSrc.value = src;
  resetView();
}

function clampScale(s: number) {
  return Math.min(8, Math.max(1, s));
}

function resetView() {
  scale.value = 1;
  tx.value = 0;
  ty.value = 0;
}

function zoomBy(factor: number) {
  const ns = clampScale(scale.value * factor);
  const k = ns / scale.value - 1;
  tx.value -= 0;
  ty.value -= 0;
  scale.value = ns;
}

function onWheel(e: WheelEvent) {
  const step = e.deltaY < 0 ? 1.18 : 1 / 1.18;
  const cur = scale.value;
  const ns = clampScale(cur * step);
  const k = ns / cur - 1;
  const rect = (e.currentTarget as HTMLElement).getBoundingClientRect();
  const mx = e.clientX - rect.left - rect.width / 2;
  const my = e.clientY - rect.top - rect.height / 2;
  tx.value -= mx * k;
  ty.value -= my * k;
  scale.value = ns;
}

function onPointerDown(e: PointerEvent) {
  dragging.value = true;
  startX = e.clientX;
  startY = e.clientY;
  sTX = tx.value;
  sTY = ty.value;
  (e.currentTarget as HTMLElement).setPointerCapture(e.pointerId);
}

function onPointerMove(e: PointerEvent) {
  if (!dragging.value) return;
  tx.value = sTX + (e.clientX - startX);
  ty.value = sTY + (e.clientY - startY);
}

function onPointerUp() {
  dragging.value = false;
}

function closePreview() {
  previewSrc.value = "";
}

function onKeydown(e: KeyboardEvent) {
  if (e.key === "Escape") closePreview();
}

onBeforeUnmount(() => {
  window.removeEventListener("keydown", onKeydown);
});
</script>

<template>
  <div class="space-y-5 max-w-6xl mx-auto">
    <!-- Back + title -->
    <div class="flex items-center justify-between gap-3">
      <div class="flex items-center gap-3">
        <router-link
          to="/violations"
          class="w-9 h-9 rounded-lg bg-white dark:bg-slate-800 border border-slate-300 dark:border-slate-700 hover:bg-slate-100 dark:hover:bg-slate-700 text-slate-600 dark:text-slate-300 flex items-center justify-center transition cursor-pointer shadow-xs dark:shadow-none"
          title="Kembali ke daftar"
        >
          <i class="fa-solid fa-arrow-left text-sm"></i>
        </router-link>
        <div>
          <div class="flex items-center gap-2">
            <h2 class="text-lg font-bold text-slate-800 dark:text-white tracking-wide">Detail Pelanggaran</h2>
            <span class="font-mono text-slate-500 dark:text-slate-500 text-xs bg-slate-100 dark:bg-slate-900 border border-slate-300 dark:border-slate-800 px-2 py-0.5 rounded">#{{ id }}</span>
          </div>
          <p class="text-xs text-slate-500 dark:text-slate-400">Bukti rekaman kamera ANPR terintegrasi Korlantas Polri</p>
        </div>
      </div>
    </div>

    <p v-if="notice" class="text-xs text-rose-600 dark:text-rose-400 bg-rose-50 dark:bg-rose-500/10 border border-rose-200 dark:border-rose-500/30 rounded-lg px-3 py-2">{{ notice }}</p>

    <div v-if="isLoading" class="text-center py-16 text-slate-400">
      <i class="fa-solid fa-circle-notch fa-spin text-2xl"></i>
      <p class="text-xs mt-3">Memuat bukti pelanggaran...</p>
    </div>

    <template v-else-if="v">
      <div class="grid grid-cols-1 lg:grid-cols-[1fr_330px] gap-6 items-start">
        <!-- KOLOM KIRI: bukti kamera + linimasa -->
        <div class="space-y-5 min-w-0">
          <!-- Bukti rekaman -->
          <div class="relative bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 rounded-2xl p-6 overflow-hidden shadow-sm dark:shadow-none">
            <div class="absolute inset-x-0 top-0 h-px bg-gradient-to-r from-transparent via-blue-500/60 to-transparent"></div>
            <p class="text-[10px] font-semibold text-slate-500 dark:text-slate-400 uppercase tracking-widest mb-3 flex items-center justify-between">
              <span class="flex items-center gap-1.5"><i class="fa-solid fa-camera text-blue-500 dark:text-blue-400"></i> Bukti Rekaman · {{ cameraLabel }}</span>
              <span class="text-slate-400 dark:text-slate-500 font-mono flex items-center gap-1.5"><i class="fa-solid fa-magnifying-glass-plus"></i> Klik untuk perbesar</span>
            </p>

            <button
              type="button"
              @click="openPreview(imageUrl)"
              :disabled="!imageUrl"
              class="relative w-full rounded-xl overflow-hidden bg-black border border-slate-800 aspect-video scan-surface cursor-zoom-in disabled:cursor-not-allowed group"
            >
              <img v-if="imageUrl" :src="imageUrl" alt="Foto bukti kamera" class="w-full h-full object-cover transition-transform duration-500 group-hover:scale-[1.02]" />
              <div v-else class="w-full h-full flex items-center justify-center text-slate-600">
                <i class="fa-solid fa-image text-3xl"></i>
              </div>
              <span class="corner tl"></span><span class="corner tr"></span>
              <span class="corner bl"></span><span class="corner br"></span>
              <div class="absolute top-2.5 left-3 flex items-center gap-1.5 text-[10px] font-mono bg-black/60 backdrop-blur px-2 py-1 rounded border border-white/10 text-emerald-300">
                <span class="w-1.5 h-1.5 rounded-full bg-red-500 animate-pulse"></span> REC
                <span class="text-slate-300">· {{ fmtEpoch(v.capture_time) }}</span>
              </div>
              <div class="absolute inset-x-0 bottom-0 h-14 bg-gradient-to-t from-black/70 to-transparent pointer-events-none"></div>
              <span class="absolute bottom-3 left-3 text-xs font-mono font-bold tracking-widest text-white flex items-center gap-2">
                <span class="bg-black/55 backdrop-blur px-2 py-0.5 rounded border border-white/10">{{ v.plate || "—" }}</span>
                <span class="text-[10px] text-slate-200 bg-black/45 backdrop-blur px-2 py-0.5 rounded border border-white/10">{{ v.violation_code }}</span>
              </span>
              <span class="absolute bottom-3 right-3 w-9 h-9 rounded-lg bg-black/50 backdrop-blur border border-white/15 flex items-center justify-center text-slate-200 opacity-0 group-hover:opacity-100 transition">
                <i class="fa-solid fa-magnifying-glass-plus"></i>
              </span>
            </button>
          </div>

          <!-- Linimasa Pengiriman -->
          <div class="relative bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 rounded-2xl p-6 shadow-sm dark:shadow-none">
            <div class="absolute inset-x-0 top-0 h-px bg-gradient-to-r from-transparent via-emerald-500/60 to-transparent"></div>
            <p class="text-[10px] font-semibold text-slate-500 dark:text-slate-400 uppercase tracking-widest mb-4 flex items-center gap-1.5">
              <i class="fa-solid fa-timeline text-emerald-500 dark:text-emerald-400"></i> Linimasa Pengiriman
            </p>
            <div class="relative">
              <div class="absolute left-[15px] top-2 bottom-2 w-px bg-slate-200 dark:bg-slate-800"></div>
              <div class="space-y-5">
                <div v-for="s in steps" :key="s.no" class="relative flex items-start gap-4">
                  <span class="relative z-10 w-8 h-8 shrink-0 rounded-full border flex items-center justify-center text-xs font-bold" :class="s.cls">
                    {{ s.no }}
                  </span>
                  <div class="pt-1 min-w-0">
                    <p class="text-sm font-semibold text-slate-800 dark:text-white">{{ s.label }}</p>
                    <p class="text-[10px] text-slate-500 dark:text-slate-500 mb-0.5">{{ s.note }}</p>
                    <p class="text-xs font-mono text-slate-600 dark:text-slate-300">{{ s.val }}</p>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>

        <!-- KOLOM KANAN: sidebar data pelanggaran -->
        <aside class="relative bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 rounded-2xl overflow-hidden lg:sticky lg:top-8 shadow-sm dark:shadow-none">
          <div class="absolute inset-x-0 top-0 h-px bg-gradient-to-r from-transparent via-blue-500/60 to-transparent"></div>

          <div class="px-5 pt-5 pb-3 flex items-center justify-between">
            <p class="text-[10px] font-bold text-slate-600 dark:text-slate-300 uppercase tracking-widest flex items-center gap-1.5">
              <i class="fa-solid fa-list-ul text-blue-500 dark:text-blue-400"></i> Data Pelanggaran
            </p>
            <span class="font-mono text-[10px] text-slate-400 dark:text-slate-500">ID #{{ id }}</span>
          </div>

          <div class="pb-4">
            <div class="space-y-0">
              <div
                v-for="m in metaList"
                :key="m.k"
                class="px-5 py-2 flex items-start justify-between gap-3"
                :class="m.plate ? 'bg-blue-50/70 dark:bg-blue-500/10 border-y border-blue-100 dark:border-blue-500/20' : ''"
              >
                <span class="text-[10px] text-slate-500 dark:text-slate-400 uppercase tracking-wide pt-0.5 shrink-0">{{ m.k }}</span>
                <span class="text-right text-xs text-slate-800 dark:text-slate-100 break-words min-w-0"
                  :class="[m.mono ? 'font-mono' : '', m.hl && !m.plate ? 'text-blue-500 dark:text-blue-400 font-semibold' : '', m.plate ? 'font-mono font-bold text-blue-600 dark:text-blue-400' : '']"
                  :title="m.tooltip ? m.tooltip() : undefined"
                >
                  {{ m.val() }}
                </span>
              </div>
            </div>
          </div>

          <div class="px-5 py-3 bg-slate-50 dark:bg-slate-950/60 border-t border-slate-100 dark:border-slate-800">
            <p class="text-[10px] text-slate-500 dark:text-slate-400 font-mono flex items-center gap-1.5">
              <i class="fa-solid fa-location-dot text-blue-500 dark:text-blue-400"></i>
              {{ v.location_name || "—" }}
            </p>
          </div>
        </aside>
      </div>

      <!-- Video -->
      <div v-if="hasVideo" class="bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 rounded-xl overflow-hidden shadow-sm dark:shadow-none">
        <div class="px-4 py-3 border-b border-slate-200 dark:border-slate-800 flex items-center gap-2 text-xs text-slate-600 dark:text-slate-300">
          <i class="fa-solid fa-video text-blue-500 dark:text-blue-400"></i> Rekaman Video
        </div>
        <video :src="v.video_url" controls class="w-full max-h-[420px] bg-black"></video>
      </div>

      <!-- Keterangan tambahan -->
      <div v-if="v.error_message" class="bg-rose-50 dark:bg-rose-950/40 border border-rose-200 dark:border-rose-800/60 rounded-xl p-4 text-xs text-rose-600 dark:text-rose-300 flex items-start gap-3">
        <i class="fa-solid fa-triangle-exclamation mt-0.5"></i>
        <div>
          <p class="font-semibold mb-1">Keterangan Pengiriman</p>
          <p class="font-mono text-rose-500 dark:text-rose-200/90">{{ v.error_message }}</p>
        </div>
      </div>
    </template>
  </div>

  <!-- Preview besar dengan zoom & geser (selalu tampilan sinematik gelap) -->
  <Teleport to="body">
    <div
      v-if="previewSrc"
      class="fixed inset-0 z-50 flex flex-col items-center justify-center p-4 bg-black/90 backdrop-blur-sm"
      @click.self="closePreview"
    >
      <div class="relative w-full max-w-6xl flex flex-col">
        <!-- Toolbar -->
        <div class="flex items-center justify-between mb-3">
          <span class="viewer-bar flex items-center gap-2">
            <span class="w-1.5 h-1.5 rounded-full bg-red-500 animate-pulse"></span>
            Bukti Rekaman · {{ v?.plate || "—" }} · {{ fmtEpoch(v?.capture_time) }}
            <span class="viewer-dim">· {{ Math.round(scale * 100) }}%</span>
          </span>
          <div class="flex items-center gap-1.5">
            <button type="button" @click="zoomBy(1.25)" class="viewbtn" title="Perbesar"><i class="fa-solid fa-plus"></i></button>
            <button type="button" @click="zoomBy(1 / 1.25)" class="viewbtn" title="Perkecil"><i class="fa-solid fa-minus"></i></button>
            <button type="button" @click="resetView" class="viewbtn" title="Tampilkan pantas layar"><i class="fa-solid fa-expand"></i></button>
            <button
              type="button"
              @click="closePreview"
              class="viewbtn viewbtn-close"
              title="Tutup (Esc)"
            >
              <i class="fa-solid fa-xmark"></i>
            </button>
          </div>
        </div>

        <!-- Area gambar: geser saat dizoom -->
        <div
          class="relative rounded-xl overflow-hidden bg-black border border-slate-700 shadow-2xl select-none touch-none"
          style="height: 78vh"
          @wheel.prevent="onWheel"
          @pointerdown="onPointerDown"
          @pointermove="onPointerMove"
          @pointerup="onPointerUp"
          @pointercancel="onPointerUp"
          :class="dragging ? 'cursor-grabbing' : 'cursor-grab'"
        >
          <span class="corner tl"></span><span class="corner tr"></span>
          <span class="corner bl"></span><span class="corner br"></span>
          <img
            :src="previewSrc"
            alt="Preview besar"
            class="absolute top-1/2 left-1/2 block max-w-none"
            :style="{
              transform: `translate(calc(-50% + ${tx}px), calc(-50% + ${ty}px)) scale(${scale})`,
              transition: dragging ? 'none' : 'transform 0.15s ease-out'
            }"
            draggable="false"
          />
        </div>

        <p class="viewer-hint mt-2 text-center font-mono">
          Klik &amp; geser untuk menggeser gambar · scroll untuk zoom · 100% = pas layar
        </p>
      </div>
    </div>
  </Teleport>
</template>

<style scoped>
.corner {
  position: absolute;
  width: 26px;
  height: 26px;
  border-color: rgba(96, 165, 250, 0.9);
  z-index: 10;
  filter: drop-shadow(0 0 4px rgba(96, 165, 250, 0.6));
}
.corner.tl { top: 10px; left: 10px; border-top: 2px solid; border-left: 2px solid; border-radius: 4px 0 0 0; }
.corner.tr { top: 10px; right: 10px; border-top: 2px solid; border-right: 2px solid; border-radius: 0 4px 0 0; }
.corner.bl { bottom: 10px; left: 10px; border-bottom: 2px solid; border-left: 2px solid; border-radius: 0 0 0 4px; }
.corner.br { bottom: 10px; right: 10px; border-bottom: 2px solid; border-right: 2px solid; border-radius: 0 0 4px 0; }

.scan-surface::after {
  content: "";
  position: absolute;
  inset: 0;
  pointer-events: none;
  background: linear-gradient(
    180deg,
    transparent 0%,
    rgba(56, 189, 248, 0.03) 48%,
    rgba(56, 189, 248, 0.12) 50%,
    rgba(56, 189, 248, 0.03) 52%,
    transparent 100%
  );
  background-size: 100% 220%;
  animation: scan 5s linear infinite;
}
@keyframes scan {
  0% { background-position: 0 -120%; }
  100% { background-position: 0 120%; }
}

/* Modal viewer selalu gelap: gunakan class sendiri agar tidak
   terkena override tema terang dari style.css */
.viewer-bar {
  font-size: 12px;
  font-family: ui-monospace, "Courier New", monospace;
  color: #e2e8f0;
  background: rgba(15, 23, 42, 0.7);
  border: 1px solid #334155;
  border-radius: 8px;
  padding: 6px 10px;
}
.viewer-dim {
  color: #64748b;
}
.viewer-hint {
  font-size: 10px;
  color: #94a3b8;
}
.viewbtn {
  width: 32px;
  height: 32px;
  border-radius: 8px;
  background: rgba(30, 41, 59, 0.9);
  border: 1px solid #334155;
  color: #cbd5e1;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: all 0.15s;
  cursor: pointer;
}
.viewbtn:hover {
  background: #334155;
  color: #ffffff;
}
.viewbtn-close:hover {
  background: #dc2626;
  border-color: #ef4444;
  color: #ffffff;
}
</style>