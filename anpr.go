package main

import (
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// FTPXML represents the Hikvision/Dahua ANPR FTP XML payload
type FTPXML struct {
	XMLName         xml.Name        `xml:"FTP"`
	DateTime        string          `xml:"dateTime"`
	LicensePlate    string          `xml:"licensePlate"`
	PlateColor      string          `xml:"plateColor"`
	VehicleType     string          `xml:"vehicleType"`
	IllegalInfo     FTPIllegalInfo  `xml:"illegalInfo"`
	VehicleInfo     FTPVehicleInfo  `xml:"vehicleInfo"`
	PictureInfoList PictureInfoList `xml:"pictureInfoList"`
	UUID            string          `xml:"UUID"`
}

type FTPIllegalInfo struct {
	IllegalCode        string `xml:"illegalCode"`
	IllegalName        string `xml:"illegalName"`
	IllegalDescription string `xml:"illegalDescription"`
}

type FTPVehicleInfo struct {
	Color       string `xml:"color"`
	VehicleType string `xml:"vehicleType"`
	Speed       int    `xml:"speed"`
}

type PictureInfoList struct {
	PictureInfos []FTPPictureInfo `xml:"pictureInfo"`
}

type FTPPictureInfo struct {
	FileName string `xml:"fileName"`
	Type     string `xml:"type"`
	AbsTime  string `xml:"absTime"`
}

// ParsedANPRMeta contains metadata extracted from the filename:
// format: kodeUnix_NamaLokasi_alamat_tanggallengkapjam_nomerPlat
// or:     kodeUnix_alamat_tanggallengkapjam_nomerPlat
type ParsedANPRMeta struct {
	CameraCode   string // Kode unik kamera (misal: UNIX1)
	DeviceName   string
	LocationName string
	Address      string
	CaptureTime  time.Time
	Plate        string
}

// ParseANPRFilename parses the naming convention defined by PM:
// Contoh 5 bagian: UNIX1_pasuruan_Jalan A.Yani Bangil Kab.Pasuruan_20240716111021710_W5003QC
// Contoh 4 bagian: UNIX1_Jalan A.Yani Bangil Kab.Pasuruan_20240716111021710_W5003QC
func ParseANPRFilename(filename string) ParsedANPRMeta {
	base := filepath.Base(filename)
	ext := filepath.Ext(base)
	nameWithoutExt := strings.TrimSuffix(base, ext)

	parts := strings.Split(nameWithoutExt, "_")
	var meta ParsedANPRMeta

	switch {
	case len(parts) >= 5:
		meta.CameraCode = strings.TrimSpace(parts[0])
		meta.DeviceName = strings.TrimSpace(parts[0])
		meta.LocationName = strings.TrimSpace(parts[1])
		meta.Address = strings.TrimSpace(parts[2])
		meta.CaptureTime = parseCustomTimestamp(parts[3])
		meta.Plate = strings.TrimSpace(parts[4])
	case len(parts) == 4:
		meta.CameraCode = strings.TrimSpace(parts[0])
		meta.DeviceName = strings.TrimSpace(parts[0])
		meta.Address = strings.TrimSpace(parts[1])
		meta.LocationName = meta.Address
		meta.CaptureTime = parseCustomTimestamp(parts[2])
		meta.Plate = strings.TrimSpace(parts[3])
	case len(parts) >= 2:
		meta.CameraCode = strings.TrimSpace(parts[0])
		meta.DeviceName = strings.TrimSpace(parts[0])
		meta.Plate = strings.TrimSpace(parts[len(parts)-1])
		// Format nama file kamera umum: kodeUnix_YYYYMMDDHHmmssfff_noPlat
		// (3 bagian). Ambil waktu tangkap dari bagian kedua agar tidak
		// bergantung pada tag <dateTime> di dalam XML.
		if t := parseCustomTimestamp(parts[1]); !t.IsZero() {
			meta.CaptureTime = t
		}
	default:
		meta.CameraCode = nameWithoutExt
		meta.DeviceName = nameWithoutExt
	}

	return meta
}

// parseCustomTimestamp parses YYYYMMDDHHmmssfff (17 chars) or YYYYMMDDHHmmss (14 chars)
func parseCustomTimestamp(s string) time.Time {
	s = strings.TrimSpace(s)
	// 17 characters: 20240716111021710 -> 20060102150405.000
	if len(s) == 17 {
		t, err := time.ParseInLocation("20060102150405", s[:14], time.Local)
		if err == nil {
			var ms int
			_, _ = fmt.Sscanf(s[14:], "%d", &ms)
			return t.Add(time.Duration(ms) * time.Millisecond)
		}
	} else if len(s) == 14 {
		t, err := time.ParseInLocation("20060102150405", s, time.Local)
		if err == nil {
			return t
		}
	}
	return time.Time{}
}

// ParseANPRFile reads an ANPR XML file, extracts both filename metadata and XML body,
// and merges them into a ViolationItem ready to be enqueued.
func ParseANPRFile(xmlPath string) (*ViolationItem, error) {
	data, err := os.ReadFile(xmlPath)
	if err != nil {
		return nil, fmt.Errorf("read xml file: %w", err)
	}

	var ftp FTPXML
	if err := xml.Unmarshal(data, &ftp); err != nil {
		return nil, fmt.Errorf("decode xml: %w", err)
	}

	meta := ParseANPRFilename(xmlPath)

	// Plate: prioritaskan dari nama file, jika kosong fallback ke tag licensePlate di XML
	plate := meta.Plate
	if plate == "" || strings.EqualFold(plate, "unknown") {
		plate = ftp.LicensePlate
	}

	// CaptureTime: prioritaskan dari nama file, fallback ke tag dateTime di XML
	capTime := meta.CaptureTime
	if capTime.IsZero() && ftp.DateTime != "" {
		if t, err := time.Parse(time.RFC3339, ftp.DateTime); err == nil {
			capTime = t
		}
	}

	// Format location
	loc := meta.LocationName
	if meta.Address != "" && meta.Address != meta.LocationName {
		if loc != "" {
			loc = loc + ", " + meta.Address
		} else {
			loc = meta.Address
		}
	}

	// Resolusi file gambar foto plat dan foto kendaraan relatif terhadap folder XML
	baseDir := filepath.Dir(xmlPath)
	plateImg := ""
	vehicleImg := ""

	for _, p := range ftp.PictureInfoList.PictureInfos {
		fullPath := p.FileName
		if !filepath.IsAbs(fullPath) && p.FileName != "" {
			fullPath = filepath.Join(baseDir, p.FileName)
		}
		switch p.Type {
		case "licensePlatePicture":
			plateImg = fullPath
		case "detectionPicture":
			vehicleImg = fullPath
		}
	}

	// Jika plateImg atau vehicleImg masih kosong tapi ada item di pictureInfo
	if plateImg == "" && len(ftp.PictureInfoList.PictureInfos) > 0 {
		plateImg = filepath.Join(baseDir, ftp.PictureInfoList.PictureInfos[0].FileName)
	}
	if vehicleImg == "" && len(ftp.PictureInfoList.PictureInfos) > 1 {
		vehicleImg = filepath.Join(baseDir, ftp.PictureInfoList.PictureInfos[1].FileName)
	}

	// File snapshot yang dikirim kamera biasanya bernama sama dengan file XML
	// (mis. UNIX1_20260916043832189_EA8128KC.jpg). Jika file foto yang
	// direferensikan XML (licensePlatePicture.jpg / detectionPicture.jpg) tidak
	// ada di disk, gunakan file snapshot tersebut sebagai pengganti.
	natural := strings.TrimSuffix(xmlPath, filepath.Ext(xmlPath)) + ".jpg"
	if !fileExists(plateImg) && fileExists(natural) {
		plateImg = natural
	}
	if !fileExists(vehicleImg) && fileExists(natural) {
		vehicleImg = natural
	}

	item := &ViolationItem{
		DeviceName:      meta.DeviceName,
		Plate:           plate,
		PlateColor:      ftp.PlateColor,
		PlateImageURL:   plateImg,
		VehicleType:     ftp.VehicleType,
		VehicleColor:    ftp.VehicleInfo.Color,
		VehicleImageURL: vehicleImg,
		ViolationCode:   ftp.IllegalInfo.IllegalCode,
		ViolationName:   ftp.IllegalInfo.IllegalName,
		LocationName:    loc,
		CaptureTime:     capTime.UnixMilli(),
	}

	return item, nil
}
