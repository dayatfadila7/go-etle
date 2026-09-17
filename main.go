package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"
)

type Handler struct {
	cfg  *Config
	repo *Repository
	etle *ETLEClient
	disp *Dispatcher
}

func main() {
	_ = godotenv.Load()
	cfg := LoadConfig()

	db, err := openDB(cfg)
	if err != nil {
		log.Fatalf("db: %v", err)
	}
	defer db.Close()
	if err := migrate(db); err != nil {
		log.Fatalf("migrate: %v", err)
	}

	repo := NewRepository(db)
	etle := NewETLEClient()

	args := os.Args[1:]
	if len(args) > 0 {
		switch args[0] {
		case "send":
			runSender(cfg, repo, etle, args[1:])
			return
		case "cleanup", "purge":
			runCleanup(cfg, repo, args[1:])
			return
		case "adduser":
			runAddUser(repo, args[1:])
			return
		case "seed":
			runSeed(repo, etle)
			return
		case "sync":
			runSync(repo, etle, args[1:])
			return
		case "import-xml", "scan-xml":
			runImportXML(cfg, repo, etle, args[1:])
			return
		}
	}
	runServer(cfg, repo, etle)
}

// PurgeOldFiles menghapus file dengan ekstensi tertentu yang berumur > days hari dari direktori dir.
func PurgeOldFiles(dir string, days int, exts []string) (int, int64, error) {
	if dir == "" {
		return 0, 0, nil
	}
	info, err := os.Stat(dir)
	if os.IsNotExist(err) || !info.IsDir() {
		return 0, 0, nil
	}
	if days <= 0 {
		days = 2
	}

	threshold := time.Now().Add(-time.Duration(days) * 24 * time.Hour)
	deletedFiles := 0
	var bytesFreed int64 = 0

	extMap := make(map[string]bool)
	for _, e := range exts {
		extMap[strings.ToLower(strings.TrimPrefix(e, "."))] = true
	}

	err = filepath.Walk(dir, func(path string, fi os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if fi.IsDir() {
			return nil
		}

		ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(path), "."))
		if len(extMap) > 0 && !extMap[ext] {
			return nil
		}

		if fi.ModTime().Before(threshold) {
			size := fi.Size()
			if remErr := os.Remove(path); remErr == nil {
				deletedFiles++
				bytesFreed += size
			}
		}
		return nil
	})

	return deletedFiles, bytesFreed, err
}

func runCleanup(cfg *Config, repo *Repository, args []string) {
	fs := flag.NewFlagSet("cleanup", flag.ExitOnError)
	defaultDays := cfg.RetentionDays
	if defaultDays <= 0 {
		defaultDays = 2
	}
	days := fs.Int("days", defaultDays, "Hapus data dan file yang lebih lama dari N hari")
	dir := fs.String("dir", cfg.MediaDir, "Path folder tempat file XML dan gambar disimpan")
	fs.Parse(args)

	// 1. Hapus dari database
	deletedDB, err := repo.PurgeOldViolations(*days)
	if err != nil {
		log.Fatalf("cleanup db error: %v", err)
	}

	// 2. Hapus file fisik XML, gambar (jpg, jpeg, png), dan video (mp4)
	exts := []string{"xml", "jpg", "jpeg", "png", "mp4"}
	deletedFiles, bytesFreed, _ := PurgeOldFiles(*dir, *days, exts)

	mbFreed := float64(bytesFreed) / (1024 * 1024)
	log.Printf("cleanup berhasil: %d data DB dihapus, %d file media (%s) dihapus dari '%s' (menghemat %.2f MB)",
		deletedDB, deletedFiles, strings.Join(exts, ", "), *dir, mbFreed)
}

// send: kirim violation ke ETLE. Jalankan via cronjob.
//
//	go run . send            -> one-shot: drain semua pending lalu exit
//	go run . send --daemon   -> jalan terus (background worker)
func runSender(cfg *Config, repo *Repository, etle *ETLEClient, args []string) {
	daemon := false
	for _, a := range args {
		if a == "--daemon" {
			daemon = true
		}
	}
	disp := NewDispatcher(cfg.WorkerCount, cfg.QueueSize, cfg.MaxRetry, cfg.MaxDelayMinutes, etle, repo, cfg)
	disp.Start()
	if daemon {
		log.Printf("sender daemon running (workers=%d)", cfg.WorkerCount)
		select {}
	}
	log.Println("sender one-shot: draining pending violations")
	disp.DrainOnce()
}

