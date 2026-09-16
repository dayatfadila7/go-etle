package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestParseCaptureTime(t *testing.T) {
	// Zero
	if !parseCaptureTime(0).IsZero() {
		t.Errorf("expected zero time for 0")
	}

	// Seconds (10 digits)
	sec := int64(1700000000)
	tmSec := parseCaptureTime(sec)
	if tmSec.Unix() != sec {
		t.Errorf("expected %d, got %d", sec, tmSec.Unix())
	}

	// Milliseconds (13 digits)
	msec := int64(1700000000123)
	tmMsec := parseCaptureTime(msec)
	if tmMsec.UnixMilli() != msec {
		t.Errorf("expected %d, got %d", msec, tmMsec.UnixMilli())
	}
}

func TestRealtimeExpirationLogic(t *testing.T) {
	now := time.Now()
	maxDelay := 3 * time.Minute

	// Fresh: 30 seconds ago
	freshTime := now.Add(-30 * time.Second)
	isCrossDay := freshTime.Year() != now.Year() || freshTime.YearDay() != now.YearDay()
	isStale := isCrossDay || now.Sub(freshTime) > maxDelay
	if isStale {
		t.Errorf("expected freshTime (30s ago) to not be stale")
	}

	// Expired: 4 minutes ago
	delayedTime := now.Add(-4 * time.Minute)
	isDelayedStale := delayedTime.Year() != now.Year() || delayedTime.YearDay() != now.YearDay() || now.Sub(delayedTime) > maxDelay
	if !isDelayedStale {
		t.Errorf("expected delayedTime (4m ago) to be stale")
	}

	// Expired: yesterday (same time of day)
	yesterday := now.AddDate(0, 0, -1)
	isYesterdayCrossDay := yesterday.Year() != now.Year() || yesterday.YearDay() != now.YearDay()
	if !isYesterdayCrossDay {
		t.Errorf("expected yesterday to be cross day")
	}
}

func TestConfigDefaultMaxDelay(t *testing.T) {
	cfg := LoadConfig()
	if cfg.MaxDelayMinutes != 3 {
		t.Errorf("expected default MaxDelayMinutes to be 3, got %d", cfg.MaxDelayMinutes)
	}
	if cfg.RetentionDays != 2 {
		t.Errorf("expected default RetentionDays to be 2, got %d", cfg.RetentionDays)
	}
	if cfg.MediaDir != "./storage" {
		t.Errorf("expected default MediaDir to be ./storage, got %s", cfg.MediaDir)
	}
}

func TestPurgeOldFiles(t *testing.T) {
	tempDir := t.TempDir()

	oldTime := time.Now().Add(-72 * time.Hour) // 3 hari lalu
	freshTime := time.Now().Add(-1 * time.Hour) // 1 jam lalu

	// Buat file old (xml, jpg)
	oldXML := filepath.Join(tempDir, "sample_old.xml")
	oldJPG := filepath.Join(tempDir, "sample_old.jpg")
	freshXML := filepath.Join(tempDir, "sample_fresh.xml")
	freshJPG := filepath.Join(tempDir, "sample_fresh.jpg")
	otherFile := filepath.Join(tempDir, "sample_old.txt")

	_ = os.WriteFile(oldXML, []byte("<violation>old</violation>"), 0644)
	_ = os.Chtimes(oldXML, oldTime, oldTime)

	_ = os.WriteFile(oldJPG, []byte("fake-jpeg-data-old"), 0644)
	_ = os.Chtimes(oldJPG, oldTime, oldTime)

	_ = os.WriteFile(freshXML, []byte("<violation>fresh</violation>"), 0644)
	_ = os.Chtimes(freshXML, freshTime, freshTime)

	_ = os.WriteFile(freshJPG, []byte("fake-jpeg-data-fresh"), 0644)
	_ = os.Chtimes(freshJPG, freshTime, freshTime)

	_ = os.WriteFile(otherFile, []byte("plain text"), 0644)
	_ = os.Chtimes(otherFile, oldTime, oldTime)

	// Jalankan PurgeOldFiles dengan batas 2 hari
	exts := []string{"xml", "jpg", "jpeg", "png"}
	deleted, _, err := PurgeOldFiles(tempDir, 2, exts)
	if err != nil {
		t.Fatalf("PurgeOldFiles failed: %v", err)
	}

	if deleted != 2 {
		t.Errorf("expected 2 files deleted, got %d", deleted)
	}

	// File fresh harus tetap ada
	if _, err := os.Stat(freshXML); os.IsNotExist(err) {
		t.Errorf("expected freshXML to still exist")
	}
	if _, err := os.Stat(freshJPG); os.IsNotExist(err) {
		t.Errorf("expected freshJPG to still exist")
	}

	// File other extension (.txt) harus tetap ada
	if _, err := os.Stat(otherFile); os.IsNotExist(err) {
		t.Errorf("expected otherFile (.txt) to still exist")
	}

	// File old xml & jpg harus sudah terhapus
	if _, err := os.Stat(oldXML); !os.IsNotExist(err) {
		t.Errorf("expected oldXML to be deleted")
	}
	if _, err := os.Stat(oldJPG); !os.IsNotExist(err) {
		t.Errorf("expected oldJPG to be deleted")
	}
}
