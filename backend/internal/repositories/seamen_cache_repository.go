package repositories

import (
	"backend/internal/models/domain"

	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type SeamenCacheRepository struct {
	DB  *gorm.DB
	Log *logrus.Logger
}

func NewSeamenCacheRepository(db *gorm.DB, log *logrus.Logger) *SeamenCacheRepository {
	return &SeamenCacheRepository{
		DB:  db,
		Log: log,
	}
}

func (r *SeamenCacheRepository) TruncateAndBatchInsert(records []domain.SeamenCache, batchSize int) error {
	tx := r.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Count existing records before delete
	var oldCount int64
	tx.Model(&domain.SeamenCache{}).Count(&oldCount)
	r.Log.Infof("      ↳ Deleting %d old records from seamen_cache...", oldCount)

	// Use DELETE instead of TRUNCATE to avoid blocking mysqldump backup
	if err := tx.Exec("DELETE FROM seamen_cache").Error; err != nil {
		tx.Rollback()
		return err
	}

	r.Log.Infof("      ↳ Inserting %d new records (batch size: %d)...", len(records), batchSize)
	if err := tx.CreateInBatches(records, batchSize).Error; err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit().Error
}

func (r *SeamenCacheRepository) GetBySeamanCode(seamanCode string) (*domain.SeamenCache, error) {
	var cache domain.SeamenCache
	if err := r.DB.Where("seaman_code = ?", seamanCode).First(&cache).Error; err != nil {
		return nil, err
	}
	return &cache, nil
}

func (r *SeamenCacheRepository) GetBySeafarerCode(seafarerCode string) (*domain.SeamenCache, error) {
	var cache domain.SeamenCache
	if err := r.DB.Where("seafarer_code = ?", seafarerCode).First(&cache).Error; err != nil {
		return nil, err
	}
	return &cache, nil
}

func (r *SeamenCacheRepository) GetAll() ([]domain.SeamenCache, error) {
	var caches []domain.SeamenCache
	if err := r.DB.Find(&caches).Error; err != nil {
		return nil, err
	}
	return caches, nil
}