func runAddUser(repo *Repository, args []string) {
	fs := flag.NewFlagSet("adduser", flag.ExitOnError)
	username := fs.String("username", "", "username (wajib)")
	email := fs.String("email", "", "email")
	password := fs.String("password", "", "password (wajib)")
	name := fs.String("name", "", "nama lengkap")
	role := fs.String("role", "operator", "role")
	fs.Parse(args)
	if *username == "" || *password == "" {
		log.Fatal("flag --username dan --password wajib")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(*password), bcrypt.DefaultCost)
	if err != nil {
		log.Fatal(err)
	}
	u := &User{Username: *username, Email: *email, PasswordHash: string(hash), Name: *name, Role: *role, IsActive: true}
	if err := repo.CreateUser(u); err != nil {
		log.Fatal(err)
	}
	log.Printf("user '%s' dibuat (id=%d)", u.Username, u.ID)
}

// seed: isi contoh client + kamera + master pelanggaran agar FE tidak kosong.
// Kredensial client masih placeholder — ganti lewat FE/API sebelum sync & kirim.
func runSeed(repo *Repository, etle *ETLEClient) {
	c := &Client{
		Name:         "Sample Polres",
		ClientID:     "sample",
		ClientSecret: "ganti_dengan_dari_tim_etle",
		Usertoken:    "sample",
		Passtoken:    "sample",
		BaseURL:      "https://api-etle.polri.go.id",
		IsActive:     true,
	}
	if err := repo.CreateClient(c); err != nil {
		log.Fatalf("seed client: %v", err)
	}
	log.Printf("sample client id=%d (isi kredensial asli lewat FE)", c.ID)

	cam := &Camera{ClientID: c.ID, CameraCode: "UNIX1", DeviceName: "CAM-BANGIL-01", LocationName: "Pasuruan", Address: "Jalan A.Yani Bangil Kab.Pasuruan"}
	if err := repo.CreateCamera(cam); err != nil {
		log.Fatalf("seed camera: %v", err)
	}

	// ambil SELURUH master pelanggaran dari endpoint resmi
	items, err := etle.FetchPublicMaster()
	if err != nil {
		log.Fatalf("fetch master: %v", err)
	}
	n, err := repo.UpsertMasterViolations(items)
	if err != nil {
		log.Fatalf("seed master: %v", err)
	}
	log.Printf("master seeded: %d (dari /master/list)", n)
}

func runSync(repo *Repository, etle *ETLEClient, args []string) {
	fs := flag.NewFlagSet("sync", flag.ExitOnError)
	clientID := fs.Int64("client-id", 0, "ID client spesifik (opsional, jika 0 akan pakai client aktif atau public endpoint)")
	fs.Parse(args)

	var items []MasterViolation
	var err error
	clientName := "Public / All"
	var clientIDPtr *int64

	if *clientID > 0 {
		c, getErr := repo.GetClient(*clientID)
		if getErr != nil || c == nil {
			log.Fatalf("client id %d tidak ditemukan", *clientID)
		}
		if !c.IsActive {
			log.Fatalf("client id %d tidak aktif, jalankan sync dengan client aktif", *clientID)
		}
		clientIDPtr = &c.ID
		clientName = c.Name
		log.Printf("sync master menggunakan client '%s' (id=%d)...", c.Name, c.ID)
		items, err = etle.SyncMaster(c)
	} else {
		// Coba client aktif pertama yang punya kredensial/token
		cls, _ := repo.ListClients()
		for _, c := range cls {
			if c.IsActive && (c.AuthToken != "" || c.Usertoken != "") {
				clientIDPtr = &c.ID
				clientName = c.Name
				log.Printf("sync master menggunakan client aktif '%s' (id=%d)...", c.Name, c.ID)
				items, err = etle.SyncMaster(&c)
				break
			}
		}
		// Fallback ke endpoint public jika tidak ada client atau client gagal
		if err != nil || len(items) == 0 {
			log.Println("sync master menggunakan endpoint publik Korlantas (/master/list)...")
			items, err = etle.FetchPublicMaster()
			clientIDPtr = nil
			clientName = "Public Endpoint"
		}
	}

	if err != nil {
		_ = repo.CreateSyncLog(&SyncLog{
			ClientID: clientIDPtr,
			Action:   "sync_master",
			Status:   "failed",
			Message:  err.Error(),
		})
		log.Fatalf("sync gagal: %v", err)
	}

	n, err := repo.UpsertMasterViolations(items)
	if err != nil {
		_ = repo.CreateSyncLog(&SyncLog{
			ClientID: clientIDPtr,
			Action:   "sync_master",
			Status:   "failed",
			Message:  "upsert: " + err.Error(),
		})
		log.Fatalf("simpan master ke database gagal: %v", err)
	}

	_ = repo.CreateSyncLog(&SyncLog{
		ClientID:      clientIDPtr,
		Action:        "sync_master",
		Status:        "success",
		RecordsSynced: n,
		Message:       fmt.Sprintf("berhasil sync %d master pelanggaran via %s", n, clientName),
	})

	log.Printf("sync berhasil! %d master pelanggaran tersinkron ke database (sumber: %s)", n, clientName)
}

