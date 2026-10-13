package services

import (
	"encoding/json"
	"testing"
	"time"

	"backend/internal/models/domain"
)

func strp(s string) *string { return &s }

func sampleCache() *domain.SeamenCache {
	return &domain.SeamenCache{
		SeamanCode: "1001", SeafarerCode: "SF-77", Name: "BUDI SANTOSO", Age: "41",
		Birthdate: "12/03/1985", Certificate: "ANT-II", LastLocation: "KM. ORIENTAL JADE",
		LastPosition: "MUALIM I", StartDate: "01/06/2026",
	}
}

func TestFillReportFromCitrixFillsEveryBlankColumn(t *testing.T) {
	// what the master form really sends: name + seaman code, everything else empty strings
	report := &domain.FullReport{
		Nama: strp("BUDI SANTOSO"), SeamanCode: strp("1001"),
		VesselName: strp(""), Jabatan: strp(""), SeafarerCode: strp(""), Certificate: strp(""),
	}
	mutations := []domain.MutationCache{{
		TransactionDate: time.Date(2026, 1, 5, 0, 0, 0, 0, time.UTC),
		FromVesselName:  "KM. A", ToVesselName: "KM. ORIENTAL JADE", Jenis: "MUTASI",
	}}

	FillReportFromCitrix(report, sampleCache(), mutations)

	got := map[string]string{
		"jabatan": *report.Jabatan, "vessel": *report.VesselName, "seafarer": *report.SeafarerCode,
		"cert": *report.Certificate, "age": *report.Age, "lahir": *report.TanggalLahir,
	}
	want := map[string]string{
		"jabatan": "MUALIM I", "vessel": "KM. ORIENTAL JADE", "seafarer": "SF-77",
		"cert": "ANT-II", "age": "41", "lahir": "12/03/1985",
	}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("%s = %q, want %q", k, got[k], v)
		}
	}
	if report.StartDate == nil || report.StartDate.Format("2006-01-02") != "2026-06-01" {
		t.Errorf("start date not parsed: %v", report.StartDate)
	}
	var history []map[string]string
	if report.VesselHistory == nil || json.Unmarshal([]byte(*report.VesselHistory), &history) != nil || len(history) != 1 {
		t.Fatalf("vessel history not filled with valid JSON: %v", report.VesselHistory)
	}
	if history[0]["to_vessel"] != "KM. ORIENTAL JADE" || history[0]["transaction_date"] != "2026-01-05" {
		t.Errorf("unexpected history entry: %v", history[0])
	}
}

func TestFillReportFromCitrixKeepsWhatTheUserTyped(t *testing.T) {
	report := &domain.FullReport{Nama: strp("NAMA KETIK SENDIRI"), Jabatan: strp("NAKHODA"), VesselName: strp("")}
	FillReportFromCitrix(report, sampleCache(), nil)

	if *report.Nama != "NAMA KETIK SENDIRI" || *report.Jabatan != "NAKHODA" {
		t.Errorf("user input was overwritten: %q / %q", *report.Nama, *report.Jabatan)
	}
	if *report.VesselName != "KM. ORIENTAL JADE" {
		t.Errorf("blank vessel not filled: %q", *report.VesselName)
	}
	if report.VesselHistory != nil {
		t.Errorf("empty mutation history should stay NULL, got %q", *report.VesselHistory)
	}
}

func TestVesselHistoryJSONSurvivesQuotesAndBackslashes(t *testing.T) {
	out := buildVesselHistoryJSON([]domain.MutationCache{{
		FromVesselName: `KM. "QUOTED" \ SHIP`, ToVesselName: "KM. B",
	}})
	var parsed []map[string]string
	if err := json.Unmarshal([]byte(out), &parsed); err != nil {
		t.Fatalf("invalid JSON %q: %v", out, err)
	}
	if parsed[0]["from_vessel"] != `KM. "QUOTED" \ SHIP` {
		t.Errorf("name corrupted: %q", parsed[0]["from_vessel"])
	}
	if buildVesselHistoryJSON(nil) != "[]" {
		t.Error("empty history must be []")
	}
}

func TestParseCitrixDate(t *testing.T) {
	for _, in := range []string{"01/06/2026", "2026-06-01", "01-06-2026", " 01/06/2026 "} {
		if d := parseCitrixDate(in); d == nil || d.Format("2006-01-02") != "2026-06-01" {
			t.Errorf("%q parsed as %v", in, d)
		}
	}
	if parseCitrixDate("") != nil || parseCitrixDate("bukan tanggal") != nil {
		t.Error("blank/invalid must be nil")
	}
}

func TestOverlayCitrixOnMasterRowFillsOnlyBlanks(t *testing.T) {
	row := &domain.MasterReport{Nama: "ADIAZA FAJAR ANJANA", SeamanCode: "1001", Jabatan: "NAKHODA"}
	overlayCitrixOnMasterRow(row, sampleCache())

	if row.Jabatan != "NAKHODA" {
		t.Errorf("stored value overwritten: %q", row.Jabatan)
	}
	if row.VesselName != "KM. ORIENTAL JADE" || row.Certificate != "ANT-II" || row.Age != "41" || row.SeafarerCode != "SF-77" {
		t.Errorf("blank columns not filled: %+v", row)
	}
	if row.StartDate != "2026-06-01" {
		t.Errorf("start date = %q", row.StartDate)
	}

	// CITRIX has nothing for these -> stay empty (UI shows "-")
	empty := &domain.MasterReport{SeamanCode: "2"}
	overlayCitrixOnMasterRow(empty, &domain.SeamenCache{SeamanCode: "2"})
	if empty.VesselName != "" || empty.Age != "" || empty.StartDate != "" {
		t.Errorf("must stay blank: %+v", empty)
	}
}
