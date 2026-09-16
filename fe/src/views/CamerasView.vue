<script setup lang="ts">
import { ref, onMounted, computed } from "vue";
import { api } from "../api";

const cameras = ref<any[]>([]);
const clients = ref<any[]>([]);
const filterClientId = ref<string | number>("");
const isLoading = ref(false);
const showAddForm = ref(false);
const notice = ref("");
const noticeType = ref<"success" | "error">("success");

const newCam = ref({
  camera_code: "",
  client_id: "",
  device_name: "",
  location_name: "",
  address: "",
});

const editingCamera = ref<any>(null);
const isUpdating = ref(false);

const deletingCamera = ref<any>(null);
const isDeleting = ref(false);

function setNotice(text: string, type: "success" | "error" = "success") {
  notice.value = text;
  noticeType.value = type;
  if (type === "success") {
    setTimeout(() => {
      if (notice.value === text) notice.value = "";
    }, 5000);
  }
}

async function loadClients() {
  try {
    clients.value = await api.clients();
  } catch (e: any) {
    console.error("Gagal memuat klien:", e);
  }
}

async function loadCameras() {
  isLoading.value = true;
  try {
    cameras.value = await api.cameras();
  } catch (e: any) {
    setNotice("Gagal memuat kamera: " + e.message, "error");
  } finally {
    isLoading.value = false;
  }
}

function getClientLabel(clientId: number) {
  const cl = clients.value.find((c) => c.id === clientId);
  return cl ? cl.name : `Client #${clientId}`;
}

const displayedCameras = computed(() => {
  if (!filterClientId.value) return cameras.value;
  return cameras.value.filter((c) => String(c.client_id) === String(filterClientId.value));
});

async function submitAddCam() {
  try {
    await api.createCamera({
      ...newCam.value,
      client_id: Number(newCam.value.client_id),
    });
    setNotice(`Kamera [${newCam.value.camera_code || '-'}] "${newCam.value.device_name}" berhasil ditambahkan`, "success");
    newCam.value = { camera_code: "", client_id: "", device_name: "", location_name: "", address: "" };
    showAddForm.value = false;
    loadCameras();
  } catch (e: any) {
    setNotice("Gagal menambah kamera: " + e.message, "error");
  }
}

function openEdit(c: any) {
  editingCamera.value = { ...c };
}

async function submitUpdateCamera() {
  if (!editingCamera.value) return;
  isUpdating.value = true;
  try {
    await api.updateCamera(editingCamera.value.id, {
      ...editingCamera.value,
      client_id: Number(editingCamera.value.client_id),
    });
    setNotice(`Kamera "${editingCamera.value.device_name}" berhasil diperbarui`, "success");
    editingCamera.value = null;
    loadCameras();
  } catch (e: any) {
    setNotice("Gagal memperbarui kamera: " + e.message, "error");
  } finally {
    isUpdating.value = false;
  }
}

function openDelete(c: any) {
  deletingCamera.value = c;
}

async function confirmDeleteCamera() {
  if (!deletingCamera.value) return;
  isDeleting.value = true;
  try {
    await api.deleteCamera(deletingCamera.value.id);
    setNotice(`Kamera "${deletingCamera.value.device_name}" berhasil di-softdelete`, "success");
    deletingCamera.value = null;
    loadCameras();
  } catch (e: any) {
    setNotice("Gagal menghapus kamera: " + e.message, "error");
  } finally {
    isDeleting.value = false;
  }
}

onMounted(() => {
  loadCameras();
  loadClients();
});
</script>