func runImportXML(cfg *Config, repo *Repository, etle *ETLEClient, args []string) {
	fs := flag.NewFlagSet("import-xml", flag.ExitOnError)
	dir := fs.String("dir", cfg.MediaDir, "Path folder tempat file XML dan gambar disimpan")
	clientID := fs.Int64("client-id", 0, "ID Client Polda/Polres (default: client aktif pertama)")
	watch := fs.Bool("watch", false, "Pantau folder terus-menerus (daemon realtime watcher)")
	fs.Parse(args)

	var cid int64 = *clientID
	if cid <= 0 {
		cls, _ := repo.ListClients()
		for _, c := range cls {
			if c.IsActive {
				cid = c.ID
				break
			}
		}
		if cid <= 0 {
			log.Fatal("tidak ada client aktif. Daftarkan client terlebih dahulu.")
		}
	} else {
		c, getErr := repo.GetClient(cid)
		if getErr != nil || c == nil {
			log.Fatalf("client id %d tidak ditemukan", cid)
		}
		if !c.IsActive {
			log.Fatalf("client id %d tidak aktif, tidak diproses", cid)
		}
	}

	disp := NewDispatcher(cfg.WorkerCount, cfg.QueueSize, cfg.MaxRetry, cfg.MaxDelayMinutes, etle, repo, cfg)
	disp.Start()

	// Muat master pelanggaran ETLE agar bisa memetakan kode dan nama resmi.
	masters, _ := repo.ListMaster()
	mm := map[string]string{}
	for _, m := range masters {
		mm[m.Code] = m.Name
	}

	processDir := func() (int, int, int) {
		enqueued, skipped, expired := 0, 0, 0
		now := time.Now()
		maxDelay := time.Duration(cfg.MaxDelayMinutes) * time.Minute
		if maxDelay <= 0 {
			maxDelay = 3 * time.Minute
		}

		_ = filepath.Walk(*dir, func(path string, fi os.FileInfo, err error) error {
			if err != nil || fi.IsDir() {
				return nil
			}
			// Hanya proses file .xml yang belum berakhiran .processed
			if !strings.EqualFold(filepath.Ext(path), ".xml") || strings.HasSuffix(path, ".processed") {
				return nil
			}

			item, err := ParseANPRFile(path)
			if err != nil {
				log.Printf("gagal parse xml %s: %v", filepath.Base(path), err)
				skipped++
				return nil
			}

			meta := ParseANPRFilename(path)
			actualCID := cid
			var camID int64
			if meta.CameraCode != "" {
				if cam, err := repo.FindCameraByCode(meta.CameraCode); err == nil && cam != nil {
					camID = cam.ID
					if actualCID == 0 && cam.ClientID > 0 {
						actualCID = cam.ClientID
					}
					if item.DeviceName == meta.CameraCode && cam.DeviceName != "" {
						item.DeviceName = cam.DeviceName
					}
					if item.LocationName == "" && cam.LocationName != "" {
						item.LocationName = cam.LocationName
					}
				}
			}

			// Hanya kirim pelanggaran dengan plat valid (mirip filter di POST /violations).
			// Plat yang tidak terbaca (unknown/kosong) TIDAK di-record sama sekali:
			// dilewati tanpa membuat baris di database, file langsung ditandai processed.
			plate := strings.TrimSpace(item.Plate)
			if plate == "" || strings.EqualFold(plate, "unknown") {
				skipped++
				_ = os.Rename(path, path+".processed")
				return nil
			}

			// Map kode pelanggaran kamera (angka, mis. 1240) ke kode master ETLE
			// Korlantas (huruf, mis. PS). Kode tanpa mapping TIDAK dikirim ke ETLE
			// karena ETLE mencocokkan violationCode dengan daftar masternya.
			etleCode, ok := MapViolationCode(item.ViolationCode)
			if !ok {
				log.Printf("lewatkan %s: violation_code %q tidak punya mapping ke ETLE", filepath.Base(path), item.ViolationCode)
				skipped++
				_ = os.Rename(path, path+".processed")
				return nil
			}
			item.ViolationCode = etleCode
			if name, ok := mm[etleCode]; ok {
				item.ViolationName = name
			}

			capTime := parseCaptureTime(item.CaptureTime)
			isStale := false
			staleReason := ""

			if !capTime.IsZero() {
				if capTime.Year() != now.Year() || capTime.YearDay() != now.YearDay() {
					isStale = true
					staleReason = "expired: capture_time dari hari yang berbeda (bukan hari ini)"
				} else if now.Sub(capTime) > maxDelay {
					isStale = true
					staleReason = fmt.Sprintf("expired: delay capture_time melebihi batas %d menit", cfg.MaxDelayMinutes)
				}
			}

			v := &Violation{
				ClientID:        actualCID,
				CameraID:        camID,
				DeviceName:      item.DeviceName,
				Plate:           plate,
				PlateColor:      item.PlateColor,
				PlateImageURL:   CopyToSnapshots(cfg, item.PlateImageURL),
				VehicleType:     item.VehicleType,
				VehicleColor:    item.VehicleColor,
				VehicleImageURL: CopyToSnapshots(cfg, item.VehicleImageURL),
				ViolationCode:   item.ViolationCode,
				ViolationName:   item.ViolationName,
				LocationName:    item.LocationName,
				CaptureTime:     item.CaptureTime,
			}

			if isStale {
				v.Status = "failed"
				v.ErrorMessage = staleReason
				_, _ = repo.EnqueueViolation(v)
				expired++
			} else {
				id, err := repo.EnqueueViolation(v)
				if err == nil {
					disp.Submit(Job{ViolationID: id, ClientID: cid})
					enqueued++
				}
			}

			// Rename file XML agar tidak diproses ganda
			_ = os.Rename(path, path+".processed")
			return nil
		})

		return enqueued, skipped, expired
	}

	if *watch {
		log.Printf("memulai ANPR XML watcher pada folder '%s' (client_id=%d, toleransi delay=%d menit)...", *dir, cid, cfg.MaxDelayMinutes)
		ticker := time.NewTicker(2 * time.Second)
		for range ticker.C {
			enq, _, exp := processDir()
			if enq > 0 || exp > 0 {
				log.Printf("[watcher] %d pelanggaran baru di-enqueue, %d kadaluarsa (expired)", enq, exp)
			}
		}
	} else {
		log.Printf("memindai file XML pada '%s'...", *dir)
		enq, skip, exp := processDir()
		log.Printf("scan selesai: %d di-enqueue ke antrean kirim, %d dilewati/error, %d kadaluarsa (expired)", enq, skip, exp)
		time.Sleep(2 * time.Second)
	}
}

