package repositories

import (
	"strings"

	"backend/internal/models/web"

	"gorm.io/gorm"
)

type SeamanRepository struct {
	DB *gorm.DB
}

func NewSeamanRepository(db *gorm.DB) *SeamanRepository {
	return &SeamanRepository{DB: db}
}

func (r *SeamanRepository) FindAvailableSeamen(
	query string,
	limit int,
) ([]web.SeamanLookupResponse, error) {

	if limit <= 0 || limit > 50 {
		limit = 10
	}

	q := "%" + strings.ToLower(query) + "%"

	var results []web.SeamanLookupResponse

	// Seamen who already have a report are still listed (flagged), otherwise a search here
	// looks different from the dashboard/master list for the same name.
	err := r.DB.Raw(`
		SELECT
			s.name AS name,
			s.seaman_code AS seaman_code,
			EXISTS (
				SELECT 1 FROM reports m WHERE m.seaman_code = s.seaman_code
			) AS already_added
		FROM seamen_cache s
		WHERE s.name != ''
		AND s.name IS NOT NULL
		AND (
			LOWER(s.name) LIKE ?
			OR s.seaman_code LIKE ?
		)
		ORDER BY already_added ASC, s.name
		LIMIT ?
	`, q, "%"+query+"%", limit).
		Scan(&results).Error

	return results, err
}
