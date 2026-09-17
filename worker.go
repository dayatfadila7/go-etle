package main

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"
)

// deliveryLogFile = log pengiriman ETLE (TSV). Satu baris per upaya kirim.
var deliveryLogFile = "/var/log/etle-delivery.log"

func (d *Dispatcher) logDelivery(v *Violation, result string, attempt, httpStatus, etleStatus int, errMsg string, dur time.Duration) {
	line := fmt.Sprintf("%s\t%s\t%d\t%s\t%s\t%d\t%s\t%d\t%d\t%d\t%s\t%d\n",
		time.Now().Format(time.RFC3339),
		result,
		v.ID,
		v.Plate,
		v.DeviceName,
		attempt,
		v.ViolationCode,
		v.CaptureTime,
		httpStatus,
		etleStatus,
		errMsg,
		dur.Milliseconds(),
	)
	f, err := os.OpenFile(deliveryLogFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer f.Close()
	if fi, e := f.Stat(); e == nil && fi.Size() == 0 {
		_, _ = f.WriteString("time\tresult\tid\tplate\tdevice\tattempt\tviolation_code\tcapture_time_ms\thttp_status\tetle_status\terror\tduration_ms\n")
	}
	_, _ = f.WriteString(line)
}

type Job struct {
	ViolationID int64
	ClientID    int64
}

// Dispatcher = orchestrator. Job masuk channel lalu disebar (rotator) ke N worker
// yang bebas dipakai ulang (swap). Channel berbatas -> sistem tak pernah hang.
// Claim atomik di DB mencegah 2 proses/worker kirim row sama.
type Dispatcher struct {
	jobChan         chan Job
	workers         int
	etle            *ETLEClient
	repo            *Repository
	maxRetry        int
	maxDelayMinutes int
	mediaURL        func(string) string
}

func NewDispatcher(workers, queueSize, maxRetry, maxDelayMinutes int, etle *ETLEClient, repo *Repository, cfg *Config) *Dispatcher {
	if maxDelayMinutes <= 0 {
		maxDelayMinutes = 3
	}
	var mfunc func(string) string
	if cfg != nil {
		mfunc = func(p string) string { return CopyToSnapshots(cfg, p) }
	}
	return &Dispatcher{
		jobChan:         make(chan Job, queueSize),
		workers:         workers,
		etle:            etle,
		repo:            repo,
		maxRetry:        maxRetry,
		maxDelayMinutes: maxDelayMinutes,
		mediaURL:        mfunc,
	}
}

func (d *Dispatcher) Start() {
	for i := 0; i < d.workers; i++ {
		go d.worker(i)
	}
	go d.reaper()
}

func (d *Dispatcher) Submit(j Job) {
	select {
	case d.jobChan <- j:
	default:
		log.Printf("queue full, leave pending id=%d", j.ViolationID)
	}
}

func (d *Dispatcher) worker(id int) {
	for j := range d.jobChan {
		d.process(j)
	}
}

func (d *Dispatcher) process(j Job) {
	v, err := d.repo.GetViolation(j.ViolationID)
	if err != nil || v == nil {
		return
	}
	claimed, err := d.repo.ClaimViolation(j.ViolationID)
	if err != nil || !claimed {
		return // sudah diambil worker/proses lain
	}

	// Pre-flight check: validasi batas waktu real-time (maksimal d.maxDelayMinutes & tidak beda hari)
	now := time.Now()
	maxDelay := time.Duration(d.maxDelayMinutes) * time.Minute
	if maxDelay <= 0 {
		maxDelay = 3 * time.Minute
	}
	isCrossDay := v.CreatedAt.Year() != now.Year() || v.CreatedAt.YearDay() != now.YearDay()
	if isCrossDay || now.Sub(v.CreatedAt) > maxDelay {
		_ = d.repo.MarkFailed(j.ViolationID, "expired: delay melebihi batas atau beda hari kalender")
		d.logDelivery(v, "expired", v.Attempts, 0, 0, "delay melebihi batas atau beda hari", 0)
		return
	}

	c, err := d.repo.GetClient(j.ClientID)
	if err != nil || c == nil {
		_ = d.repo.MarkFailed(j.ViolationID, "client not found")
		d.logDelivery(v, "failed", v.Attempts, 0, 0, "client not found", 0)
		return
	}
	if !c.IsActive {
		_ = d.repo.MarkFailed(j.ViolationID, "client tidak aktif — dilewati")
		d.logDelivery(v, "skipped_inactive", v.Attempts, 0, 0, "client tidak aktif", 0)
		return
	}

	item, reason := d.prepareItem(v)
	if reason != "" {
		_ = d.repo.MarkFailed(j.ViolationID, "dilewati: "+reason+". Tidak dikirim ke ETLE.")
		d.logDelivery(v, "skipped_unknown", v.Attempts, 0, 0, reason, 0)
		return
	}

	start := time.Now()
	resp, err := d.deliver(c, []ViolationItem{item})
	dur := time.Since(start)
	if err != nil {
		httpStatus := 0
		if apiErr, ok := err.(*APIError); ok {
			httpStatus = apiErr.StatusCode
		}
		_ = d.repo.FailOrRetry(j.ViolationID, v.Attempts, d.maxRetry, err.Error())
		d.logDelivery(v, "failed", v.Attempts+1, httpStatus, 0, err.Error(), dur)
		return
	}
	_ = d.repo.MarkSent(j.ViolationID, resp.Status)
	d.logDelivery(v, "sent", v.Attempts+1, 200, resp.Status, "", dur)
}

// maxBatchSize = jumlah maksimal item pelanggaran dalam satu request /violation/insert.
const maxBatchSize = 50

// SendSummary merangkum hasil satu kali pengiriman (batch atau per-item).
type SendSummary struct {
	Requested int `json:"requested"`
	Claimed   int `json:"claimed"`
	Sent      int `json:"sent"`
	Failed    int `json:"failed"`
	Skipped   int `json:"skipped"`
	Retry     int `json:"retry"`
	Batches   int `json:"batches"`
}

// ensureFreshToken memastikan klien punya access token yang belum kedaluwarsa;
// bila kosong/expired, login ulang on-the-fly dan simpan token barunya.
func (d *Dispatcher) ensureFreshToken(c *Client) {
	if c.AuthToken != "" && !TokenExpired(c.AuthToken) {
		return
	}
	if lr, e := d.etle.Login(c); e == nil {
		c.AuthToken = lr.AccessToken
		_ = d.repo.SetTokens(c.ID, lr.AccessToken, lr.RefreshToken)
	}
}

// deliver mengirim payload ke ETLE. Bila ETLE membalas token kedaluwarsa
// (401/403 jwt expired), login ulang lalu kirim ulang sekali.
func (d *Dispatcher) deliver(c *Client, items []ViolationItem) (*SendResponse, error) {
	d.ensureFreshToken(c)
	resp, err := d.etle.SendViolation(c, ViolationPayload{Datas: items})
	if err != nil && IsTokenExpired(err) {
		if lr, e := d.etle.Login(c); e == nil {
			_ = d.repo.SetTokens(c.ID, lr.AccessToken, lr.RefreshToken)
			c.AuthToken = lr.AccessToken
			resp, err = d.etle.SendViolation(c, ViolationPayload{Datas: items})
		}
	}
	return resp, err
}

// prepareItem membangun payload ETLE dari satu violation: mapping kode master,
// URL media snapshot, dan lokasi dari kamera. Mengembalikan alasan (non-kosong)
// bila violation harus dilewati.
func (d *Dispatcher) prepareItem(v *Violation) (ViolationItem, string) {
	item := ViolationItem{
		DeviceName:      v.DeviceName,
		Plate:           v.Plate,
		PlateColor:      v.PlateColor,
		PlateImageURL:   v.PlateImageURL,
		VehicleType:     v.VehicleType,
		VehicleColor:    v.VehicleColor,
		VehicleImageURL: v.VehicleImageURL,
		VideoURL:        v.VideoURL,
		ViolationCode:   v.ViolationCode,
		ViolationName:   v.ViolationName,
		LocationName:    v.LocationName,
		CaptureTime:     v.CaptureTime,
	}
	// Safety net: pastikan yang dikirim selalu kode master ETLE (huruf).
	if mapped, ok := MapViolationCode(item.ViolationCode); ok {
		item.ViolationCode = mapped
	}
	if d.mediaURL != nil {
		item.PlateImageURL = d.mediaURL(item.PlateImageURL)
		item.VehicleImageURL = d.mediaURL(item.VehicleImageURL)
	}
	// Lokasi wajib sama persis (ETLE memakai "=" bukan LIKE) -> ambil dari kamera bila kosong.
	if item.LocationName == "" && v.CameraID > 0 {
		if cam, e := d.repo.GetCameraByID(v.CameraID); e == nil && cam != nil && cam.LocationName != "" {
			item.LocationName = cam.LocationName
		}
	}
	// antivirus: jangan kirim row dengan plat tidak valid ke ETLE
	plate := strings.TrimSpace(item.Plate)
	if plate == "" || strings.EqualFold(plate, "unknown") {
		return item, "plat nomor tidak terbaca (unknown)"
	}
	return item, ""
}

// SendPending mengirim antrean pending ke ETLE secara batch (beberapa pelanggaran
// per satu request). Setiap row di-claim atomik sehingga aman berjalan bersamaan
// dengan worker/reaper. Bila satu batch gagal, item dikirim ulang satu-per-satu
// untuk mengisolasi error tanpa menggagalkan seluruh batch.
func (d *Dispatcher) SendPending(limit int) (*SendSummary, error) {
	if limit <= 0 {
		limit = 200
	}
	if limit > 2000 {
		limit = 2000
	}
	rows, err := d.repo.PendingViolations(limit, d.maxRetry, d.maxDelayMinutes)
	if err != nil {
		return nil, err
	}
	s := &SendSummary{Requested: len(rows)}
	now := time.Now()
	maxDelay := time.Duration(d.maxDelayMinutes) * time.Minute
	if maxDelay <= 0 {
		maxDelay = 3 * time.Minute
	}

	grouped := map[int64][]*Violation{}
	for i := range rows {
		v := &rows[i]
		claimed, e := d.repo.ClaimViolation(v.ID)
		if e != nil || !claimed {
			continue
		}
		s.Claimed++
		isCrossDay := v.CreatedAt.Year() != now.Year() || v.CreatedAt.YearDay() != now.YearDay()
		if isCrossDay || now.Sub(v.CreatedAt) > maxDelay {
			_ = d.repo.MarkFailed(v.ID, "expired: delay melebihi batas atau beda hari kalender")
			d.logDelivery(v, "expired", v.Attempts, 0, 0, "delay melebihi batas atau beda hari", 0)
			s.Failed++
			continue
		}
		grouped[v.ClientID] = append(grouped[v.ClientID], v)
	}
	d.sendGrouped(grouped, s)
	return s, nil
}

// SendOne mengirim satu violation tertentu sekarang (on-the-fly). Row berstatus
// failed akan di-requeue lebih dulu agar bisa dicoba ulang manual.
func (d *Dispatcher) SendOne(id int64) (*SendSummary, error) {
	v, err := d.repo.GetViolation(id)
	if err != nil {
		return nil, err
	}
	if v == nil {
		return nil, fmt.Errorf("violation id %d tidak ditemukan", id)
	}
	s := &SendSummary{Requested: 1}
	if v.Status == "sent" {
		return s, fmt.Errorf("violation id %d sudah terkirim", id)
	}
	if v.Status == "failed" {
		_ = d.repo.RequeueViolation(id)
	}
	claimed, err := d.repo.ClaimViolation(id)
	if err != nil {
		return s, err
	}
	if !claimed {
		return s, fmt.Errorf("violation id %d sedang diproses atau tidak pending", id)
	}
	s.Claimed = 1

	now := time.Now()
	maxDelay := time.Duration(d.maxDelayMinutes) * time.Minute
	if maxDelay <= 0 {
		maxDelay = 3 * time.Minute
	}
	isCrossDay := v.CreatedAt.Year() != now.Year() || v.CreatedAt.YearDay() != now.YearDay()
	if isCrossDay || now.Sub(v.CreatedAt) > maxDelay {
		_ = d.repo.MarkFailed(id, "expired: delay melebihi batas atau beda hari kalender")
		d.logDelivery(v, "expired", v.Attempts, 0, 0, "delay melebihi batas atau beda hari", 0)
		s.Failed++
		return s, nil
	}
	d.sendGrouped(map[int64][]*Violation{v.ClientID: {v}}, s)
	return s, nil
}

// sendGrouped mengirim violation yang sudah dikelompokkan per klien. Satu request
// ke ETLE hanya memuat violation dari satu klien (token berbeda per klien).
func (d *Dispatcher) sendGrouped(grouped map[int64][]*Violation, s *SendSummary) {
	for clientID, vs := range grouped {
		c, e := d.repo.GetClient(clientID)
		if e != nil || c == nil {
			for _, v := range vs {
				_ = d.repo.MarkFailed(v.ID, "client not found")
				d.logDelivery(v, "failed", v.Attempts, 0, 0, "client not found", 0)
				s.Failed++
			}
			continue
		}
		if !c.IsActive {
			for _, v := range vs {
				_ = d.repo.MarkFailed(v.ID, "client tidak aktif — dilewati")
				d.logDelivery(v, "skipped_inactive", v.Attempts, 0, 0, "client tidak aktif", 0)
				s.Skipped++
			}
			continue
		}

		type readyItem struct {
			v    *Violation
			item ViolationItem
		}
		ready := make([]readyItem, 0, len(vs))
		for _, v := range vs {
			it, reason := d.prepareItem(v)
			if reason != "" {
				_ = d.repo.MarkFailed(v.ID, "dilewati: "+reason+". Tidak dikirim ke ETLE.")
				d.logDelivery(v, "skipped_unknown", v.Attempts, 0, 0, reason, 0)
				s.Skipped++
				continue
			}
			ready = append(ready, readyItem{v, it})
		}

		for start := 0; start < len(ready); start += maxBatchSize {
			end := start + maxBatchSize
			if end > len(ready) {
				end = len(ready)
			}
			chunk := ready[start:end]
			payload := make([]ViolationItem, len(chunk))
			for i, r := range chunk {
				payload[i] = r.item
			}

			begin := time.Now()
			resp, err := d.deliver(c, payload)
			dur := time.Since(begin)
			s.Batches++
			if err == nil {
				for _, r := range chunk {
					_ = d.repo.MarkSent(r.v.ID, resp.Status)
					d.logDelivery(r.v, "sent", r.v.Attempts+1, 200, resp.Status, "", dur)
					s.Sent++
				}
				continue
			}

			// Batch gagal -> isolasi per item agar satu data bermasalah tidak
			// menggagalkan data lain di batch yang sama.
			for _, r := range chunk {
				b2 := time.Now()
				resp2, err2 := d.deliver(c, []ViolationItem{r.item})
				d2 := time.Since(b2)
				if err2 == nil {
					_ = d.repo.MarkSent(r.v.ID, resp2.Status)
					d.logDelivery(r.v, "sent", r.v.Attempts+1, 200, resp2.Status, "", d2)
					s.Sent++
					continue
				}
				httpStatus := 0
				if apiErr, ok := err2.(*APIError); ok {
					httpStatus = apiErr.StatusCode
				}
				_ = d.repo.FailOrRetry(r.v.ID, r.v.Attempts, d.maxRetry, err2.Error())
				d.logDelivery(r.v, "failed", r.v.Attempts+1, httpStatus, 0, err2.Error(), d2)
				if r.v.Attempts+1 >= d.maxRetry {
					s.Failed++
				} else {
					s.Retry++
				}
			}
		}
	}
}

func (d *Dispatcher) reaper() {
	ticker := time.NewTicker(3 * time.Second)
	for range ticker.C {
		_, _ = d.repo.ResetStuck()
		_, _ = d.repo.ExpireStaleViolations(d.maxDelayMinutes)
		rows, err := d.repo.PendingViolations(100, d.maxRetry, d.maxDelayMinutes)
		if err != nil {
			continue
		}
		for _, v := range rows {
			d.Submit(Job{ViolationID: v.ID, ClientID: v.ClientID})
		}
	}
}

// DrainOnce: kirim semua pending lalu keluar. Cocok untuk cronjob.
func (d *Dispatcher) DrainOnce() {
	for {
		_, _ = d.repo.ResetStuck()
		_, _ = d.repo.ExpireStaleViolations(d.maxDelayMinutes)
		rows, err := d.repo.PendingViolations(200, d.maxRetry, d.maxDelayMinutes)
		if err != nil || len(rows) == 0 {
			break
		}
		log.Printf("drain: %d pending", len(rows))
		for _, v := range rows {
			d.Submit(Job{ViolationID: v.ID, ClientID: v.ClientID})
		}
		time.Sleep(time.Second)
	}
	time.Sleep(30 * time.Second) // grace period agar worker selesai
	log.Println("drain done")
}