func runServer(cfg *Config, repo *Repository, etle *ETLEClient) {
	// server jalan realtime: ingest -> enqueue -> worker kirim.
	disp := NewDispatcher(cfg.WorkerCount, cfg.QueueSize, cfg.MaxRetry, cfg.MaxDelayMinutes, etle, repo, cfg)
	disp.Start()

	h := &Handler{cfg: cfg, repo: repo, etle: etle, disp: disp}
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("ok")) })
	mux.Handle("/media/", http.StripPrefix("/media/", serveMedia(cfg.MediaDir)))
	mux.HandleFunc("/dashboard/stats", h.handleDashboardStats)
	mux.HandleFunc("/sync-logs", h.handleSyncLogs)
	mux.HandleFunc("/sync-all", h.handleSyncAll)
	mux.HandleFunc("/cleanup", h.handleCleanup)
	mux.HandleFunc("/send", h.handleSendPending)
	mux.HandleFunc("/clients", h.handleClients)
	mux.HandleFunc("/clients/", h.handleClients)
	mux.HandleFunc("/cameras", h.handleCameras)
	mux.HandleFunc("/cameras/", h.handleCameras)
	mux.HandleFunc("/violations", h.handleViolations)
	mux.HandleFunc("/violations/{id}", h.handleViolationDetail)
	mux.HandleFunc("/violations/{id}/send", h.handleSendOne)
	mux.HandleFunc("/monitoring", h.handleMonitoring)
	mux.HandleFunc("/master", h.handleMaster)
	mux.HandleFunc("/users", h.handleUsers)

	addr := ":" + strconv.Itoa(cfg.Port)
	log.Printf("go-etle server %s (workers=%d)", addr, cfg.WorkerCount)
	log.Fatal(http.ListenAndServe(addr, mux))
}

func writeJSON(w http.ResponseWriter, code int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(v)
}

// serveMedia hanya menampilkan file media (gambar/video ANPR) dari MediaDir,
// tanpa directory listing dan tanpa path keluar dari root.
func serveMedia(root string) http.Handler {
	if root == "" {
		root = "./storage"
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rel := strings.TrimPrefix(r.URL.Path, "/")
		rel = strings.ReplaceAll(rel, "\\", "/")
		abs := filepath.Join(root, rel)
		if _, relOK := mediaRel(root, abs); !relOK {
			http.NotFound(w, r)
			return
		}
		fi, err := os.Stat(abs)
		if err != nil || fi.IsDir() {
			http.NotFound(w, r)
			return
		}
		switch strings.ToLower(filepath.Ext(abs)) {
		case ".jpg", ".jpeg", ".png", ".gif", ".webp", ".bmp", ".mp4", ".mp3":
		default:
			http.NotFound(w, r)
			return
		}
		http.ServeFile(w, r, abs)
	})
}

