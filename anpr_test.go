package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseANPRFileFallsBackToSnapshotImage(t *testing.T) {
	tempDir := t.TempDir()
	// Snapshot file kamera (nama sama dengan XML, ekstensi .jpg)
	snapshot := filepath.Join(tempDir, "UNIX1_20260916043832189_EA8128KC.jpg")
	if err := os.WriteFile(snapshot, []byte("fake-jpeg"), 0644); err != nil {
		t.Fatalf("write snapshot: %v", err)
	}
	xmlPath := filepath.Join(tempDir, "UNIX1_20260916043832189_EA8128KC.xml")
	xmlContent := `<FTP>
<dateTime>2026-09-16T04:38:32.189+07:00</dateTime>
<licensePlate>EA8128KC</licensePlate>
<illegalInfo>
<illegalCode>1240</illegalCode>
<illegalName>No_Seatbelt_Fastened</illegalName>
</illegalInfo>
<vehicleType>truck</vehicleType>
<pictureInfoList>
<pictureInfo>
<fileName>licensePlatePicture.jpg</fileName>
<type>licensePlatePicture</type>
</pictureInfo>
<pictureInfo>
<fileName>detectionPicture.jpg</fileName>
<type>detectionPicture</type>
</pictureInfo>
</pictureInfoList>
</FTP>`
	if err := os.WriteFile(xmlPath, []byte(xmlContent), 0644); err != nil {
		t.Fatalf("write xml: %v", err)
	}

	item, err := ParseANPRFile(xmlPath)
	if err != nil {
		t.Fatalf("ParseANPRFile: %v", err)
	}
	// licensePlatePicture.jpg / detectionPicture.jpg tidak ada di disk,
	// sehingga harus fallback ke snapshot .jpg milik XML itu sendiri.
	if item.PlateImageURL != snapshot {
		t.Errorf("expected PlateImageURL fallback to %s, got %s", snapshot, item.PlateImageURL)
	}
	if item.VehicleImageURL != snapshot {
		t.Errorf("expected VehicleImageURL fallback to %s, got %s", snapshot, item.VehicleImageURL)
	}
}

func TestParseANPRFilename(t *testing.T) {
	// 5-part format
	fn5 := "UNIX1_pasuruan_Jalan A.Yani Bangil Kab.Pasuruan_20240716111021710_W5003QC.xml"
	meta5 := ParseANPRFilename(fn5)
	if meta5.CameraCode != "UNIX1" {
		t.Errorf("expected UNIX1 camera code, got %s", meta5.CameraCode)
	}
	if meta5.DeviceName != "UNIX1" {
		t.Errorf("expected UNIX1, got %s", meta5.DeviceName)
	}
	if meta5.LocationName != "pasuruan" {
		t.Errorf("expected pasuruan, got %s", meta5.LocationName)
	}
	if meta5.Address != "Jalan A.Yani Bangil Kab.Pasuruan" {
		t.Errorf("expected Jalan A.Yani Bangil Kab.Pasuruan, got %s", meta5.Address)
	}
	if meta5.Plate != "W5003QC" {
		t.Errorf("expected W5003QC, got %s", meta5.Plate)
	}
	if meta5.CaptureTime.Year() != 2024 || meta5.CaptureTime.Month() != 7 || meta5.CaptureTime.Day() != 16 {
		t.Errorf("expected 2024-07-16, got %v", meta5.CaptureTime)
	}
	if meta5.CaptureTime.Hour() != 11 || meta5.CaptureTime.Minute() != 10 || meta5.CaptureTime.Second() != 21 {
		t.Errorf("expected 11:10:21, got %v", meta5.CaptureTime)
	}

	// 4-part format (without explicit city name)
	fn4 := "UNIX1_Jalan A.Yani Bangil Kab.Pasuruan_20240716111021710_W5003QC.xml"
	meta4 := ParseANPRFilename(fn4)
	if meta4.CameraCode != "UNIX1" {
		t.Errorf("expected UNIX1 camera code, got %s", meta4.CameraCode)
	}
	if meta4.DeviceName != "UNIX1" {
		t.Errorf("expected UNIX1, got %s", meta4.DeviceName)
	}
	if meta4.Address != "Jalan A.Yani Bangil Kab.Pasuruan" {
		t.Errorf("expected Jalan A.Yani Bangil Kab.Pasuruan, got %s", meta4.Address)
	}
	if meta4.Plate != "W5003QC" {
		t.Errorf("expected W5003QC, got %s", meta4.Plate)
	}
}

