<script setup lang="ts">
import { ref, onMounted } from "vue";
import { api } from "../api";

const clients = ref<any[]>([]);
const isLoading = ref(false);
const syncLoadingId = ref<number | null>(null);
const notice = ref("");
const noticeType = ref<"success" | "error">("success");

const showAddForm = ref(false);
const newClient = ref({
  name: "",
  client_id: "",
  client_secret: "",
  usertoken: "",
  passtoken: "",
  base_url: "",
});

const editingClient = ref<any>(null);
const isUpdating = ref(false);

const togglingClient = ref<any>(null);
const targetStatus = ref(false);
const isTogglingStatus = ref(false);

const deletingClient = ref<any>(null);
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
  isLoading.value = true;
  try {
    clients.value = await api.clients();
  } catch (e: any) {
    setNotice("Gagal memuat daftar klien: " + e.message, "error");
  } finally {
    isLoading.value = false;
  }
}

async function submitAddClient() {
  try {
    await api.createClient(newClient.value);
    setNotice(`Klien "${newClient.value.name}" berhasil ditambahkan`, "success");
    newClient.value = { name: "", client_id: "", client_secret: "", usertoken: "", passtoken: "", base_url: "" };
    showAddForm.value = false;
    loadClients();
  } catch (e: any) {
    setNotice("Gagal menambah klien: " + e.message, "error");
  }
}

function openEdit(c: any) {
  editingClient.value = { ...c };
}

async function submitUpdateClient() {
  if (!editingClient.value) return;
  isUpdating.value = true;
  try {
    await api.updateClient(editingClient.value.id, editingClient.value);
    setNotice(`Klien "${editingClient.value.name}" berhasil diperbarui`, "success");
    editingClient.value = null;
    loadClients();
  } catch (e: any) {
    setNotice("Gagal memperbarui klien: " + e.message, "error");
  } finally {
    isUpdating.value = false;
  }
}

function openDelete(c: any) {
  deletingClient.value = c;
}

function openToggle(c: any) {
  togglingClient.value = c;
  targetStatus.value = !c.is_active;
}

async function confirmToggleClient() {
  if (!togglingClient.value) return;
  isTogglingStatus.value = true;
  const cli = togglingClient.value;
  try {
    await api.updateClient(cli.id, { ...cli, is_active: targetStatus.value });
    setNotice(
      `Klien "${cli.name}" berhasil ${targetStatus.value ? "diaktifkan" : "dinonaktifkan"}`,
      "success"
    );
    togglingClient.value = null;
    loadClients();
  } catch (e: any) {
    setNotice("Gagal mengubah status klien: " + e.message, "error");
  } finally {
    isTogglingStatus.value = false;
  }
}

async function confirmDeleteClient() {
  if (!deletingClient.value) return;
  isDeleting.value = true;
  try {
    await api.deleteClient(deletingClient.value.id);
    setNotice(`Klien "${deletingClient.value.name}" berhasil di-softdelete`, "success");
    deletingClient.value = null;
    loadClients();
  } catch (e: any) {
    setNotice("Gagal menghapus klien: " + e.message, "error");
  } finally {
    isDeleting.value = false;
  }
}

async function handleSyncClient(id: number, name: string) {
  syncLoadingId.value = id;
  try {
    const res = await api.syncMaster(id);
    setNotice(`Sinkronisasi Polantas "${name}" sukses: ${res.synced} data diperbarui`, "success");
    loadClients();
  } catch (e: any) {
    setNotice(`Sinkronisasi "${name}" gagal: ` + e.message, "error");
  } finally {
    syncLoadingId.value = null;
  }
}

onMounted(loadClients);
</script>