func readJSON(r *http.Request, v interface{}) error {
	b, err := io.ReadAll(r.Body)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, v)
}

// ---------- clients ----------

func (h *Handler) handleClients(w http.ResponseWriter, r *http.Request) {
	cleanPath := strings.Trim(r.URL.Path, "/")
	parts := strings.Split(cleanPath, "/")
	var id int64
	if len(parts) >= 2 && parts[0] == "clients" {
		id, _ = strconv.ParseInt(parts[1], 10, 64)
	}
	if id == 0 {
		id, _ = strconv.ParseInt(r.URL.Query().Get("id"), 10, 64)
	}

	if len(parts) >= 3 && parts[0] == "clients" {
		switch parts[2] {
		case "sync":
			h.syncMaster(w, r, id)
			return
		case "login":
			h.setToken(w, r, id)
			return
		}
	}

	switch r.Method {
	case http.MethodGet:
		if id > 0 {
			c, err := h.repo.GetClient(id)
			if err != nil {
				http.Error(w, err.Error(), 500)
				return
			}
			if c == nil {
				http.NotFound(w, r)
				return
			}
			writeJSON(w, 200, c)
			return
		}
		h.listClients(w, r)

	case http.MethodPost:
		h.createClient(w, r)

	case http.MethodPut, http.MethodPatch:
		var c Client
		if err := readJSON(r, &c); err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		if id > 0 {
			c.ID = id
		}
		if c.ID == 0 {
			http.Error(w, "client id required", 400)
			return
		}
		if c.Name == "" {
			http.Error(w, "name required", 400)
			return
		}
		if err := h.repo.UpdateClient(&c); err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		writeJSON(w, 200, c)

	case http.MethodDelete:
		if id == 0 {
			http.Error(w, "client id required", 400)
			return
		}
		if err := h.repo.DeleteClient(id); err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		writeJSON(w, 200, map[string]interface{}{"status": "deleted", "id": id})

	default:
		http.Error(w, "method not allowed", 405)
	}
}

func (h *Handler) createClient(w http.ResponseWriter, r *http.Request) {
	var c Client
	if err := readJSON(r, &c); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	if c.Name == "" {
		http.Error(w, "name required", 400)
		return
	}
	if err := h.repo.CreateClient(&c); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	writeJSON(w, 201, c)
}

func (h *Handler) listClients(w http.ResponseWriter, r *http.Request) {
	cs, err := h.repo.ListClients()
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	writeJSON(w, 200, cs)
}

func (h *Handler) syncMaster(w http.ResponseWriter, r *http.Request, id int64) {
	c, err := h.repo.GetClient(id)
	if err != nil || c == nil {
		http.Error(w, "client not found", 404)
		return
	}
	if !c.IsActive {
		http.Error(w, "client tidak aktif", 400)
		return
	}
	items, err := h.etle.SyncMaster(c)
	if err != nil {
		_ = h.repo.CreateSyncLog(&SyncLog{
			ClientID: &id,
			Action:   "sync_master",
			Status:   "failed",
			Message:  err.Error(),
		})
		http.Error(w, "sync failed: "+err.Error(), 502)
		return
	}
	n, err := h.repo.UpsertMasterViolations(items)
	if err != nil {
		_ = h.repo.CreateSyncLog(&SyncLog{
			ClientID: &id,
			Action:   "sync_master",
			Status:   "failed",
			Message:  err.Error(),
		})
		http.Error(w, err.Error(), 500)
		return
	}
	_ = h.repo.CreateSyncLog(&SyncLog{
		ClientID:      &id,
		Action:        "sync_master",
		Status:        "success",
		RecordsSynced: n,
		Message:       fmt.Sprintf("Berhasil sinkronisasi %d master dari %s", n, c.Name),
	})
	writeJSON(w, 200, map[string]interface{}{"synced": n, "items": items})
}

func (h *Handler) setToken(w http.ResponseWriter, r *http.Request, id int64) {
	var body struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
	}
	if err := readJSON(r, &body); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	if err := h.repo.SetTokens(id, body.AccessToken, body.RefreshToken); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	writeJSON(w, 200, map[string]string{"status": "ok"})
}

// ---------- cameras ----------

