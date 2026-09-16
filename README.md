# go-etle

Integrasi data ETLE Polri (Elektronik Traffic Law Enforcement). Menyimpan banyak client
(Polres/Polda), master pelanggaran, master kamera, dan mengirim data pelanggaran ke
`https://api-etle.polri.go.id` secara realtime maupun via cronjob.

## Arsitektur

```
┌─────────┐    ┌──────────────────────────┐    ┌────────────────────┐
│  Vue FE │───▶│  BFF (Node + better-auth) │───▶│  Go backend        │───▶ ETLE API
│ :5173   │    │  :3000  (auth + proxy)    │    │  :8080 (data/API)  │     /violation/insert
└─────────┘    └──────────────────────────┘    └────────────────────┘
                    │                                  │
                    ▼                                  ▼
              Postgres (etle)                    Postgres (etle)
        (tabel auth better-auth)           (clients, cameras, violations, ...)
```

- **Go backend** — API data, penyimpanan, worker pool pengirim ke ETLE, migrasi skema.
- **BFF (Node)** — better-auth (login FE) + proxy ke Go API. Semua panggilan data FE harus
  sudah login.
- **FE (Vue 3 + Vite + Tailwind)** — dashboard: Clients, Master Pelanggaran, Violations, Cameras.

## Persyaratan

- Go 1.20+
- Node.js 18+ & npm
- PostgreSQL (user `ursa`, db `etle`)

## Setup

### 1. Database

```bash
createdb -U ursa etle
# atau: psql -U ursa -c "CREATE DATABASE etle;"
```

Env ada di `.env` (root project):

```
DB_HOST=localhost
DB_PORT=5432
DB_USERNAME=root
DB_PASSWORD=rahasia
DB_NAME=etle
DB_CONNECTION=pgsql
ETLE_BASE_URL=https://api-etle.polri.go.id
WORKER_COUNT=8
JOB_QUEUE_SIZE=1000
MAX_RETRY=5
PORT=8080
MEDIA_DIR=./storage
RETENTION_DAYS=2
```

### 2. Go backend

```bash
go run .
```

Otomatis migrasi tabel (`clients`, `cameras`, `violations`, `master_violations`, `users`).
Server jalan di `:8080` dan mengirim data secara realtime (worker pool).

### 3. BFF (auth + proxy)

```bash
cd bff
npm install --legacy-peer-deps
npm start          # :3000
```

better-auth membuat tabel auth sendiri (`user`, `session`, `account`, `verification`) di DB `etle`.

### 4. FE

```bash
cd fe
npm install --legacy-peer-deps
npm run dev        # :5173
```

Buka `http://localhost:5173/login` (default login: `admin@etle.local` / `admin123`).
Setelah login, navigasi menggunakan sistem URL menu berbasis Vue Router:
- `http://localhost:5173/dashboard` — Ringkasan metrik, retensi 3 hari, sync 1-klik & log
- `http://localhost:5173/clients` — Manajemen Klien (Polda/Polres)
- `http://localhost:5173/master-pelanggaran` — Daftar Master Pelanggaran Korlantas
- `http://localhost:5173/cameras` — Daftar & Penambahan Kamera
- `http://localhost:5173/violations` — Antrean & Log Pelanggaran

## Perintah CLI (Go)

```bash
go run .                 # jalankan server API + worker realtime
go run . send            # one-shot: kirim semua violations pending lalu exit (cocok cron)
go run . send --daemon   # background worker terus-menerus
go run . cleanup         # hapus data DB & file media (XML, gambar) > 2 hari
go run . cleanup --days 2 --dir ./storage # hapus data DB & media dengan custom hari dan direktori
go run . seed            # isi contoh client + kamera + master pelanggaran
go run . sync            # sinkronisasi master pelanggaran Korlantas ke database
go run . sync --client-id 1 # sinkronisasi master pelanggaran via client ID tertentu
go run . import-xml      # scan file XML dari folder storage & kirim ke Korlantas
go run . import-xml --dir /path/to/xml --watch # daemon watcher realtime untuk folder XML kamera
go run . adduser --username admin --password rahasia --email a@b.c --role admin
```