<template>
  <div class="space-y-6 max-w-7xl mx-auto">
    <!-- Header with Action -->
    <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
      <div>
        <h2 class="text-lg font-bold text-white tracking-wide">Klien ETLE (Polres / Polda)</h2>
        <p class="text-xs text-slate-400">Manajemen kredensial instansi kepolisian daerah dan endpoint gateway</p>
      </div>

      <div class="flex items-center gap-2.5">
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
          <span>{{ showAddForm ? "Tutup Formulir" : "Tambah Klien" }}</span>
        </button>

        <button
          @click="loadClients"
          class="bg-white hover:bg-slate-100 text-slate-700 border border-slate-300 dark:bg-slate-800 dark:hover:bg-slate-700 dark:text-slate-300 dark:border-slate-700 text-xs p-2 rounded-lg transition cursor-pointer shadow-2xs flex items-center justify-center"
          title="Segarkan Data"
        >
          <i class="fa-solid fa-rotate"></i>
        </button>
      </div>
    </div>

    <!-- Feedback Notice -->
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

    <!-- Collapsible Add Form -->
    <div v-if="showAddForm" class="bg-slate-900 border border-slate-800 p-5 rounded-xl space-y-4">
      <div class="flex items-center gap-2 pb-2 border-b border-slate-800 text-xs font-semibold text-white">
        <i class="fa-solid fa-building-shield text-blue-400"></i>
        <span>Pendaftaran Klien Satuan Wilayah Baru</span>
      </div>

      <form @submit.prevent="submitAddClient" class="grid grid-cols-1 md:grid-cols-3 gap-4">
        <div>
          <label class="block text-xs font-medium text-slate-300 mb-1">Nama Satuan / Instansi *</label>
          <input
            v-model="newClient.name"
            placeholder="cth: Ditlantas Polda Jatim / Polres Pasuruan"
            required
            class="w-full bg-slate-950 border border-slate-800 text-slate-100 text-xs px-3 py-2 rounded-lg focus:outline-none focus:border-blue-500"
          />
        </div>

        <div>
          <label class="block text-xs font-medium text-slate-300 mb-1">Client ID</label>
          <input
            v-model="newClient.client_id"
            placeholder="Kredensial dari Korlantas"
            class="w-full bg-slate-950 border border-slate-800 text-slate-100 text-xs px-3 py-2 rounded-lg focus:outline-none focus:border-blue-500"
          />
        </div>

        <div>
          <label class="block text-xs font-medium text-slate-300 mb-1">Client Secret</label>
          <input
            v-model="newClient.client_secret"
            placeholder="Kredensial dari Korlantas"
            type="password"
            class="w-full bg-slate-950 border border-slate-800 text-slate-100 text-xs px-3 py-2 rounded-lg focus:outline-none focus:border-blue-500"
          />
        </div>

        <div>
          <label class="block text-xs font-medium text-slate-300 mb-1">Usertoken</label>
          <input
            v-model="newClient.usertoken"
            placeholder="Usertoken API"
            class="w-full bg-slate-950 border border-slate-800 text-slate-100 text-xs px-3 py-2 rounded-lg focus:outline-none focus:border-blue-500"
          />
        </div>

        <div>
          <label class="block text-xs font-medium text-slate-300 mb-1">Passtoken</label>
          <input
            v-model="newClient.passtoken"
            placeholder="Passtoken API"
            type="password"
            class="w-full bg-slate-950 border border-slate-800 text-slate-100 text-xs px-3 py-2 rounded-lg focus:outline-none focus:border-blue-500"
          />
        </div>

        <div>
          <label class="block text-xs font-medium text-slate-300 mb-1">Base URL Endpoint</label>
          <input
            v-model="newClient.base_url"
            placeholder="https://api-etle.polri.go.id (default)"
            class="w-full bg-slate-950 border border-slate-800 text-slate-100 text-xs px-3 py-2 rounded-lg focus:outline-none focus:border-blue-500"
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
            <span>Simpan Klien</span>
          </button>
        </div>
      </form>
    </div>

    <!-- Table Card -->
    <div class="bg-slate-900 border border-slate-800 rounded-xl overflow-hidden shadow-xs">
      <div class="overflow-x-auto">
        <table class="w-full text-left text-xs">
          <thead class="bg-slate-950/70 text-slate-400 border-b border-slate-800 font-medium">
            <tr>
              <th class="py-3 px-4">ID</th>
              <th class="py-3 px-4">Nama Instansi</th>
              <th class="py-3 px-4">Client ID</th>
              <th class="py-3 px-4">Base URL</th>
              <th class="py-3 px-4">Status</th>
              <th class="py-3 px-4">Status Token</th>
              <th class="py-3 px-4 text-right">Aksi</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-slate-800/60">
            <tr v-if="clients.length === 0 && !isLoading">
              <td colspan="7" class="py-10 text-center text-slate-500 italic">
                Belum ada data klien terdaftar
              </td>
            </tr>
            <tr v-for="c in clients" :key="c.id" class="hover:bg-slate-800/40 transition">
              <td class="py-3 px-4 font-mono text-slate-400">#{{ c.id }}</td>
              <td class="py-3 px-4 font-semibold text-white">{{ c.name }}</td>
              <td class="py-3 px-4 font-mono text-slate-300">{{ c.client_id || "—" }}</td>
              <td class="py-3 px-4 text-slate-400 font-mono text-[11px] truncate max-w-[200px]">{{ c.base_url || "default" }}</td>
              <td class="py-3 px-4">
                <button
                  @click="openToggle(c)"
                  :disabled="isTogglingStatus"
                  role="switch"
                  :aria-checked="c.is_active"
                  :title="c.is_active ? 'Klik untuk menonaktifkan' : 'Klik untuk mengaktifkan'"
                  class="group inline-flex items-center gap-2 cursor-pointer disabled:opacity-50"
                >
                  <span
                    class="relative inline-flex items-center h-5 w-9 rounded-full transition-colors"
                    :class="c.is_active ? 'bg-emerald-600' : 'bg-slate-700'"
                  >
                    <span
                      class="inline-block h-3.5 w-3.5 transform rounded-full bg-white shadow-sm transition-transform"
                      :class="c.is_active ? 'translate-x-[19px]' : 'translate-x-[3px]'"
                    ></span>
                  </span>
                  <span
                    class="text-[11px] font-semibold"
                    :class="c.is_active ? 'text-emerald-400' : 'text-slate-500'"
                  >
                    {{ c.is_active ? "Aktif" : "Tidak Aktif" }}
                  </span>
                </button>
              </td>
              <td class="py-3 px-4">
                <span
                  v-if="c.auth_token"
                  class="inline-flex items-center gap-1.5 px-2 py-0.5 rounded bg-emerald-950/40 text-emerald-400 border border-emerald-800/40 text-[10px] font-semibold"
                >
                  <span class="w-1.5 h-1.5 rounded-full bg-emerald-400"></span>
                  Terhubung
                </span>
                <span
                  v-else
                  class="inline-flex items-center gap-1.5 px-2 py-0.5 rounded bg-slate-800 text-slate-400 text-[10px]"
                >
                  <span class="w-1.5 h-1.5 rounded-full bg-slate-500"></span>
                  Belum Login
                </span>
              </td>
              <td class="py-3 px-4 text-right">
                <div class="flex items-center justify-end gap-1.5">
                  <!-- Sync Polantas -->
                  <button
                    @click="handleSyncClient(c.id, c.name)"
                    :disabled="syncLoadingId === c.id"
                    class="bg-white hover:bg-slate-100 text-slate-700 border border-slate-300 dark:bg-slate-800 dark:hover:bg-blue-600 dark:text-slate-300 dark:hover:text-white dark:border-slate-700 text-xs px-2.5 py-1.5 rounded-lg transition flex items-center gap-1 cursor-pointer"
                    title="Sinkronisasi Data Polantas Klien"
                  >
                    <i :class="['fa-solid', syncLoadingId === c.id ? 'fa-circle-notch fa-spin' : 'fa-arrows-rotate text-[10px]']"></i>
                    <span>Sync</span>
                  </button>

                  <!-- Edit Button -->
                  <button
                    @click="openEdit(c)"
                    class="bg-white hover:bg-blue-50 text-blue-600 border border-slate-300 hover:border-blue-400 dark:bg-slate-800 dark:hover:bg-slate-700 dark:text-blue-400 dark:border-slate-700 text-xs px-2.5 py-1.5 rounded-lg transition flex items-center gap-1 cursor-pointer"
                    title="Edit Data Klien"
                  >
                    <i class="fa-solid fa-pen-to-square text-[11px]"></i>
                    <span>Edit</span>
                  </button>

                  <!-- Delete Button (Softdelete) -->
                  <button
                    @click="openDelete(c)"
                    class="bg-white hover:bg-rose-50 text-rose-600 border border-slate-300 hover:border-rose-400 dark:bg-slate-800 dark:hover:bg-rose-950/60 dark:text-rose-400 dark:border-slate-700 text-xs px-2 py-1.5 rounded-lg transition flex items-center gap-1 cursor-pointer"
                    title="Hapus Klien (Soft Delete)"
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

    <!-- Modal Edit Klien -->
    <Teleport to="body">
      <div
        v-if="editingClient"
        class="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/60 backdrop-blur-xs transition-opacity"
      >
        <div class="bg-slate-900 border border-slate-800 rounded-2xl shadow-2xl max-w-xl w-full p-6 text-slate-100">
          <div class="flex items-center justify-between pb-3 border-b border-slate-800 mb-4">
            <div class="flex items-center gap-2.5">
              <div class="w-8 h-8 rounded-lg bg-blue-600/10 border border-blue-500/30 flex items-center justify-center text-blue-500">
                <i class="fa-solid fa-pen-to-square text-sm"></i>
              </div>
              <div>
                <h3 class="text-sm font-bold text-white tracking-wide">Edit Data Klien (Polres / Polda)</h3>
                <p class="text-xs text-slate-400">ID Klien: #{{ editingClient.id }}</p>
              </div>
            </div>
            <button @click="editingClient = null" class="text-slate-400 hover:text-white cursor-pointer">
              <i class="fa-solid fa-xmark text-base"></i>
            </button>
          </div>

          <form @submit.prevent="submitUpdateClient" class="space-y-3.5">
            <div>
              <label class="block text-xs font-medium text-slate-300 mb-1">Nama Satuan / Instansi *</label>
              <input
                v-model="editingClient.name"
                required
                class="w-full bg-slate-950 border border-slate-800 text-slate-100 text-xs px-3 py-2 rounded-lg focus:outline-none focus:border-blue-500"
              />
            </div>

            <div class="grid grid-cols-1 sm:grid-cols-2 gap-3">
              <div>
                <label class="block text-xs font-medium text-slate-300 mb-1">Client ID</label>
                <input
                  v-model="editingClient.client_id"
                  class="w-full bg-slate-950 border border-slate-800 text-slate-100 text-xs px-3 py-2 rounded-lg focus:outline-none focus:border-blue-500"
                />
              </div>

              <div>
                <label class="block text-xs font-medium text-slate-300 mb-1">Client Secret</label>
                <input
                  v-model="editingClient.client_secret"
                  type="password"
                  placeholder="Isi jika ingin memperbarui"
                  class="w-full bg-slate-950 border border-slate-800 text-slate-100 text-xs px-3 py-2 rounded-lg focus:outline-none focus:border-blue-500"
                />
              </div>
            </div>

            <div class="grid grid-cols-1 sm:grid-cols-2 gap-3">
              <div>
                <label class="block text-xs font-medium text-slate-300 mb-1">Usertoken</label>
                <input
                  v-model="editingClient.usertoken"
                  class="w-full bg-slate-950 border border-slate-800 text-slate-100 text-xs px-3 py-2 rounded-lg focus:outline-none focus:border-blue-500"
                />
              </div>

              <div>
                <label class="block text-xs font-medium text-slate-300 mb-1">Passtoken</label>
                <input
                  v-model="editingClient.passtoken"
                  type="password"
                  placeholder="Isi jika ingin memperbarui"
                  class="w-full bg-slate-950 border border-slate-800 text-slate-100 text-xs px-3 py-2 rounded-lg focus:outline-none focus:border-blue-500"
                />
              </div>
            </div>

            <div>
              <label class="block text-xs font-medium text-slate-300 mb-1">Base URL Endpoint</label>
              <input
                v-model="editingClient.base_url"
                placeholder="https://api-etle.polri.go.id"
                class="w-full bg-slate-950 border border-slate-800 text-slate-100 text-xs px-3 py-2 rounded-lg focus:outline-none focus:border-blue-500"
              />
            </div>

            <div class="flex items-center justify-end gap-2.5 pt-4 border-t border-slate-800">
              <button
                type="button"
                @click="editingClient = null"
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

    <!-- Modal Konfirmasi Ubah Status Klien -->
    <Teleport to="body">
      <div
        v-if="togglingClient"
        class="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/60 backdrop-blur-xs transition-opacity"
      >
        <div class="bg-slate-900 border border-slate-800 rounded-2xl shadow-2xl max-w-sm w-full p-6 text-slate-100">
          <div class="flex items-center gap-3.5 mb-4">
            <div class="w-11 h-11 rounded-xl bg-blue-500/10 border border-blue-500/30 flex items-center justify-center text-blue-500 shrink-0">
              <i class="fa-solid fa-circle-question text-lg"></i>
            </div>
            <div>
              <h3 class="text-sm font-bold text-white tracking-wide">Ubah Status Klien</h3>
              <p class="text-xs text-slate-400">Konfirmasi Pengaktifan / Penonaktifan</p>
            </div>
          </div>

          <p class="text-xs text-slate-300 mb-3 leading-relaxed">
            Apakah Anda yakin ingin
            <strong class="text-white">{{ targetStatus ? "mengaktifkan" : "menonaktifkan" }}</strong>
            klien <strong class="text-white">"{{ togglingClient.name }}"</strong>?
          </p>
          <div
            class="p-2.5 rounded-lg bg-amber-500/10 border border-amber-500/20 text-amber-500 text-[11px] mb-5"
          >
            <i class="fa-solid fa-circle-info mr-1"></i>
            <span v-if="!targetStatus">
              Klien yang tidak aktif tidak akan diproses pengiriman pelanggaran & sinkronisasi.
            </span>
            <span v-else>
              Klien aktif akan kembali diproses pengiriman pelanggaran & sinkronisasi.
            </span>
          </div>

          <div class="flex items-center justify-end gap-2.5">
            <button
              @click="togglingClient = null"
              :disabled="isTogglingStatus"
              class="px-4 py-2 text-xs font-medium text-slate-700 dark:text-slate-300 bg-slate-100 dark:bg-slate-800 hover:bg-slate-200 dark:hover:bg-slate-700 rounded-lg transition cursor-pointer"
            >
              Batal
            </button>
            <button
              @click="confirmToggleClient"
              :disabled="isTogglingStatus"
              :class="targetStatus ? 'bg-emerald-600 hover:bg-emerald-500' : 'bg-rose-600 hover:bg-rose-500'"
              class="px-4 py-2 text-xs font-semibold text-white rounded-lg transition flex items-center gap-2 cursor-pointer shadow-sm disabled:opacity-50"
            >
              <i v-if="isTogglingStatus" class="fa-solid fa-circle-notch fa-spin text-xs"></i>
              <i v-else :class="['fa-solid', targetStatus ? 'fa-check' : 'fa-ban']"></i>
              <span>
                {{ isTogglingStatus ? "Menyimpan..." : targetStatus ? "Ya, Aktifkan" : "Ya, Nonaktifkan" }}
              </span>
            </button>
          </div>
        </div>
      </div>
    </Teleport>

    <!-- Modal Konfirmasi Soft Delete Klien -->
    <Teleport to="body">
      <div
        v-if="deletingClient"
        class="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/60 backdrop-blur-xs transition-opacity"
      >
        <div class="bg-slate-900 border border-slate-800 rounded-2xl shadow-2xl max-w-sm w-full p-6 text-slate-100">
          <div class="flex items-center gap-3.5 mb-4">
            <div class="w-11 h-11 rounded-xl bg-rose-500/10 border border-rose-500/30 flex items-center justify-center text-rose-500 shrink-0">
              <i class="fa-solid fa-trash-can text-lg"></i>
            </div>
            <div>
              <h3 class="text-sm font-bold text-white tracking-wide">Hapus Klien (Soft Delete)</h3>
              <p class="text-xs text-slate-400">Konfirmasi Penghapusan</p>
            </div>
          </div>

          <p class="text-xs text-slate-300 mb-2 leading-relaxed">
            Apakah Anda yakin ingin menghapus klien <strong class="text-white">"{{ deletingClient.name }}"</strong>?
          </p>
          <div class="p-2.5 rounded-lg bg-amber-500/10 border border-amber-500/20 text-amber-500 text-[11px] mb-5">
            <i class="fa-solid fa-circle-info mr-1"></i>
            <span>Data tidak akan hilang permanen (soft delete) dan seluruh riwayat pelanggaran tetap aman.</span>
          </div>

          <div class="flex items-center justify-end gap-2.5">
            <button
              @click="deletingClient = null"
              :disabled="isDeleting"
              class="px-4 py-2 text-xs font-medium text-slate-700 dark:text-slate-300 bg-slate-100 dark:bg-slate-800 hover:bg-slate-200 dark:hover:bg-slate-700 rounded-lg transition cursor-pointer"
            >
              Batal
            </button>
            <button
              @click="confirmDeleteClient"
              :disabled="isDeleting"
              class="px-4 py-2 text-xs font-semibold text-white bg-rose-600 hover:bg-rose-500 rounded-lg transition flex items-center gap-2 cursor-pointer shadow-sm disabled:opacity-50"
            >
              <i v-if="isDeleting" class="fa-solid fa-circle-notch fa-spin text-xs"></i>
              <i v-else class="fa-solid fa-trash text-xs"></i>
              <span>{{ isDeleting ? "Menghapus..." : "Ya, Hapus Klien" }}</span>
            </button>
          </div>
        </div>
      </div>
    </Teleport>
  </div>
</template>
