package services

import (
	"backend/internal/models/domain"
	"backend/internal/repositories"
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type NanikaAPIService struct {
	DB                  *gorm.DB
	Log                 *logrus.Logger
	SeamenCacheRepo     *repositories.SeamenCacheRepository
	MutationCacheRepo   *repositories.MutationCacheRepository
	SeamenAPIURL        string
	MutationAPIURL      string
	HTTPTimeout         time.Duration
}

func NewNanikaAPIService(
	db *gorm.DB,
	log *logrus.Logger,
	seamenCacheRepo *repositories.SeamenCacheRepository,
	mutationCacheRepo *repositories.MutationCacheRepository,
) *NanikaAPIService {
	return &NanikaAPIService{
		DB:                db,
		Log:               log,
		SeamenCacheRepo:   seamenCacheRepo,
		MutationCacheRepo: mutationCacheRepo,
		SeamenAPIURL:      "http://nanika.spil.co.id:3021/get-seamen",
		MutationAPIURL:    "http://nanika.spil.co.id:3021/get-mutation",
		HTTPTimeout:       10 * time.Minute,
	}
}

func (s *NanikaAPIService) FetchAndCacheSeamenData() error {
	var lastErr error
	startTime := time.Now()
	s.Log.Info("📡 [SEAMEN API] Starting fetch from Nanika API...")

	for attempt := 1; attempt <= 3; attempt++ {
		s.Log.Infof("   ↳ Attempt %d/3: Calling API...", attempt)

		records, err := s.fetchSeamenFromAPI()
		if err != nil {
			lastErr = err
			s.Log.Warnf("   ↳ Attempt %d failed: %v", attempt, err)
			time.Sleep(5 * time.Second)
			continue
		}

		s.Log.Infof("   ↳ Fetched %d records from API (%.2f seconds)", len(records), time.Since(startTime).Seconds())
		s.Log.Info("   ↳ Deleting old seamen_cache records...")
		
		insertStart := time.Now()
		if err := s.SeamenCacheRepo.TruncateAndBatchInsert(records, 1000); err != nil {
			lastErr = err
			s.Log.Warnf("   ↳ Failed to save seamen cache: %v", err)
			continue
		}

		s.Log.Infof("✅ [SEAMEN API] Successfully cached %d records in %.2f seconds (batch size: 1000)", 
			len(records), time.Since(insertStart).Seconds())
		s.Log.Infof("   ↳ Total time: %.2f seconds", time.Since(startTime).Seconds())
		return nil
	}

	s.Log.Errorf("❌ [SEAMEN API] Failed after 3 attempts: %v", lastErr)
	return fmt.Errorf("API error after 3 retries: %w", lastErr)
}

func (s *NanikaAPIService) fetchSeamenFromAPI() ([]domain.SeamenCache, error) {
	payload := map[string]interface{}{
		"age":           0,
		"status":        "",
		"education":     "",
		"experience":    "",
		"certificate":   "",
		"last_location": "",
		"last_position": "",
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	client := &http.Client{Timeout: s.HTTPTimeout}
	req, err := http.NewRequest("GET", s.SeamenAPIURL, bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	var result struct {
		DataSeamen []struct {
			Age          string `json:"age"`
			Birthdate    string `json:"birthdate"`
			Birthplace   string `json:"birthplace"`
			Certificate  string `json:"certificate"`
			EduLevel     string `json:"edu_level"`
			EndDate      string `json:"end_date"`
			Experience   string `json:"experience"`
			Fleet        string `json:"fleet"`
			Gender       string `json:"gender"`
			LastLocation string `json:"last_location"`
			LastPosition string `json:"last_position"`
			LastVesselID string `json:"last_vesselid"`
			Name         string `json:"name"`
			PrevLocation string `json:"prevlocation"`
			PrevPosition string `json:"prevposition"`
			SeafarerCode string `json:"seafarercode"`
			SeamanCode   string `json:"seamancode"`
			StartDate    string `json:"start_date"`
			Status       string `json:"status"`
		} `json:"data_seamen"`
	}

	decoder := json.NewDecoder(resp.Body)
	if err := decoder.Decode(&result); err != nil {
		return nil, err
	}

	records := make([]domain.SeamenCache, 0, len(result.DataSeamen))
	for _, item := range result.DataSeamen {
		records = append(records, domain.SeamenCache{
			SeamanCode:   item.SeamanCode,
			SeafarerCode: item.SeafarerCode,
			Name:         item.Name,
			Age:          item.Age,
			Birthdate:    item.Birthdate,
			Birthplace:   item.Birthplace,
			Certificate:  item.Certificate,
			LastLocation: item.LastLocation,
			LastPosition: item.LastPosition,
			LastVesselID: item.LastVesselID,
			StartDate:    item.StartDate,
			EndDate:      item.EndDate,
			Status:       item.Status,
			Gender:       item.Gender,
			EduLevel:     item.EduLevel,
			Experience:   item.Experience,
			Fleet:        item.Fleet,
			PrevLocation: item.PrevLocation,
			PrevPosition: item.PrevPosition,
		})
	}

	return records, nil
}

func (s *NanikaAPIService) FetchAndCacheMutationData() error {
	var lastErr error
	startTime := time.Now()
	s.Log.Info(" [MUTATION API] Starting fetch from Nanika API...")

	for attempt := 1; attempt <= 3; attempt++ {
		s.Log.Infof("   ↳ Attempt %d/3: Calling API...", attempt)

		records, err := s.fetchMutationFromAPI()
		if err != nil {
			lastErr = err
			s.Log.Warnf("   ↳ Attempt %d failed: %v", attempt, err)
			time.Sleep(5 * time.Second)
			continue
		}

		s.Log.Infof("   ↳ Fetched %d records from API (%.2f seconds)", len(records), time.Since(startTime).Seconds())
		s.Log.Info("   ↳ Deleting old mutation_cache records...")
		
		insertStart := time.Now()
		if err := s.MutationCacheRepo.TruncateAndBatchInsert(records, 1000); err != nil {
			lastErr = err
			s.Log.Warnf("   ↳ Failed to save mutation cache: %v", err)
			continue
		}

		s.Log.Infof("[MUTATION API] Successfully cached %d records in %.2f seconds (batch size: 1000)", 
			len(records), time.Since(insertStart).Seconds())
		s.Log.Infof("   ↳ Total time: %.2f seconds", time.Since(startTime).Seconds())
		return nil
	}

	s.Log.Errorf("[MUTATION API] Failed after 3 attempts: %v", lastErr)
	return fmt.Errorf("API error after 3 retries: %w", lastErr)
}

// jakartaLocation falls back to a fixed WIB offset when tzdata is unavailable.
func jakartaLocation() *time.Location {
	if loc, err := time.LoadLocation("Asia/Jakarta"); err == nil {
		return loc
	}
	return time.FixedZone("WIB", 7*60*60)
}

func (s *NanikaAPIService) fetchMutationFromAPI() ([]domain.MutationCache, error) {
	payload := map[string]interface{}{
		"seaman_name":        "",
		"transaction_date_1": "01/01/2020",
		"transaction_date_2": time.Now().In(jakartaLocation()).Format("02/01/2006"),
		"from_rank_name":     "",
		"to_rank_name":       "",
		"from_vessel_code":   "",
		"to_vessel_code":     "",
		"jenis":              "",
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	client := &http.Client{Timeout: s.HTTPTimeout}
	req, err := http.NewRequest("GET", s.MutationAPIURL, bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	var result struct {
		DataMutation []struct {
			FromRankCode    string `json:"fromrankcode"`
			FromRankName    string `json:"fromrankname"`
			FromVesselCode  string `json:"fromvesselcode"`
			FromVesselName  string `json:"fromvesselname"`
			Jenis           string `json:"jenis"`
			MutationNoID    string `json:"mutationnoid"`
			SeamanCode      string `json:"seamancode"`
			SeamanName      string `json:"seamanname"`
			ToRankCode      string `json:"torankcode"`
			ToRankName      string `json:"torankname"`
			ToVesselCode    string `json:"tovesselcode"`
			ToVesselName    string `json:"tovesselname"`
			TransactionDate string `json:"transactiondate"`
		} `json:"data_mutation"`
	}

	decoder := json.NewDecoder(resp.Body)
	if err := decoder.Decode(&result); err != nil {
		return nil, err
	}

	records := make([]domain.MutationCache, 0, len(result.DataMutation))
	for _, item := range result.DataMutation {
		transDate, _ := time.Parse(time.RFC3339, item.TransactionDate)
		records = append(records, domain.MutationCache{
			SeamanCode:      item.SeamanCode,
			SeamanName:      item.SeamanName,
			TransactionDate: transDate,
			FromRankCode:    item.FromRankCode,
			FromRankName:    item.FromRankName,
			ToRankCode:      item.ToRankCode,
			ToRankName:      item.ToRankName,
			FromVesselCode:  item.FromVesselCode,
			FromVesselName:  item.FromVesselName,
			ToVesselCode:    item.ToVesselCode,
			ToVesselName:    item.ToVesselName,
			Jenis:           item.Jenis,
			MutationNoID:    item.MutationNoID,
		})
	}

	return records, nil
}
