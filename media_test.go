package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildMediaURL(t *testing.T) {
	cfg := &Config{MediaDir: "/tmp/opencode/pelanggaran-test", PublicURL: "http://103.199.117.48"}

	// Path lokal di dalam MediaDir -> URL publik yang bisa dilihat end-user
	got := BuildMediaURL(cfg, "/tmp/opencode/pelanggaran-test/pasuruan/0021/20260916/UNIX1_20260916043832189_EA8128KC.jpg")
	want := "http://103.199.117.48/media/pasuruan/0021/20260916/UNIX1_20260916043832189_EA8128KC.jpg"
	if got != want {
		t.Errorf("expected %s, got %s", want, got)
	}

	// Sudah URL http -> dibiarkan
	if v := BuildMediaURL(cfg, "http://exa-mple.com/a.jpg"); v != "http://exa-mple.com/a.jpg" {
		t.Errorf("expected http url untouched, got %s", v)
	}
	if v := BuildMediaURL(cfg, "https://exa-mple.com/a.jpg"); v != "https://exa-mple.com/a.jpg" {
		t.Errorf("expected https url untouched, got %s", v)
	}

	// Path di luar MediaDir -> tidak bisa dipetakan, dikembalikan apa adanya
	out := "/var/x/other.jpg"
	if v := BuildMediaURL(cfg, out); v != out {
		t.Errorf("expected outside path untouched, got %s", v)
	}

	// Kosong
	if v := BuildMediaURL(cfg, ""); v != "" {
		t.Errorf("expected empty, got %s", v)
	}
}

func TestCopyToSnapshots(t *testing.T) {
	dir := t.TempDir()
	// buat file sumber di subfolder dengan spasi (mirip "ETLE - Kab.Pasuruan")
	srcDir := filepath.Join(dir, "ETLE - Kab.Pasuruan", "0021")
	if err := os.MkdirAll(srcDir, 0755); err != nil {
		t.Fatal(err)
	}
	src := filepath.Join(srcDir, "UNIX1_20260916043832189_EA8128KC.jpg")
	if err := os.WriteFile(src, []byte("fake-image"), 0644); err != nil {
		t.Fatal(err)
	}

	cfg := &Config{MediaDir: dir, PublicURL: "http://103.199.117.48"}

	// 1. Copy ke snapshots -> URL datar tanpa spasi
	got := CopyToSnapshots(cfg, src)
	want := "http://103.199.117.48/media/snapshots/UNIX1_20260916043832189_EA8128KC.jpg"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
	snapPath := filepath.Join(dir, "snapshots", filepath.Base(src))
	if _, err := os.Stat(snapPath); err != nil {
		t.Fatalf("snapshot file tidak dibuat: %v", err)
	}

	// 2. Idempotent: panggil lagi pakai path object snapshot -> URL sama
	if got2 := CopyToSnapshots(cfg, snapPath); !strings.HasPrefix(got2, "http://103.199.117.48/media/snapshots/UNIX1_") {
		t.Fatalf("idempotent gagal: %q", got2)
	}

	// 3. URL http(s) dibiarkan apa adanya
	if v := CopyToSnapshots(cfg, "https://cdn.example/x.jpg"); v != "https://cdn.example/x.jpg" {
		t.Fatalf("https seharusnya tak diubah, got %q", v)
	}

	// 4. File tidak ada -> fallback ke BuildMediaURL (masih tergantung lokasinya)
	missing := filepath.Join(dir, "ETLE - Kab.Pasuruan", "no-such.jpg")
	if v := CopyToSnapshots(cfg, missing); v == "" {
		t.Fatalf("expected fallback non-empty")
	}

	// 5. Kosong
	if v := CopyToSnapshots(cfg, ""); v != "" {
		t.Fatalf("expected empty, got %q", v)
	}
}
