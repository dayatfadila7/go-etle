package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestLogDelivery(t *testing.T) {
	dir := t.TempDir()
	old := deliveryLogFile
	deliveryLogFile = filepath.Join(dir, "delivery.log")
	defer func() { deliveryLogFile = old }()

	d := &Dispatcher{}
	v := &Violation{ID: 123, Plate: "B1234CD", DeviceName: "CAM-X", ViolationCode: "151002", CaptureTime: time.Now().UnixMilli()}
	d.logDelivery(v, "sent", 1, 200, 1112, "", 37*time.Millisecond)

	b, err := os.ReadFile(deliveryLogFile)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(string(b)), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected header + 1 row, got %d lines", len(lines))
	}
	if !strings.HasPrefix(lines[0], "time\tresult\tid") {
		t.Fatalf("bad header: %q", lines[0])
	}
	if !strings.Contains(lines[1], "\tsent\t123\tB1234CD\tCAM-X\t1\t151002\t") ||
		!strings.Contains(lines[1], "\t200\t1112\t\t37") {
		t.Fatalf("bad row: %q", lines[1])
	}

	d.logDelivery(v, "failed", 2, 500, 0, "etle http 500", 12*time.Millisecond)
	b, _ = os.ReadFile(deliveryLogFile)
	lines = strings.Split(strings.TrimSpace(string(b)), "\n")
	if len(lines) != 3 {
		t.Fatalf("expected 3 lines after 2nd write, got %d", len(lines))
	}
	if !strings.HasPrefix(lines[2], "2026") || !strings.Contains(lines[2], "\tfailed\t") ||
		!strings.Contains(lines[2], "\t2\t") || !strings.Contains(lines[2], "\t500\t0\t") ||
		!strings.Contains(lines[2], "etle http 500") {
		t.Fatalf("bad failed row: %q", lines[2])
	}
}