func (h *Handler) handleCameras(w http.ResponseWriter, r *http.Request) {
	cleanPath := strings.Trim(r.URL.Path, "/")
	parts := strings.Split(cleanPath, "/")
	var id int64
	if len(parts) >= 2 && parts[0] == "cameras" {
		id, _ = strconv.ParseInt(parts[1], 10, 64)
	}
	if id == 0 {
		id, _ = strconv.ParseInt(r.URL.Query().Get("id"), 10, 64)
	}

	switch r.Method {
	case http.MethodGet:
		if id > 0 {
			c, err := h.repo.GetCamera(id)
			if err != nil {
				http.Error(w, err.Error(), 500)
				return
			}
			if c == nil {
				http.NotFound(w, r)
				return
			}
			writeJSON(w, 200, c)
			return
		}
		clientID, _ := strconv.ParseInt(r.URL.Query().Get("client_id"), 10, 64)
		cs, err := h.repo.ListCameras(clientID)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		writeJSON(w, 200, cs)

	case http.MethodPost:
		var c Camera
		if err := readJSON(r, &c); err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		if c.ClientID == 0 || c.DeviceName == "" {
			http.Error(w, "client_id and device_name required", 400)
			return
		}
		if err := h.repo.CreateCamera(&c); err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		writeJSON(w, 201, c)

	case http.MethodPut, http.MethodPatch:
		var c Camera
		if err := readJSON(r, &c); err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		if id > 0 {
			c.ID = id
		}
		if c.ID == 0 {
			http.Error(w, "camera id required", 400)
			return
		}
		if c.ClientID == 0 || c.DeviceName == "" {
			http.Error(w, "client_id and device_name required", 400)
			return
		}
		if err := h.repo.UpdateCamera(&c); err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		writeJSON(w, 200, c)

	case http.MethodDelete:
		if id == 0 {
			http.Error(w, "camera id required", 400)
			return
		}
		if err := h.repo.DeleteCamera(id); err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		writeJSON(w, 200, map[string]interface{}{"status": "deleted", "id": id})

	default:
		http.Error(w, "method not allowed", 405)
	}
}

// ---------- violations ----------

func (h *Handler) handleViolations(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		h.ingestViolations(w, r)
	case http.MethodGet:
		status := r.URL.Query().Get("status")
		page, _ := strconv.Atoi(r.URL.Query().Get("page"))
		perPage, _ := strconv.Atoi(r.URL.Query().Get("per_page"))
		if page < 1 {
			page = 1
		}
		if perPage <= 0 {
			perPage = 20
		}
		vs, total, err := h.repo.ListViolations(status, page, perPage)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		writeJSON(w, 200, map[string]interface{}{
			"items":    vs,
			"total":    total,
			"page":     page,
			"per_page": perPage,
		})
	default:
		http.Error(w, "method not allowed", 405)
	}
}

func parseCaptureTime(t int64) time.Time {
	if t <= 0 {
		return time.Time{}
	}
	if t > 1e11 { // milliseconds
		return time.UnixMilli(t)
	}
	return time.Unix(t, 0)
}

func (h *Handler) ingestViolations(w http.ResponseWriter, r *http.Request) {
	var in struct {
		ClientID int64           `json:"client_id"`
		Datas    []ViolationItem `json:"datas"`
	}
	if err := readJSON(r, &in); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	if in.ClientID == 0 {
		http.Error(w, "client_id required", 400)
		return
	}
	if len(in.Datas) == 0 {
		http.Error(w, "datas empty", 400)
		return
	}
	c, err := h.repo.GetClient(in.ClientID)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	if c == nil {
		http.Error(w, "client not found", 400)
		return
	}
	if !c.IsActive {
		http.Error(w, "client tidak aktif", 400)
		return
	}
	// load master agar bisa filter code yang unknown (tidak diproses)
	masters, err := h.repo.ListMaster()
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	mm := map[string]string{}
	for _, m := range masters {
		mm[m.Code] = m.Name
	}

	enqueued, skipped, expired := 0, 0, 0
	now := time.Now()
	maxDelay := time.Duration(h.disp.maxDelayMinutes) * time.Minute
	if maxDelay <= 0 {
		maxDelay = 3 * time.Minute
	}

	for _, it := range in.Datas {
		code := strings.TrimSpace(it.ViolationCode)
		plate := strings.TrimSpace(it.Plate)
		// Map kode kamera (angka) ke kode master ETLE Korlantas (huruf) bila ada.
		if mapped, ok := MapViolationCode(code); ok {
			code = mapped
			it.ViolationCode = mapped
		}
		if code == "" || mm[code] == "" {
			// violation_code unknown / kosong -> jangan diproses
			skipped++
			continue
		}
		if plate == "" || strings.EqualFold(plate, "unknown") {
			// plate unknown / kosong -> jangan diproses
			skipped++
			continue
		}
		if it.ViolationName == "" {
			it.ViolationName = mm[code]
		}

		capTime := parseCaptureTime(it.CaptureTime)
		isStale := false
		staleReason := ""

		if !capTime.IsZero() {
			if capTime.Year() != now.Year() || capTime.YearDay() != now.YearDay() {
				isStale = true
				staleReason = "expired: capture_time dari hari yang berbeda (bukan hari ini)"
			} else if now.Sub(capTime) > maxDelay {
				isStale = true
				staleReason = fmt.Sprintf("expired: delay capture_time melebihi batas %d menit", h.disp.maxDelayMinutes)
			}
		}

		v := &Violation{
			ClientID:        in.ClientID,
			DeviceName:      it.DeviceName,
			Plate:           plate,
			PlateColor:      it.PlateColor,
			PlateImageURL:   BuildMediaURL(h.cfg, it.PlateImageURL),
			VehicleType:     it.VehicleType,
			VehicleColor:    it.VehicleColor,
			VehicleImageURL: BuildMediaURL(h.cfg, it.VehicleImageURL),
			VideoURL:        it.VideoURL,
			ViolationCode:   code,
			ViolationName:   it.ViolationName,
			LocationName:    it.LocationName,
			CaptureTime:     it.CaptureTime,
		}

		if isStale {
			v.Status = "failed"
			v.ErrorMessage = staleReason
			if _, err := h.repo.EnqueueViolation(v); err != nil {
				http.Error(w, err.Error(), 500)
				return
			}
			expired++
			continue
		}

		id, err := h.repo.EnqueueViolation(v)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		h.disp.Submit(Job{ViolationID: id, ClientID: in.ClientID})
		enqueued++
	}
	writeJSON(w, 202, map[string]interface{}{"enqueued": enqueued, "skipped": skipped, "expired": expired, "status": "processed"})
}

