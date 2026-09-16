package main

import "testing"

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