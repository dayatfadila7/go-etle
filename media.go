package main

import (
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