<template>
  <div class="space-y-6 max-w-7xl mx-auto">
    <!-- Header -->
    <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
      <div>
        <h2 class="text-lg font-bold text-slate-900 dark:text-white tracking-wide">Kamera Pemantau ETLE</h2>
        <p class="text-xs text-slate-500 dark:text-slate-400">Pengelolaan perangkat kamera sensor yang terhubung ke sistem gateway</p>
      </div>

      <div class="flex flex-wrap items-center gap-2.5">
        <!-- Filter Klien -->
        <select
          v-model="filterClientId"
          class="bg-white hover:bg-slate-50 text-slate-700 border border-slate-300 dark:bg-slate-800 dark:hover:bg-slate-700 dark:text-slate-200 dark:border-slate-700 text-xs py-2 px-3 rounded-lg transition cursor-pointer shadow-2xs font-medium focus:outline-none"
        >
          <option value="">Semua Klien (Polres / Polda)</option>
          <option v-for="cl in clients" :key="cl.id" :value="cl.id">
            #{{ cl.id }} - {{ cl.name }}
          </option>
        </select>

        <button
          @click="showAddForm = !showAddForm"
          :class="[
            showAddForm
              ? 'bg-slate-200 hover:bg-slate-300 text-slate-800 dark:bg-slate-800 dark:hover:bg-slate-700 dark:text-slate-200 border border-slate-300 dark:border-slate-700'
              : 'bg-blue-600 hover:bg-blue-500 text-white shadow-sm'
          ]"
          class="text-xs font-semibold py-2 px-3.5 rounded-lg transition flex items-center gap-2 cursor-pointer"
        >
          <i :class="['fa-solid', showAddForm ? 'fa-xmark' : 'fa-plus']"></i>
          <span>{{ showAddForm ? "Tutup Formulir" : "Tambah Kamera" }}</span>
        </button>

        <button
          @click="loadCameras"
          class="bg-white hover:bg-slate-100 text-slate-700 border border-slate-300 dark:bg-slate-800 dark:hover:bg-slate-700 dark:text-slate-300 dark:border-slate-700 text-xs p-2 rounded-lg transition cursor-pointer shadow-2xs flex items-center justify-center"
          title="Segarkan Data"
        >
          <i class="fa-solid fa-rotate"></i>
        </button>
      </div>
    </div>

    <!-- Alert -->
    <div
      v-if="notice"
      :class="[
        noticeType === 'success' ? 'bg-emerald-950/60 border-emerald-800 text-emerald-300' :
        'bg-rose-950/60 border-rose-800 text-rose-300'
      ]"
      class="p-3 border rounded-lg text-xs flex items-center justify-between"
    >
      <div class="flex items-center gap-2">
        <i :class="['fa-solid', noticeType === 'success' ? 'fa-circle-check text-emerald-400' : 'fa-circle-exclamation text-rose-400']"></i>
        <span>{{ notice }}</span>
      </div>
      <button @click="notice = ''" class="text-slate-400 hover:text-white cursor-pointer">
        <i class="fa-solid fa-xmark"></i>
      </button>
    </div>

    <!-- Add Form -->
    <div v-if="showAddForm" class="bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 p-5 rounded-xl space-y-4 shadow-xs">
      <div class="flex items-center justify-between pb-2 border-b border-slate-200 dark:border-slate-800 text-xs font-semibold text-slate-900 dark:text-white">
        <div class="flex items-center gap-2">
          <i class="fa-solid fa-video text-blue-500"></i>
          <span>Registrasi Perangkat Kamera Baru</span>
        </div>
        <span class="text-[11px] text-slate-500 dark:text-slate-400 font-normal">
          Kode unik digunakan untuk pencocokan otomatis nama file ANPR/XML (misal: UNIX1_...)
        </span>
      </div>

      <form @submit.prevent="submitAddCam" class="grid grid-cols-1 md:grid-cols-3 gap-4">
        <div>
          <label class="block text-xs font-medium text-slate-700 dark:text-slate-300 mb-1">
            Kode Unik (Unix Code) *
            <span class="text-[10px] text-blue-500 font-normal ml-1">(Prefix Nama File)</span>
          </label>
          <input
            v-model="newCam.camera_code"
            placeholder="cth: UNIX1 (sesuai awalan file XML)"
            required
            class="w-full bg-white dark:bg-slate-950 border border-slate-300 dark:border-slate-800 text-slate-900 dark:text-slate-100 text-xs px-3 py-2 rounded-lg font-mono focus:outline-none focus:border-blue-500"
          />
        </div>

        <div>
          <label class="block text-xs font-medium text-slate-700 dark:text-slate-300 mb-1">Nama Perangkat (Device Name) *</label>
          <input
            v-model="newCam.device_name"
            placeholder="cth: CAM-BANGIL-01"
            required
            class="w-full bg-white dark:bg-slate-950 border border-slate-300 dark:border-slate-800 text-slate-900 dark:text-slate-100 text-xs px-3 py-2 rounded-lg focus:outline-none focus:border-blue-500"
          />
        </div>

        <div>
          <label class="block text-xs font-medium text-slate-700 dark:text-slate-300 mb-1">
            Klien (Polres / Polda) *
          </label>
          <select
            v-model="newCam.client_id"
            required
            class="w-full bg-white dark:bg-slate-950 border border-slate-300 dark:border-slate-800 text-slate-900 dark:text-slate-100 text-xs px-3 py-2 rounded-lg focus:outline-none focus:border-blue-500 cursor-pointer"
          >
            <option value="" disabled>-- Pilih Klien / Satlantas --</option>
            <option v-for="cl in clients" :key="cl.id" :value="cl.id">
              #{{ cl.id }} - {{ cl.name }} ({{ cl.client_id }})
            </option>
          </select>
        </div>

        <div>
          <label class="block text-xs font-medium text-slate-700 dark:text-slate-300 mb-1">Nama Lokasi Penempatan</label>
          <input
            v-model="newCam.location_name"
            placeholder="cth: Pasuruan / Simpang Bangil"
            class="w-full bg-white dark:bg-slate-950 border border-slate-300 dark:border-slate-800 text-slate-900 dark:text-slate-100 text-xs px-3 py-2 rounded-lg focus:outline-none focus:border-blue-500"
          />
        </div>

        <div class="md:col-span-2">
          <label class="block text-xs font-medium text-slate-700 dark:text-slate-300 mb-1">Alamat Lengkap</label>
          <input
            v-model="newCam.address"
            placeholder="cth: Jalan A.Yani Bangil Kab.Pasuruan"
            class="w-full bg-white dark:bg-slate-950 border border-slate-300 dark:border-slate-800 text-slate-900 dark:text-slate-100 text-xs px-3 py-2 rounded-lg focus:outline-none focus:border-blue-500"
          />
        </div>

        <div class="md:col-span-3 flex justify-end gap-2.5 pt-2">
          <button
            type="button"
            @click="showAddForm = false"
            class="bg-slate-100 hover:bg-slate-200 text-slate-700 border border-slate-300 dark:bg-slate-800 dark:hover:bg-slate-700 dark:text-slate-300 dark:border-slate-700 text-xs py-2 px-4 rounded-lg transition cursor-pointer font-medium"
          >
            Batal
          </button>
          <button
            type="submit"
            class="bg-blue-600 hover:bg-blue-500 text-white text-xs font-semibold py-2 px-5 rounded-lg transition flex items-center gap-2 cursor-pointer shadow-sm"
          >
            <i class="fa-solid fa-floppy-disk"></i>
            <span>Simpan Kamera</span>
          </button>
        </div>
      </form>
    </div>

    <!-- Table Card -->
    <div class="bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 rounded-xl overflow-hidden shadow-xs">
      <div class="overflow-x-auto">
        <table class="w-full text-left text-xs">
          <thead class="bg-slate-50 dark:bg-slate-950/70 text-slate-500 dark:text-slate-400 border-b border-slate-200 dark:border-slate-800 font-medium">
            <tr>
              <th class="py-3 px-4">ID</th>
              <th class="py-3 px-4">Kode Unik (UNIX)</th>
              <th class="py-3 px-4">Klien (Polres / Polda)</th>
              <th class="py-3 px-4">Nama Perangkat</th>
              <th class="py-3 px-4">Lokasi Penempatan</th>
              <th class="py-3 px-4">Alamat Lengkap</th>
              <th class="py-3 px-4">Status</th>
              <th class="py-3 px-4 text-right">Aksi</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-slate-100 dark:divide-slate-800/60">
            <tr v-if="displayedCameras.length === 0 && !isLoading">
              <td colspan="8" class="py-10 text-center text-slate-500 italic">
                {{ filterClientId ? "Tidak ada kamera untuk klien yang dipilih" : "Belum ada perangkat kamera terdaftar" }}
              </td>
            </tr>
            <tr v-for="c in displayedCameras" :key="c.id" class="hover:bg-slate-50/70 dark:hover:bg-slate-800/40 transition">
              <td class="py-3 px-4 font-mono text-slate-400">#{{ c.id }}</td>
              <td class="py-3 px-4 whitespace-nowrap">
                <span
                  v-if="c.camera_code"
                  class="inline-flex items-center gap-1 font-mono font-bold text-xs bg-blue-500/10 text-blue-600 dark:text-blue-400 px-2 py-0.5 rounded border border-blue-500/20"
                >
                  <i class="fa-solid fa-barcode text-[10px]"></i>
                  <span>{{ c.camera_code }}</span>
                </span>
                <span v-else class="text-slate-400 font-mono italic text-[11px]">—</span>
              </td>
              <td class="py-3 px-4">
                <div class="font-medium text-slate-900 dark:text-slate-200">
                  {{ c.client_name || getClientLabel(c.client_id) }}
                </div>
                <div class="text-[10px] text-slate-500 dark:text-slate-400 font-mono">
                  ID: #{{ c.client_id }}
                </div>
              </td>
              <td class="py-3 px-4 font-semibold text-slate-900 dark:text-white font-mono">{{ c.device_name }}</td>
              <td class="py-3 px-4 text-slate-700 dark:text-slate-300">{{ c.location_name || "—" }}</td>
              <td class="py-3 px-4 text-slate-500 dark:text-slate-400 max-w-xs truncate">{{ c.address || "—" }}</td>
              <td class="py-3 px-4">
                <span
                  v-if="c.status === 'active'"
                  class="inline-flex items-center gap-1 text-[11px] font-medium text-emerald-700 bg-emerald-50 dark:text-emerald-400 dark:bg-emerald-950/40 px-2 py-0.5 rounded border border-emerald-200 dark:border-emerald-800/40"
                >
                  <span class="w-1.5 h-1.5 rounded-full bg-emerald-500"></span>
                  <span>Aktif</span>
                </span>
                <span
                  v-else
                  class="inline-flex items-center gap-1 text-[11px] font-medium text-slate-500 bg-slate-100 dark:text-slate-400 dark:bg-slate-800 px-2 py-0.5 rounded border border-slate-200 dark:border-slate-700"
                >
                  <span class="w-1.5 h-1.5 rounded-full bg-slate-400"></span>
                  <span>{{ c.status || 'Nonaktif' }}</span>
                </span>
              </td>
              <td class="py-3 px-4 text-right">
                <div class="flex items-center justify-end gap-1.5">
                  <!-- Edit Button -->
                  <button
                    @click="openEdit(c)"
                    class="bg-white hover:bg-blue-50 text-blue-600 border border-slate-300 hover:border-blue-400 dark:bg-slate-800 dark:hover:bg-slate-700 dark:text-blue-400 dark:border-slate-700 text-xs px-2.5 py-1.5 rounded-lg transition flex items-center gap-1 cursor-pointer"
                    title="Edit Data Kamera"
                  >
                    <i class="fa-solid fa-pen-to-square text-[11px]"></i>
                    <span>Edit</span>
                  </button>

                  <!-- Delete Button (Softdelete) -->
                  <button
                    @click="openDelete(c)"
                    class="bg-white hover:bg-rose-50 text-rose-600 border border-slate-300 hover:border-rose-400 dark:bg-slate-800 dark:hover:bg-rose-950/60 dark:text-rose-400 dark:border-slate-700 text-xs px-2 py-1.5 rounded-lg transition flex items-center gap-1 cursor-pointer"
                    title="Hapus Kamera (Soft Delete)"
                  >
                    <i class="fa-solid fa-trash-can text-[11px]"></i>
                  </button>
                </div>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>

    <!-- Modal Edit Kamera -->
    <Teleport to="body">
      <div
        v-if="editingCamera"
        class="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/60 backdrop-blur-xs transition-opacity"
      >
        <div class="bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 rounded-2xl shadow-2xl max-w-xl w-full p-6 text-slate-800 dark:text-slate-100">
          <div class="flex items-center justify-between pb-3 border-b border-slate-200 dark:border-slate-800 mb-4">
            <div class="flex items-center gap-2.5">
              <div class="w-8 h-8 rounded-lg bg-blue-600/10 border border-blue-500/30 flex items-center justify-center text-blue-500">
                <i class="fa-solid fa-pen-to-square text-sm"></i>
              </div>
              <div>
                <h3 class="text-sm font-bold text-slate-900 dark:text-white tracking-wide">Edit Perangkat Kamera</h3>
                <p class="text-xs text-slate-500 dark:text-slate-400">ID Kamera: #{{ editingCamera.id }}</p>
              </div>
            </div>
            <button @click="editingCamera = null" class="text-slate-400 hover:text-slate-600 dark:hover:text-white cursor-pointer">
              <i class="fa-solid fa-xmark text-base"></i>
            </button>
          </div>

          <form @submit.prevent="submitUpdateCamera" class="space-y-3.5">
            <div class="grid grid-cols-1 sm:grid-cols-2 gap-3">
              <div>
                <label class="block text-xs font-medium text-slate-700 dark:text-slate-300 mb-1">
                  Kode Unik (Unix Code) *
                  <span class="text-[10px] text-blue-500 font-normal ml-1">(Prefix Nama File)</span>
                </label>
                <input
                  v-model="editingCamera.camera_code"
                  required
                  placeholder="cth: UNIX1"
                  class="w-full bg-white dark:bg-slate-950 border border-slate-300 dark:border-slate-800 text-slate-900 dark:text-slate-100 text-xs px-3 py-2 rounded-lg font-mono focus:outline-none focus:border-blue-500"
                />
              </div>

              <div>
                <label class="block text-xs font-medium text-slate-700 dark:text-slate-300 mb-1">Nama Perangkat *</label>
                <input
                  v-model="editingCamera.device_name"
                  required
                  placeholder="cth: CAM-BANGIL-01"
                  class="w-full bg-white dark:bg-slate-950 border border-slate-300 dark:border-slate-800 text-slate-900 dark:text-slate-100 text-xs px-3 py-2 rounded-lg focus:outline-none focus:border-blue-500"
                />
              </div>
            </div>

            <div class="grid grid-cols-1 sm:grid-cols-2 gap-3">
              <div>
                <label class="block text-xs font-medium text-slate-700 dark:text-slate-300 mb-1">
                  Klien (Polres / Polda) *
                </label>
                <select
                  v-model="editingCamera.client_id"
                  required
                  class="w-full bg-white dark:bg-slate-950 border border-slate-300 dark:border-slate-800 text-slate-900 dark:text-slate-100 text-xs px-3 py-2 rounded-lg focus:outline-none focus:border-blue-500 cursor-pointer"
                >
                  <option value="" disabled>-- Pilih Klien / Satlantas --</option>
                  <option v-for="cl in clients" :key="cl.id" :value="cl.id">
                    #{{ cl.id }} - {{ cl.name }} ({{ cl.client_id }})
                  </option>
                </select>
              </div>

              <div>
                <label class="block text-xs font-medium text-slate-700 dark:text-slate-300 mb-1">Status Kamera</label>
                <select
                  v-model="editingCamera.status"
                  class="w-full bg-white dark:bg-slate-950 border border-slate-300 dark:border-slate-800 text-slate-900 dark:text-slate-100 text-xs px-3 py-2 rounded-lg focus:outline-none focus:border-blue-500 cursor-pointer"
                >
                  <option value="active">Aktif</option>
                  <option value="inactive">Nonaktif (Maintenance)</option>
                </select>
              </div>
            </div>

            <div>
              <label class="block text-xs font-medium text-slate-700 dark:text-slate-300 mb-1">Nama Lokasi Penempatan</label>
              <input
                v-model="editingCamera.location_name"
                placeholder="cth: Pasuruan / Simpang Bangil"
                class="w-full bg-white dark:bg-slate-950 border border-slate-300 dark:border-slate-800 text-slate-900 dark:text-slate-100 text-xs px-3 py-2 rounded-lg focus:outline-none focus:border-blue-500"
              />
            </div>

            <div>
              <label class="block text-xs font-medium text-slate-700 dark:text-slate-300 mb-1">Alamat Lengkap</label>
              <input
                v-model="editingCamera.address"
                placeholder="cth: Jalan A.Yani Bangil Kab.Pasuruan"
                class="w-full bg-white dark:bg-slate-950 border border-slate-300 dark:border-slate-800 text-slate-900 dark:text-slate-100 text-xs px-3 py-2 rounded-lg focus:outline-none focus:border-blue-500"
              />
            </div>

            <div class="flex items-center justify-end gap-2.5 pt-4 border-t border-slate-200 dark:border-slate-800">
              <button
                type="button"
                @click="editingCamera = null"
                class="px-4 py-2 text-xs font-medium text-slate-700 dark:text-slate-300 bg-slate-100 dark:bg-slate-800 hover:bg-slate-200 dark:hover:bg-slate-700 rounded-lg transition cursor-pointer"
              >
                Batal
              </button>
              <button
                type="submit"
                :disabled="isUpdating"
                class="px-5 py-2 text-xs font-semibold text-white bg-blue-600 hover:bg-blue-500 rounded-lg transition flex items-center gap-2 cursor-pointer shadow-sm disabled:opacity-50"
              >
                <i v-if="isUpdating" class="fa-solid fa-circle-notch fa-spin text-xs"></i>
                <i v-else class="fa-solid fa-floppy-disk text-xs"></i>
                <span>{{ isUpdating ? "Menyimpan..." : "Simpan Perubahan" }}</span>
              </button>
            </div>
          </form>
        </div>
      </div>
    </Teleport>

    <!-- Modal Konfirmasi Soft Delete Kamera -->
    <Teleport to="body">
      <div
        v-if="deletingCamera"
        class="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/60 backdrop-blur-xs transition-opacity"
      >
        <div class="bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 rounded-2xl shadow-2xl max-w-sm w-full p-6 text-slate-800 dark:text-slate-100">
          <div class="flex items-center gap-3.5 mb-4">
            <div class="w-11 h-11 rounded-xl bg-rose-500/10 border border-rose-500/30 flex items-center justify-center text-rose-500 shrink-0">
              <i class="fa-solid fa-trash-can text-lg"></i>
            </div>
            <div>
              <h3 class="text-sm font-bold text-slate-900 dark:text-white tracking-wide">Hapus Kamera (Soft Delete)</h3>
              <p class="text-xs text-slate-500 dark:text-slate-400">Konfirmasi Penghapusan</p>
            </div>
          </div>

          <p class="text-xs text-slate-600 dark:text-slate-300 mb-2 leading-relaxed">
            Apakah Anda yakin ingin menghapus kamera <strong class="text-slate-900 dark:text-white">"{{ deletingCamera.device_name }}"</strong> (Kode: <code class="text-blue-500 font-mono">{{ deletingCamera.camera_code || '-' }}</code>)?
          </p>
          <div class="p-2.5 rounded-lg bg-amber-500/10 border border-amber-500/20 text-amber-600 dark:text-amber-500 text-[11px] mb-5">
            <i class="fa-solid fa-circle-info mr-1"></i>
            <span>Kamera akan dinonaktifkan (soft delete). File gambar & riwayat pelanggaran masa lalu tidak terhapus.</span>
          </div>

          <div class="flex items-center justify-end gap-2.5">
            <button
              @click="deletingCamera = null"
              :disabled="isDeleting"
              class="px-4 py-2 text-xs font-medium text-slate-700 dark:text-slate-300 bg-slate-100 dark:bg-slate-800 hover:bg-slate-200 dark:hover:bg-slate-700 rounded-lg transition cursor-pointer"
            >
              Batal
            </button>
            <button
              @click="confirmDeleteCamera"
              :disabled="isDeleting"
              class="px-4 py-2 text-xs font-semibold text-white bg-rose-600 hover:bg-rose-500 rounded-lg transition flex items-center gap-2 cursor-pointer shadow-sm disabled:opacity-50"
            >
              <i v-if="isDeleting" class="fa-solid fa-circle-notch fa-spin text-xs"></i>
              <i v-else class="fa-solid fa-trash text-xs"></i>
              <span>{{ isDeleting ? "Menghapus..." : "Ya, Hapus Kamera" }}</span>
            </button>
          </div>
        </div>
      </div>
    </Teleport>
  </div>
</template>
