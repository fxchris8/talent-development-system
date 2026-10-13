package services

import (
	"encoding/json"
	"strings"
	"time"

	"backend/internal/models/domain"
)

// Helpers that copy CITRIX (seamen_cache / mutation_cache) data into a report row.
// Shared by the scheduled/manual refresh and by report creation, so both fill exactly
// the same columns.

func parseCitrixDate(value string) *time.Time {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	for _, format := range []string{"02/01/2006", "2006-01-02", "02-01-2006"} {
		if t, err := time.Parse(format, value); err == nil {
			return &t
		}
	}
	return nil
}

type vesselHistoryEntry struct {
	TransactionDate string `json:"transaction_date"`
	FromVessel      string `json:"from_vessel"`
	ToVessel        string `json:"to_vessel"`
	FromRank        string `json:"from_rank"`
	ToRank          string `json:"to_rank"`
	Type            string `json:"type"`
}

// buildVesselHistoryJSON serialises the mutation history. It uses encoding/json, so vessel
// names containing quotes or backslashes can no longer produce invalid JSON.
func buildVesselHistoryJSON(mutations []domain.MutationCache) string {
	if len(mutations) == 0 {
		return "[]"
	}

	entries := make([]vesselHistoryEntry, 0, len(mutations))
	for _, m := range mutations {
		transactionDate := "-"
		if !m.TransactionDate.IsZero() {
			transactionDate = m.TransactionDate.Format("2006-01-02")
		}
		entries = append(entries, vesselHistoryEntry{
			TransactionDate: transactionDate,
			FromVessel:      getStringOrDefault(m.FromVesselName, "-"),
			ToVessel:        getStringOrDefault(m.ToVesselName, "-"),
			FromRank:        getStringOrDefault(m.FromRankName, "-"),
			ToRank:          getStringOrDefault(m.ToRankName, "-"),
			Type:            getStringOrDefault(m.Jenis, "-"),
		})
	}

	out, err := json.Marshal(entries)
	if err != nil {
		return "[]"
	}
	return string(out)
}

// citrixReportUpdates is the set of report columns that come from CITRIX.
func citrixReportUpdates(cache *domain.SeamenCache, mutations []domain.MutationCache) map[string]interface{} {
	return map[string]interface{}{
		"nama":           cache.Name,
		"tanggal_lahir":  cache.Birthdate,
		"age":            cache.Age,
		"jabatan":        cache.LastPosition,
		"vessel_name":    cache.LastLocation,
		"seafarer_code":  cache.SeafarerCode,
		"certificate":    cache.Certificate,
		"start_date":     parseCitrixDate(cache.StartDate),
		"vessel_history": buildVesselHistoryJSON(mutations),
	}
}

func fillStringIfBlank(dst **string, value string) {
	value = strings.TrimSpace(value)
	if value == "" {
		return
	}
	if *dst == nil || strings.TrimSpace(**dst) == "" {
		v := value
		*dst = &v
	}
}

// FillReportFromCitrix completes a new report with CITRIX data. Values the user already
// typed are kept; only blank columns are filled.
func FillReportFromCitrix(report *domain.FullReport, cache *domain.SeamenCache, mutations []domain.MutationCache) {
	fillStringIfBlank(&report.Nama, cache.Name)
	fillStringIfBlank(&report.Jabatan, cache.LastPosition)
	fillStringIfBlank(&report.VesselName, cache.LastLocation)
	fillStringIfBlank(&report.SeafarerCode, cache.SeafarerCode)
	fillStringIfBlank(&report.Certificate, cache.Certificate)
	fillStringIfBlank(&report.Age, cache.Age)
	fillStringIfBlank(&report.TanggalLahir, cache.Birthdate)

	if report.StartDate == nil {
		report.StartDate = parseCitrixDate(cache.StartDate)
	}
	fillStringIfBlank(&report.VesselHistory, buildVesselHistoryJSON(mutations))
	if report.VesselHistory != nil && *report.VesselHistory == "[]" && len(mutations) == 0 {
		// no history yet: keep the column empty instead of a literal "[]"
		report.VesselHistory = nil
	}
}

func setIfBlank(dst *string, value string) {
	value = strings.TrimSpace(value)
	if value != "" && strings.TrimSpace(*dst) == "" {
		*dst = value
	}
}

// overlayCitrixOnMasterRow fills the blank personal columns of a master list row with the
// CITRIX record. Values already stored on the row win; columns CITRIX does not have stay
// empty (the UI shows "-").
func overlayCitrixOnMasterRow(row *domain.MasterReport, cache *domain.SeamenCache) {
	setIfBlank(&row.VesselName, cache.LastLocation)
	setIfBlank(&row.Jabatan, cache.LastPosition)
	setIfBlank(&row.SeafarerCode, cache.SeafarerCode)
	setIfBlank(&row.Certificate, cache.Certificate)
	setIfBlank(&row.Age, cache.Age)
	setIfBlank(&row.Nama, cache.Name)
	if strings.TrimSpace(row.StartDate) == "" {
		if d := parseCitrixDate(cache.StartDate); d != nil {
			row.StartDate = d.Format("2006-01-02")
		}
	}
}