Contoh cronjob:
```cron
# Kirim data pending tiap 5 menit:
*/5 * * * * cd /path/go-etle && go run . send >> /var/log/etle-send.log 2>&1

# Bersihkan data pelanggaran > 2 hari setiap tengah malam:
0 0 * * * cd /path/go-etle && go run . cleanup --days 2 >> /var/log/etle-cleanup.log 2>&1
```

## API (Go backend)

| Method | Path | Keterangan |
|--------|------|-----------|
| GET | `/dashboard/stats` | ringkasan statistik antrean, status kirim & retensi |
| GET | `/sync-logs` | riwayat log sinkronisasi master pelanggaran |
| POST | `/sync-all` | sinkronisasi master pelanggaran dari semua klien / endpoint resmi |
| POST | `/cleanup?days=2` | hapus data pelanggaran yang lebih lama dari N hari |
| POST | `/clients` | tambah client |
| GET | `/clients` | list client |
| GET | `/clients/{id}` | detail client |
| POST | `/clients/{id}/sync` | tarik master pelanggaran dari `/master/list` |
| POST | `/clients/{id}/login` | simpan `access_token`/`refresh_token` |
| POST | `/cameras` | tambah kamera |
| GET | `/cameras` | list kamera |
| POST | `/violations` | ingest data pelanggaran `{client_id, datas:[...]}` |
| GET | `/violations?status=pending` | list (filter status) |
| GET | `/master` | list master pelanggaran |
| GET/POST | `/users` | list/tambah user (profil) |

FE tidak memanggil Go langsung; semua lewat BFF: `/api/auth/*` (better-auth) dan
`/api/data/*` (proxy ke Go, wajib login).


## Alur pengiriman data ke ETLE

1. Ingest: `POST /violations` → simpan ke tabel `violations` status `pending`.
2. Worker (realtime atau `send`) ambil `pending` secara atomik (`status='processing'`)
   supaya tidak dobel kirim antar-worker/proses.
3. Login: `POST /user/login` dengan `usertoken/passtoken/client_secret/client_id`
   → dapat `access_token`.
4. Kirim: `POST /violation/insert` header `Authorization: Bearer <access_token>`.
5. Update: `status='sent'` + `response_status`, atau `failed`/`pending` (retry sampai `MAX_RETRY`).

`violationCode` disimpan sebagai VARCHAR supaya bisa angka maupun `"PS"`.

## Struktur tabel (ringkas)

- **clients** — 1 baris = 1 Polres/Polda (kredensial ETLE).
- **cameras** — `client_id` FK → clients (1 client banyak kamera), unique `(client_id, device_name)`.
- **violations** — antrian kirim; `client_id` FK→clients, `camera_id` FK→cameras, `violation_code` cocok ke master (logical).
- **master_violations** — hasil `/master/list` (`code` unique, `name`).
- **users** — profil/role aplikasi (auth sebenarnya di better-auth).
- **better-auth** — `user`, `session`, `account`, `verification` (auto-migrate).

Relasi FK:

```
cameras.client_id      -> clients.id      (N:1)
violations.client_id   -> clients.id      (N:1)
violations.camera_id   -> cameras.id      (N:0..1)
violations.violation_code -> master_violations.code (logical, no FK)
```

## Catatan

- FE auth menggunakan better-auth (Node); tabel `users` di Go redundant untuk auth,
  masih dipakai sebagai profil/role dan bisa dihapus bila tidak diperlukan.
- Kredensial client (`usertoken/passtoken/client_secret/client_id`) diberikan tim ETLE —
  isi lewat FE setelah `go run . seed`.
- Channel worker berbatas + `reaper` 10 detik → sistem tidak hang walau banyak perintah.