// ---------- master ----------

func (h *Handler) handleMaster(w http.ResponseWriter, r *http.Request) {
	ms, err := h.repo.ListMaster()
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	writeJSON(w, 200, ms)
}

// ---------- users ----------

func (h *Handler) handleUsers(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		us, err := h.repo.ListUsers()
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		writeJSON(w, 200, us)
	case http.MethodPost:
		var in struct {
			Username string `json:"username"`
			Email    string `json:"email"`
			Password string `json:"password"`
			Name     string `json:"name"`
			Role     string `json:"role"`
		}
		if err := readJSON(r, &in); err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		if in.Username == "" || in.Password == "" {
			http.Error(w, "username and password required", 400)
			return
		}
		hash, err := bcrypt.GenerateFromPassword([]byte(in.Password), bcrypt.DefaultCost)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		role := in.Role
		if role == "" {
			role = "operator"
		}
		u := &User{Username: in.Username, Email: in.Email, PasswordHash: string(hash), Name: in.Name, Role: role, IsActive: true}
		if err := h.repo.CreateUser(u); err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		writeJSON(w, 201, map[string]interface{}{"id": u.ID, "username": u.Username})
	default:
		http.Error(w, "method not allowed", 405)
	}
}

// ---------- dashboard, sync & cleanup ----------

func (h *Handler) handleDashboardStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", 405)
		return
	}
	stats, err := h.repo.GetDashboardStats()
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	writeJSON(w, 200, stats)
}

func (h *Handler) handleSyncLogs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", 405)
		return
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	logs, err := h.repo.ListSyncLogs(limit)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	writeJSON(w, 200, logs)
}

// handleViolationDetail: detail satu pelanggaran lengkap (termasuk gambar).
func (h *Handler) handleViolationDetail(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", 405)
		return
	}
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid id", 400)
		return
	}
	vd, err := h.repo.GetViolationDetail(id)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	if vd == nil {
		http.Error(w, "not found", 404)
		return
	}
	writeJSON(w, 200, vd)
}

// handleMonitoring: membaca log monitoring kinerja (TSV) dan mengembalikannya
// sebagai JSON untuk ditampilkan di halaman Monitoring web.
func (h *Handler) handleMonitoring(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", 405)
		return
	}
	sys := readLogTSV("/var/log/etle-system-perf.log", 60)
	dbp := readLogTSV("/var/log/etle-db-perf.log", 60)
	del := readLogTSV("/var/log/etle-delivery.log", 100)
	writeJSON(w, 200, map[string]interface{}{
		"system":        sys,
		"db":            dbp,
		"delivery":      del,
		"system_file":   "/var/log/etle-system-perf.log",
		"db_file":       "/var/log/etle-db-perf.log",
		"delivery_file": "/var/log/etle-delivery.log",
	})
}

// readLogTSV membaca file TSV ber-headbaris dan mengembalikan max baris terakhir
// sebagai array object (key = nama kolom di header).
func readLogTSV(path string, max int) []map[string]string {
	f, err := os.Open(path)
	if err != nil {
		return []map[string]string{}
	}
	defer f.Close()
	lines, err := readLines(f)
	if err != nil || len(lines) == 0 {
		return []map[string]string{}
	}
	headers := strings.Split(lines[0], "\t")
	body := lines[1:]
	if len(body) > max {
		body = body[len(body)-max:]
	}
	out := make([]map[string]string, 0, len(body))
	for _, ln := range body {
		cols := strings.Split(ln, "\t")
		row := make(map[string]string, len(headers))
		for i, h := range headers {
			if i < len(cols) {
				row[h] = cols[i]
			}
		}
		if len(cols) == len(headers) {
			out = append(out, row)
		}
	}
	return out
}

