package main

import "strings"

// CameraCodeToETLE memetakan kode pelanggaran dari kamera ANPR
// (illegalCode di XML, mis. Hikvision/Huayuan numeric) ke kode master
// ETLE Korlantas (huruf, sesuai tabel master_violations hasil /master/list).
var CameraCodeToETLE = map[string]string{
	// ---- kode ANPR angka -> kode ETLE huruf ----
	"1240":   "PS", // Tidak menggunakan sabuk pengaman
	"1241":   "PS", // Penumpang tidak memakai sabuk pengaman
	"1350":   "PO", // Melanggar kecepatan maksimum/minimum
	"2004":   "LA", // Cara mengemudi yang salah (melawan arus)
	"1625":   "LM", // Menerobos lampu merah
	"1223":   "HB", // Menggunakan handphone
	"1208":   "MR", // Melanggar rambu atau marka
	"K0002":  "GG", // Melanggar aturan Ganjil Genap
	"1301":   "PU", // Melanggar larangan putar balik
	"13451":  "MR", // Melanggar garis marka
	"1000":   "PH", // Tidak mengenakan Helm
	"1001":   "PL", // Berbonceng 3 (muatan penumpang lebih)
	"1002":   "JB", // Masuk ke jalur bus
	"151002": "PH", // Non-Helmet
	"3019":   "PS", // Front passenger not buckled up (sabuk pengaman)

	// ---- kode E-KIR / perundangan -> kode ETLE huruf ----
	"E001": "ST", // STNK, atau STCK tidak sah
	"E002": "PT", // Persyaratan teknis dan laik jalan SPM
	"E004": "MO", // Mengangkut orang
	"E005": "TT", // TNKB tidak sah
	"E006": "PM", // Penumpang tidak menggunakan helm
	"E007": "ST", // STNK tidak ada pengesahan tahunan
	"E010": "MH", // Melanggar hak pejalan kaki atau pesepeda
	"E014": "LP", // Melanggar larangan parkir
}

// MapViolationCode memetakan kode kamera ke kode master ETLE Korlantas.
// Mengembalikan kode ETLE dan true bila ada mapping; bila tidak ada mapping
// mengembalikan kode asli dan false (berarti harus ditolak, bukan dikirim apa adanya).
func MapViolationCode(code string) (string, bool) {
	mapped, ok := CameraCodeToETLE[strings.TrimSpace(code)]
	return mapped, ok
}
