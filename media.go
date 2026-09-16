package main

import (
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

func fileExists(p string) bool {
	if p == "" {
		return false
	}
	fi, err := os.Stat(p)
	return err == nil && !fi.IsDir()
}

// mediaRel mengubah path absolut file media menjadi path relatif ke root media.
func mediaRel(root, path string) (string, bool) {
	root, err := filepath.Abs(root)
	if err != nil {
		return "", false
	}
	path, err = filepath.Abs(path)
	if err != nil {
		return "", false
	}
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return "", false
	}
	if rel == "." || filepath.IsAbs(rel) || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", false
	}
	return rel, true
}

// BuildMediaURL mengubah path lokal file media menjadi URL publik agar bisa
// dilihat end-user / diunduh server ETLE. Path yang sudah berupa http(s)
// dikembalikan apa adanya.
func BuildMediaURL(c *Config, path string) string {
	p := strings.TrimSpace(path)
	if p == "" {
		return p
	}
	lp := strings.ToLower(p)
	if strings.HasPrefix(lp, "http://") || strings.HasPrefix(lp, "https://") {
		return p
	}
	root := c.MediaDir
	if root == "" {
		return p
	}
	rel, ok := mediaRel(root, p)
	if !ok {
		return p
	}
	segs := make([]string, 0, 8)
	for _, s := range strings.Split(rel, string(filepath.Separator)) {
		segs = append(segs, url.PathEscape(s))
	}
	pub := strings.TrimRight(c.PublicURL, "/")
	if pub == "" {
		pub = "http://localhost:8080"
	}
	return pub + "/media/" + strings.Join(segs, "/")
}

// CopyToSnapshots menyalin file media lokal ke satu direktori datar
// "<MediaDir>/snapshots/" (tanpa subfolder/spasi) lalu mengembalikan URL publik
// "/media/snapshots/<nama-file>". URL http(s) dibiarkan apa adanya. Kalau
// sumber sudah berada di dalam snapshots, cukup kembali URL-nya (idempotent).
// Gagal menyalin -> fallback ke BuildMediaURL.
func CopyToSnapshots(c *Config, srcPath string) string {
	p := strings.TrimSpace(srcPath)
	if p == "" {
		return ""
	}
	lp := strings.ToLower(p)
	if strings.HasPrefix(lp, "http://") || strings.HasPrefix(lp, "https://") {
		return p
	}

	root := c.MediaDir
	if root == "" {
		return p
	}
	snapDir := filepath.Join(root, "snapshots")

	// Sudah di snapshots -> jangan copy ulang, cukup buat URL publik.
	if rel, ok := mediaRel(snapDir, p); ok && rel != "." {
		pub := strings.TrimRight(c.PublicURL, "/")
		if pub == "" {
			pub = "http://localhost:8080"
		}
		return pub + "/media/snapshots/" + strings.Join(segEscape(strings.Split(rel, string(filepath.Separator))), "/")
	}

	if !fileExists(p) {
		return BuildMediaURL(c, p)
	}

	if err := os.MkdirAll(snapDir, 0755); err != nil {
		return BuildMediaURL(c, p)
	}
	dst := filepath.Join(snapDir, filepath.Base(p))
	if sameAbs(dst, p) {
		return BuildMediaURL(c, p)
	}

	if fn := copyFile(p, dst); fn != "" {
		pub := strings.TrimRight(c.PublicURL, "/")
		if pub == "" {
			pub = "http://localhost:8080"
		}
		return pub + "/media/snapshots/" + url.PathEscape(filepath.Base(dst))
	}
	return BuildMediaURL(c, p)
}

func segEscape(segs []string) []string {
	out := make([]string, len(segs))
	for i, s := range segs {
		out[i] = url.PathEscape(s)
	}
	return out
}

func sameAbs(a, b string) bool {
	aa, err1 := filepath.Abs(a)
	bb, err2 := filepath.Abs(b)
	return err1 == nil && err2 == nil && aa == bb
}

func copyFile(src, dst string) string {
	in, err := os.Open(src)
	if err != nil {
		return ""
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return ""
	}
	defer out.Close()
	if _, err := io.Copy(out, in); err != nil {
		return ""
	}
	return dst
}