func readLines(f *os.File) ([]string, error) {
	var lines []string
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 64*1024), 1024*1024)
	for sc.Scan() {
		lines = append(lines, sc.Text())
	}
	return lines, sc.Err()
}

func (h *Handler) handleSyncAll(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", 405)
		return
	}
	var items []MasterViolation
	var lastErr error

	// 1. Coba sync via client aktif pertama yang punya kredensial
	clients, _ := h.repo.ListClients()
	for _, c := range clients {
		if c.IsActive {
			res, err := h.etle.SyncMaster(&c)
			if err == nil && len(res) > 0 {
				items = res
				lastErr = nil
				break
			}
			if err != nil {
				lastErr = err
			}
		}
	}

	// 2. Jika client belum ada / gagal, coba public master endpoint
	if len(items) == 0 {
		res, err := h.etle.FetchPublicMaster()
		if err == nil && len(res) > 0 {
			items = res
			lastErr = nil
		} else if err != nil {
			lastErr = err
		}
	}

	if len(items) == 0 && lastErr != nil {
		_ = h.repo.CreateSyncLog(&SyncLog{
			Action:  "sync_master",
			Status:  "failed",
			Message: lastErr.Error(),
		})
		http.Error(w, "sync failed: "+lastErr.Error(), 502)
		return
	}

	n, err := h.repo.UpsertMasterViolations(items)
	if err != nil {
		_ = h.repo.CreateSyncLog(&SyncLog{
			Action:  "sync_master",
			Status:  "failed",
			Message: err.Error(),
		})
		http.Error(w, err.Error(), 500)
		return
	}

	msg := fmt.Sprintf("Berhasil sinkronisasi %d master pelanggaran", n)
	_ = h.repo.CreateSyncLog(&SyncLog{
		Action:        "sync_master",
		Status:        "success",
		RecordsSynced: n,
		Message:       msg,
	})
	writeJSON(w, 200, map[string]interface{}{"status": "ok", "synced": n, "items": len(items), "message": msg})
}

// handleSendPending mengirim seluruh antrean pending ke ETLE secara batch
// (beberapa pelanggaran per request) dengan token on-the-fly. Endpoint ini
// dipakai untuk pengiriman manual/on-demand ketika data menumpuk.
func (h *Handler) handleSendPending(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", 405)
		return
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	summary, err := h.disp.SendPending(limit)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	writeJSON(w, 200, summary)
}

// handleSendOne mengirim satu pelanggaran tertentu ke ETLE saat itu juga.
func (h *Handler) handleSendOne(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", 405)
		return
	}
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid id", 400)
		return
	}
	summary, err := h.disp.SendOne(id)
	if err != nil {
		writeJSON(w, 400, map[string]interface{}{"error": err.Error(), "summary": summary})
		return
	}
	writeJSON(w, 200, summary)
}

// handleCleanup menghapus data pelanggaran lama beserta file medianya.
func (h *Handler) handleCleanup(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", 405)
		return
	}
	days := 2
	if h.cfg != nil && h.cfg.RetentionDays > 0 {
		days = h.cfg.RetentionDays
	}
	if d := r.URL.Query().Get("days"); d != "" {
		if n, err := strconv.Atoi(d); err == nil && n > 0 {
			days = n
		}
	}
	dir := "./storage"
	if h.cfg != nil && h.cfg.MediaDir != "" {
		dir = h.cfg.MediaDir
	}
	if d := r.URL.Query().Get("dir"); d != "" {
		dir = d
	}

	deletedDB, err := h.repo.PurgeOldViolations(days)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	exts := []string{"xml", "jpg", "jpeg", "png", "mp4"}
	deletedFiles, bytesFreed, _ := PurgeOldFiles(dir, days, exts)

	mbFreed := float64(bytesFreed) / (1024 * 1024)
	msg := fmt.Sprintf("%d data pelanggaran DB dan %d file media (XML/gambar) yang lebih lama dari %d hari berhasil dihapus (ruang hemat: %.2f MB)",
		deletedDB, deletedFiles, days, mbFreed)
	writeJSON(w, 200, map[string]interface{}{
		"status":        "ok",
		"deleted_db":    deletedDB,
		"deleted_files": deletedFiles,
		"bytes_freed":   bytesFreed,
		"days":          days,
		"dir":           dir,
		"message":       msg,
	})
}
