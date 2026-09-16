package main

import (
	"database/sql"
	"errors"
)

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository { return &Repository{db: db} }

func (r *Repository) CreateClient(c *Client) error {
	return r.db.QueryRow(
		`INSERT INTO clients(name, client_id, client_secret, usertoken, passtoken, auth_token, refresh_token, base_url, is_active)
		 VALUES($1,$2,$3,$4,$5,$6,$7,$8,true) RETURNING id`,
		c.Name, c.ClientID, c.ClientSecret, c.Usertoken, c.Passtoken, c.AuthToken, c.RefreshToken, c.BaseURL,
	).Scan(&c.ID)
}

func (r *Repository) GetClient(id int64) (*Client, error) {
	c := &Client{}
	err := r.db.QueryRow(
		`SELECT id, name, client_id, client_secret, usertoken, passtoken, auth_token, refresh_token, base_url, is_active
		 FROM clients WHERE id=$1 AND deleted_at IS NULL`, id,
	).Scan(&c.ID, &c.Name, &c.ClientID, &c.ClientSecret, &c.Usertoken, &c.Passtoken, &c.AuthToken, &c.RefreshToken, &c.BaseURL, &c.IsActive)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return c, nil
}

func (r *Repository) ListClients() ([]Client, error) {
	rows, err := r.db.Query(`SELECT id, name, client_id, client_secret, usertoken, passtoken, auth_token, refresh_token, base_url, is_active FROM clients WHERE deleted_at IS NULL ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Client{}
	for rows.Next() {
		var c Client
		if err := rows.Scan(&c.ID, &c.Name, &c.ClientID, &c.ClientSecret, &c.Usertoken, &c.Passtoken, &c.AuthToken, &c.RefreshToken, &c.BaseURL, &c.IsActive); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, nil
}

func (r *Repository) UpdateClient(c *Client) error {
	_, err := r.db.Exec(
		`UPDATE clients SET name=$1, client_id=$2, client_secret=$3, usertoken=$4, passtoken=$5, base_url=$6, is_active=$7, updated_at=NOW()
		 WHERE id=$8 AND deleted_at IS NULL`,
		c.Name, c.ClientID, c.ClientSecret, c.Usertoken, c.Passtoken, c.BaseURL, c.IsActive, c.ID,
	)
	return err
}

func (r *Repository) DeleteClient(id int64) error {
	_, err := r.db.Exec(`UPDATE clients SET is_active=FALSE, deleted_at=NOW(), updated_at=NOW() WHERE id=$1 AND deleted_at IS NULL`, id)
	return err
}

func (r *Repository) SetTokens(id int64, access, refresh string) error {
	_, err := r.db.Exec(`UPDATE clients SET auth_token=$1, refresh_token=$2, updated_at=NOW() WHERE id=$3`, access, refresh, id)
	return err
}

// UpsertMasterViolations mengembalikan jumlah data yang BENAR-BENAR berubah
// (insert baru atau nama/raw berubah), bukan selalu seluruh row dari master API.
func (r *Repository) UpsertMasterViolations(items []MasterViolation) (int, error) {
	n := 0
	for _, it := range items {
		var raw interface{}
		if len(it.Raw) > 0 {
			raw = it.Raw
		}
		var inserted bool
		err := r.db.QueryRow(
			`INSERT INTO master_violations(code, name, raw_data, synced_at)
			 VALUES($1,$2,$3,NOW())
			 ON CONFLICT(code) DO UPDATE
			   SET name=EXCLUDED.name,
			       raw_data=EXCLUDED.raw_data,
			       synced_at=EXCLUDED.synced_at
			   WHERE master_violations.name IS DISTINCT FROM EXCLUDED.name
			      OR master_violations.raw_data IS DISTINCT FROM EXCLUDED.raw_data
			 RETURNING (xmax = 0)`,
			it.Code, it.Name, raw,
		).Scan(&inserted)
		if errors.Is(err, sql.ErrNoRows) {
			// nilai sama dengan DB -> tidak ada perubahan, jangan dihitung
			continue
		}
		if err != nil {
			return n, err
		}
		n++
	}
	return n, nil
}

func (r *Repository) ListMaster() ([]MasterViolation, error) {
	rows, err := r.db.Query(`SELECT code, name FROM master_violations ORDER BY code`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []MasterViolation{}
	for rows.Next() {
		var m MasterViolation
		if err := rows.Scan(&m.Code, &m.Name); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, nil
}

func (r *Repository) CreateCamera(c *Camera) error {
	return r.db.QueryRow(
		`INSERT INTO cameras(client_id, camera_code, device_name, location_name, address, latitude, longitude, status)
		 VALUES($1,$2,$3,$4,$5,$6,$7,'active') RETURNING id`,
		c.ClientID, c.CameraCode, c.DeviceName, c.LocationName, c.Address, c.Latitude, c.Longitude,
	).Scan(&c.ID)
}

func (r *Repository) GetCamera(id int64) (*Camera, error) {
	var c Camera
	err := r.db.QueryRow(`
		SELECT c.id, c.client_id, COALESCE(cl.name, ''), COALESCE(c.camera_code, ''), c.device_name, c.location_name, c.address, c.latitude, c.longitude, c.status 
		FROM cameras c
		LEFT JOIN clients cl ON c.client_id = cl.id
		WHERE c.id=$1 AND c.deleted_at IS NULL`, id).
		Scan(&c.ID, &c.ClientID, &c.ClientName, &c.CameraCode, &c.DeviceName, &c.LocationName, &c.Address, &c.Latitude, &c.Longitude, &c.Status)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (r *Repository) ListCameras(clientID int64) ([]Camera, error) {
	rows, err := r.db.Query(`
		SELECT c.id, c.client_id, COALESCE(cl.name, ''), COALESCE(c.camera_code, ''), c.device_name, c.location_name, c.address, c.latitude, c.longitude, c.status 
		FROM cameras c
		LEFT JOIN clients cl ON c.client_id = cl.id
		WHERE c.deleted_at IS NULL AND ($1=0 OR c.client_id=$1) 
		ORDER BY c.id`, clientID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Camera{}
	for rows.Next() {
		var c Camera
		if err := rows.Scan(&c.ID, &c.ClientID, &c.ClientName, &c.CameraCode, &c.DeviceName, &c.LocationName, &c.Address, &c.Latitude, &c.Longitude, &c.Status); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, nil
}

func (r *Repository) UpdateCamera(c *Camera) error {
	status := c.Status
	if status == "" {
		status = "active"
	}
	_, err := r.db.Exec(
		`UPDATE cameras SET client_id=$1, camera_code=$2, device_name=$3, location_name=$4, address=$5, latitude=$6, longitude=$7, status=$8, updated_at=NOW()
		 WHERE id=$9 AND deleted_at IS NULL`,
		c.ClientID, c.CameraCode, c.DeviceName, c.LocationName, c.Address, c.Latitude, c.Longitude, status, c.ID,
	)
	return err
}

func (r *Repository) DeleteCamera(id int64) error {
	_, err := r.db.Exec(`UPDATE cameras SET status='deleted', deleted_at=NOW(), updated_at=NOW() WHERE id=$1 AND deleted_at IS NULL`, id)
	return err
}

func (r *Repository) FindCameraByCode(code string) (*Camera, error) {
	if code == "" {
		return nil, nil
	}
	var c Camera
	err := r.db.QueryRow(`
		SELECT c.id, c.client_id, COALESCE(cl.name, ''), COALESCE(c.camera_code, ''), c.device_name, c.location_name, c.address, c.latitude, c.longitude, c.status 
		FROM cameras c
		LEFT JOIN clients cl ON c.client_id = cl.id
		WHERE c.camera_code=$1 AND c.deleted_at IS NULL LIMIT 1`, code).
		Scan(&c.ID, &c.ClientID, &c.ClientName, &c.CameraCode, &c.DeviceName, &c.LocationName, &c.Address, &c.Latitude, &c.Longitude, &c.Status)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (r *Repository) EnqueueViolation(v *Violation) (int64, error) {
	status := v.Status
	if status == "" {
		status = "pending"
	}
	err := r.db.QueryRow(
		`INSERT INTO violations(client_id, camera_id, device_name, plate, plate_color, plate_image_url,
			vehicle_type, vehicle_color, vehicle_image_url, video_url, violation_code, violation_name,
			location_name, capture_time, status, error_message)
		 VALUES($1,NULLIF($2,0),$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16) RETURNING id`,
		v.ClientID, v.CameraID, v.DeviceName, v.Plate, v.PlateColor, v.PlateImageURL,
		v.VehicleType, v.VehicleColor, v.VehicleImageURL, v.VideoURL, v.ViolationCode, v.ViolationName,
		v.LocationName, v.CaptureTime, status, v.ErrorMessage,
	).Scan(&v.ID)
	return v.ID, err
}

func (r *Repository) GetViolation(id int64) (*Violation, error) {
	v := &Violation{}
	err := r.db.QueryRow(
		`SELECT id, client_id, COALESCE(camera_id,0), device_name, plate, COALESCE(plate_color,''),
			COALESCE(plate_image_url,''), COALESCE(vehicle_type,''), COALESCE(vehicle_color,''),
			COALESCE(vehicle_image_url,''), COALESCE(video_url,''), violation_code, violation_name,
			COALESCE(location_name,''), COALESCE(capture_time,0), status, COALESCE(response_status,0),
			attempts, COALESCE(error_message, ''), created_at
		 FROM violations WHERE id=$1`, id,
	).Scan(&v.ID, &v.ClientID, &v.CameraID, &v.DeviceName, &v.Plate, &v.PlateColor, &v.PlateImageURL,
		&v.VehicleType, &v.VehicleColor, &v.VehicleImageURL, &v.VideoURL, &v.ViolationCode, &v.ViolationName,
		&v.LocationName, &v.CaptureTime, &v.Status, &v.ResponseStatus, &v.Attempts, &v.ErrorMessage, &v.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return v, nil
}

// GetViolationDetail mengambil satu pelanggaran lengkap beserta nama client & kamera.
func (r *Repository) GetViolationDetail(id int64) (*ViolationDetail, error) {
	vd := &ViolationDetail{}
	var sentAt sql.NullTime
	err := r.db.QueryRow(
		`SELECT v.id, v.client_id, COALESCE(v.camera_id,0), v.device_name, v.plate, COALESCE(v.plate_color,''),
		        COALESCE(v.plate_image_url,''), COALESCE(v.vehicle_type,''), COALESCE(v.vehicle_color,''),
		        COALESCE(v.vehicle_image_url,''), COALESCE(v.video_url,''),
		        v.violation_code, v.violation_name, COALESCE(v.location_name,''), COALESCE(v.capture_time,0), v.status,
		        COALESCE(v.response_status,0), v.attempts, COALESCE(v.error_message,''), v.created_at, v.sent_at,
		        COALESCE(c.name,''), COALESCE(c.client_id,''),
		        COALESCE(cam.device_name,''), COALESCE(cam.camera_code,'')
		   FROM violations v
		   LEFT JOIN clients c  ON c.id  = v.client_id
		   LEFT JOIN cameras cam ON cam.id = v.camera_id
		  WHERE v.id=$1`, id,
	).Scan(&vd.ID, &vd.ClientID, &vd.CameraID, &vd.DeviceName, &vd.Plate, &vd.PlateColor,
		&vd.PlateImageURL, &vd.VehicleType, &vd.VehicleColor, &vd.VehicleImageURL, &vd.VideoURL,
		&vd.ViolationCode, &vd.ViolationName, &vd.LocationName, &vd.CaptureTime, &vd.Status,
		&vd.ResponseStatus, &vd.Attempts, &vd.ErrorMessage, &vd.CreatedAt, &sentAt,
		&vd.ClientName, &vd.ClientCode, &vd.CameraName, &vd.CameraCode)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if sentAt.Valid {
		st := sentAt.Time
		vd.SentAt = &st
	}
	return vd, nil
}

// ClaimViolation ambil satu row secara atomik supaya banyak worker/proses tak kirim dua kali.
func (r *Repository) ClaimViolation(id int64) (bool, error) {
	res, err := r.db.Exec(`UPDATE violations SET status='processing', updated_at=NOW() WHERE id=$1 AND status='pending'`, id)
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()
	return n > 0, nil
}

func (r *Repository) ResetStuck() (int64, error) {
	res, err := r.db.Exec(`UPDATE violations SET status='pending' WHERE status='processing' AND updated_at < NOW() - INTERVAL '5 minutes'`)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

// ExpireStaleViolations mendiskualifikasi data yang delay > maxMinutes atau beda hari kalender.
func (r *Repository) ExpireStaleViolations(maxMinutes int) (int64, error) {
	if maxMinutes <= 0 {
		maxMinutes = 3
	}
	res, err := r.db.Exec(
		`UPDATE violations
		 SET status='failed',
		     error_message='expired: delay melebihi batas ' || $1 || ' menit atau beda hari',
		     updated_at=NOW()
		 WHERE status IN ('pending', 'processing')
		   AND (
		       created_at < CURRENT_DATE
		       OR created_at < NOW() - ($1 * INTERVAL '1 minute')
		   )`,
		maxMinutes,
	)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

// PendingViolations hanya mengambil data yang masih fresh (hari ini & delay <= maxDelayMinutes).
func (r *Repository) PendingViolations(limit, maxRetry, maxDelayMinutes int) ([]Violation, error) {
	if maxDelayMinutes <= 0 {
		maxDelayMinutes = 3
	}
	rows, err := r.db.Query(
		`SELECT id, client_id, COALESCE(camera_id,0), device_name, plate, COALESCE(plate_color,''),
			COALESCE(plate_image_url,''), COALESCE(vehicle_type,''), COALESCE(vehicle_color,''),
			COALESCE(vehicle_image_url,''), COALESCE(video_url,''), violation_code, violation_name,
			COALESCE(location_name,''), COALESCE(capture_time,0), status, COALESCE(response_status,0),
			attempts, COALESCE(error_message, ''), created_at
		 FROM violations
		 WHERE status='pending'
		   AND attempts < $1
		   AND created_at >= CURRENT_DATE
		   AND created_at >= NOW() - ($2 * INTERVAL '1 minute')
		 ORDER BY id LIMIT $3`,
		maxRetry, maxDelayMinutes, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Violation{}
	for rows.Next() {
		var v Violation
		if err := rows.Scan(&v.ID, &v.ClientID, &v.CameraID, &v.DeviceName, &v.Plate, &v.PlateColor, &v.PlateImageURL,
			&v.VehicleType, &v.VehicleColor, &v.VehicleImageURL, &v.VideoURL, &v.ViolationCode, &v.ViolationName,
			&v.LocationName, &v.CaptureTime, &v.Status, &v.ResponseStatus, &v.Attempts, &v.ErrorMessage, &v.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, nil
}

func (r *Repository) MarkFailed(id int64, errMsg string) error {
	_, err := r.db.Exec(`UPDATE violations SET status='failed', error_message=$1, updated_at=NOW() WHERE id=$2`, errMsg, id)
	return err
}

func (r *Repository) MarkSent(id int64, status int) error {
	_, err := r.db.Exec(
		`UPDATE violations SET status='sent', response_status=$1, attempts=attempts+1, sent_at=NOW(), updated_at=NOW() WHERE id=$2`,
		status, id)
	return err
}

func (r *Repository) FailOrRetry(id int64, attemptsSoFar, maxRetry int, errMsg string) error {
	if attemptsSoFar+1 >= maxRetry {
		_, err := r.db.Exec(`UPDATE violations SET status='failed', attempts=attempts+1, error_message=$1, updated_at=NOW() WHERE id=$2`, errMsg, id)
		return err
	}
	_, err := r.db.Exec(`UPDATE violations SET status='pending', attempts=attempts+1, error_message=$1, updated_at=NOW() WHERE id=$2`, errMsg, id)
	return err
}

// ListViolations mengembalikan halaman data violations beserta total tanpa filter status.
func (r *Repository) ListViolations(status string, page, perPage int) ([]Violation, int64, error) {
	if perPage <= 0 {
		perPage = 20
	}
	if perPage > 500 {
		perPage = 500
	}
	if page < 1 {
		page = 1
	}
	var total int64
	if err := r.db.QueryRow(`SELECT COUNT(*) FROM violations WHERE ($1='' OR status=$1)`, status).Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := r.db.Query(
		`SELECT id, client_id, device_name, plate, violation_code, violation_name, status,
		        COALESCE(response_status,0), attempts, COALESCE(capture_time,0), sent_at, created_at
		 FROM violations WHERE ($1='' OR status=$1) ORDER BY id DESC LIMIT $2 OFFSET $3`,
		status, perPage, (page-1)*perPage,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	out := []Violation{}
	for rows.Next() {
		var v Violation
		var sentAt sql.NullTime
		if err := rows.Scan(&v.ID, &v.ClientID, &v.DeviceName, &v.Plate, &v.ViolationCode, &v.ViolationName,
			&v.Status, &v.ResponseStatus, &v.Attempts, &v.CaptureTime, &sentAt, &v.CreatedAt); err != nil {
			return nil, 0, err
		}
		if sentAt.Valid {
			st := sentAt.Time
			v.SentAt = &st
		}
		out = append(out, v)
	}
	return out, total, nil
}

// GetCameraByID mengambil data kamera aktif berdasarkan ID.
func (r *Repository) GetCameraByID(id int64) (*Camera, error) {
	c := &Camera{}
	err := r.db.QueryRow(
		`SELECT id, client_id, COALESCE(camera_code,''), device_name,
		        COALESCE(location_name,''), COALESCE(address,''), COALESCE(latitude,0), COALESCE(longitude,0), status
		   FROM cameras WHERE id=$1 AND deleted_at IS NULL`, id,
	).Scan(&c.ID, &c.ClientID, &c.CameraCode, &c.DeviceName, &c.LocationName, &c.Address, &c.Latitude, &c.Longitude, &c.Status)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return c, nil
}

func (r *Repository) CreateUser(u *User) error {
	return r.db.QueryRow(
		`INSERT INTO users(username, email, password_hash, name, role, is_active)
		 VALUES($1,$2,$3,$4,$5,true) RETURNING id`,
		u.Username, u.Email, u.PasswordHash, u.Name, u.Role,
	).Scan(&u.ID)
}

func (r *Repository) GetUserByUsername(username string) (*User, error) {
	u := &User{}
	err := r.db.QueryRow(`SELECT id, username, email, password_hash, name, role, is_active FROM users WHERE username=$1`, username).
		Scan(&u.ID, &u.Username, &u.Email, &u.PasswordHash, &u.Name, &u.Role, &u.IsActive)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return u, nil
}

func (r *Repository) ListUsers() ([]User, error) {
	rows, err := r.db.Query(`SELECT id, username, email, name, role, is_active FROM users ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []User{}
	for rows.Next() {
		var u User
		if err := rows.Scan(&u.ID, &u.Username, &u.Email, &u.Name, &u.Role, &u.IsActive); err != nil {
			return nil, err
		}
		out = append(out, u)
	}
	return out, nil
}

func (r *Repository) CreateSyncLog(sl *SyncLog) error {
	return r.db.QueryRow(
		`INSERT INTO sync_logs(client_id, action, status, records_synced, message)
		 VALUES($1, $2, $3, $4, $5) RETURNING id, created_at`,
		sl.ClientID, sl.Action, sl.Status, sl.RecordsSynced, sl.Message,
	).Scan(&sl.ID, &sl.CreatedAt)
}

func (r *Repository) ListSyncLogs(limit int) ([]SyncLog, error) {
	if limit <= 0 || limit > 500 {
		limit = 50
	}
	rows, err := r.db.Query(
		`SELECT s.id, s.client_id, COALESCE(c.name, 'Public / All'), s.action, s.status, s.records_synced, COALESCE(s.message, ''), s.created_at
		 FROM sync_logs s
		 LEFT JOIN clients c ON s.client_id = c.id
		 ORDER BY s.id DESC LIMIT $1`, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []SyncLog{}
	for rows.Next() {
		var sl SyncLog
		if err := rows.Scan(&sl.ID, &sl.ClientID, &sl.ClientName, &sl.Action, &sl.Status, &sl.RecordsSynced, &sl.Message, &sl.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, sl)
	}
	return out, nil
}

func (r *Repository) PurgeOldViolations(days int) (int64, error) {
	if days <= 0 {
		days = 2
	}
	res, err := r.db.Exec(`DELETE FROM violations WHERE created_at < NOW() - ($1 * INTERVAL '1 day')`, days)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

func (r *Repository) GetDashboardStats() (*DashboardStats, error) {
	s := &DashboardStats{RetentionDays: 3}
	_ = r.db.QueryRow(`SELECT COUNT(*) FROM violations`).Scan(&s.TotalViolations)
	_ = r.db.QueryRow(`SELECT COUNT(*) FROM violations WHERE status='pending'`).Scan(&s.Pending)
	_ = r.db.QueryRow(`SELECT COUNT(*) FROM violations WHERE status='processing'`).Scan(&s.Processing)
	_ = r.db.QueryRow(`SELECT COUNT(*) FROM violations WHERE status='sent'`).Scan(&s.Sent)
	_ = r.db.QueryRow(`SELECT COUNT(*) FROM violations WHERE status='failed'`).Scan(&s.Failed)
	_ = r.db.QueryRow(`SELECT COUNT(*) FROM clients WHERE is_active=true AND deleted_at IS NULL`).Scan(&s.TotalClients)
	_ = r.db.QueryRow(`SELECT COUNT(*) FROM cameras WHERE status='active' AND deleted_at IS NULL`).Scan(&s.TotalCameras)
	_ = r.db.QueryRow(`SELECT COUNT(*) FROM master_violations`).Scan(&s.MasterCount)
	return s, nil
}
