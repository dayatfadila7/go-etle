package main

import (
	"testing"
	"time"
)

func TestSoftDeleteModelFields(t *testing.T) {
	now := time.Now()
	client := Client{
		ID:        1,
		Name:      "Polres Pasuruan",
		IsActive:  false,
		DeletedAt: &now,
	}

	if client.DeletedAt == nil || client.IsActive {
		t.Errorf("expected client to be soft deleted and inactive")
	}

	cam := Camera{
		ID:         1,
		ClientID:   1,
		CameraCode: "UNIX1",
		DeviceName: "CAM-BANGIL-01",
		Status:     "deleted",
		DeletedAt:  &now,
	}

	if cam.DeletedAt == nil || cam.Status != "deleted" {
		t.Errorf("expected camera to be soft deleted with deleted status")
	}
}