func TestParseANPRFile(t *testing.T) {
	tempDir := t.TempDir()
	xmlContent := `<FTP>
<dateTime>2024-07-16T11:10:21.710+00:00</dateTime>
<licensePlate>W5003QC</licensePlate>
<line>2</line>
<confidenceLevel>0</confidenceLevel>
<plateType>unknown</plateType>
<plateColor>unknown</plateColor>
<licenseBright>46</licenseBright>
<pilotsafebelt>unknown</pilotsafebelt>
<vicepilotsafebelt>unknown</vicepilotsafebelt>
<speedLimit>70</speedLimit>
<illegalInfo>
<illegalCode>151002</illegalCode>
<illegalName>Non-Helmet</illegalName>
<illegalDescription></illegalDescription>
</illegalInfo>
<vehicleType>twoWheelVehicle</vehicleType>
<vehicleInfo>
<index>8682</index>
<vehicleType>4</vehicleType>
<colorDepth>0</colorDepth>
<color>unknown</color>
<speed>0</speed>
</vehicleInfo>
<pictureInfoList>
<pictureInfo>
<fileName>licensePlatePicture.jpg</fileName>
<type>licensePlatePicture</type>
<dataType>0</dataType>
<absTime>20240716111021709</absTime>
</pictureInfo>
<pictureInfo>
<fileName>detectionPicture.jpg</fileName>
<type>detectionPicture</type>
<dataType>0</dataType>
</pictureInfo>
</pictureInfoList>
<UUID>fed0a57a-1dd1-11b2-a09a-a08d9b29eb54</UUID>
<picNum>2</picNum>
<country>IDN</country>
</FTP>`

	filePath := filepath.Join(tempDir, "UNIX1_pasuruan_Jalan A.Yani Bangil Kab.Pasuruan_20240716111021710_W5003QC.xml")
	err := os.WriteFile(filePath, []byte(xmlContent), 0644)
	if err != nil {
		t.Fatalf("write xml failed: %v", err)
	}

	item, err := ParseANPRFile(filePath)
	if err != nil {
		t.Fatalf("ParseANPRFile failed: %v", err)
	}

	if item.DeviceName != "UNIX1" {
		t.Errorf("expected device UNIX1, got %s", item.DeviceName)
	}
	if item.Plate != "W5003QC" {
		t.Errorf("expected plate W5003QC, got %s", item.Plate)
	}
	if item.ViolationCode != "151002" {
		t.Errorf("expected code 151002, got %s", item.ViolationCode)
	}
	if item.ViolationName != "Non-Helmet" {
		t.Errorf("expected name Non-Helmet, got %s", item.ViolationName)
	}
	if item.VehicleType != "twoWheelVehicle" {
		t.Errorf("expected vehicleType twoWheelVehicle, got %s", item.VehicleType)
	}
	if item.PlateImageURL != filepath.Join(tempDir, "licensePlatePicture.jpg") {
		t.Errorf("unexpected PlateImageURL: %s", item.PlateImageURL)
	}
	if item.VehicleImageURL != filepath.Join(tempDir, "detectionPicture.jpg") {
		t.Errorf("unexpected VehicleImageURL: %s", item.VehicleImageURL)
	}
	if item.CaptureTime <= 0 {
		t.Errorf("expected valid capture time, got %d", item.CaptureTime)
	}
}
