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
		_ = d.repo.MarkFailed(j.ViolationID, "expired: delay melebihi batas 3 menit atau beda hari kalender")
		d.logDelivery(v, "expired", v.Attempts, 0, 0, "delay melebihi batas atau beda hari", 0)
		return
	}

	c, err := d.repo.GetClient(j.ClientID)
	if err != nil || c == nil {
		_ = d.repo.MarkFailed(j.ViolationID, "client not found")
		d.logDelivery(v, "failed", v.Attempts, 0, 0, "client not found", 0)
		return
	}

	if c.AuthToken == "" {
		if lr, e := d.etle.Login(c); e == nil {
			c.AuthToken = lr.AccessToken
			_ = d.repo.SetTokens(c.ID, lr.AccessToken, lr.RefreshToken)
		}
	}

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
		_ = d.repo.MarkFailed(j.ViolationID, "dilewati: plat nomor tidak terbaca (unknown). Tidak dikirim ke ETLE.")
		d.logDelivery(v, "skipped_unknown", v.Attempts, 0, 0, "plat nomor tidak terbaca (unknown)", 0)
		return
	}
	start := time.Now()
	resp, err := d.etle.SendViolation(c, ViolationPayload{Datas: []ViolationItem{item}})
	if err != nil {
		// token expired (401) -> login ulang lalu retry sekali
		if apiErr, ok := err.(*APIError); ok && apiErr.StatusCode == 401 {
			if lr, e := d.etle.Login(c); e == nil {
				_ = d.repo.SetTokens(c.ID, lr.AccessToken, lr.RefreshToken)
				c.AuthToken = lr.AccessToken
				resp, err = d.etle.SendViolation(c, ViolationPayload{Datas: []ViolationItem{item}})
			}
		}
	}
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
