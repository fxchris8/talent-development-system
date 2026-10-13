package services

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/robfig/cron/v3"
	"github.com/sirupsen/logrus"
)

type CronService struct {
	cron                  *cron.Cron
	log                   *logrus.Logger
	IDPCalculationService *IDPCalculationService
	ApolloAPIService      *ApolloAPIService
	NanikaAPIService      *NanikaAPIService
	BatchSnapshotService  *BatchSnapshotService
	ReportService         *ReportService
	syncMu                sync.Mutex
}

func NewCronService(
	log *logrus.Logger,
	idpCalculationService *IDPCalculationService,
	apolloAPIService *ApolloAPIService,
	nanikaAPIService *NanikaAPIService,
	batchSnapshotService *BatchSnapshotService,
	reportService *ReportService,
) *CronService {
	// Create cron with seconds support
	c := cron.New(cron.WithSeconds())

	return &CronService{
		cron:                  c,
		log:                   log,
		IDPCalculationService: idpCalculationService,
		ApolloAPIService:      apolloAPIService,
		NanikaAPIService:      nanikaAPIService,
		BatchSnapshotService:  batchSnapshotService,
		ReportService:         reportService,
	}
}

// ErrCitrixSyncRunning is returned when a CITRIX sync is already in progress.
var ErrCitrixSyncRunning = errors.New("sinkronisasi CITRIX sedang berjalan")

// RunCitrixSync pulls seamen + mutation data from CITRIX into the cache tables
// and refreshes reports from them. It is shared by the cron job and the manual
// sync endpoint, and never runs twice at the same time.
func (s *CronService) RunCitrixSync() error {
	if !s.syncMu.TryLock() {
		return ErrCitrixSyncRunning
	}
	defer s.syncMu.Unlock()

	if err := s.NanikaAPIService.FetchAndCacheSeamenData(); err != nil {
		return fmt.Errorf("gagal mengambil data seamen: %w", err)
	}
	if err := s.NanikaAPIService.FetchAndCacheMutationData(); err != nil {
		return fmt.Errorf("gagal mengambil data mutasi: %w", err)
	}
	if err := s.ReportService.RefreshPersonalDataForAllReports(); err != nil {
		return fmt.Errorf("gagal memperbarui report: %w", err)
	}
	return nil
}

func (s *CronService) Start() error {
	s.log.Info("Starting cron service...")

	// Daily readiness calculation at 2 AM
	_, err := s.cron.AddFunc("0 0 2 * * *", func() {
		s.log.Info("Running daily readiness calculation...")
		if err := s.IDPCalculationService.CalculateReadinessForAllReports(); err != nil {
			s.log.Errorf("Failed to calculate readiness for all reports: %v", err)
		} else {
			s.log.Info("Daily readiness calculation completed successfully")
		}
	})
	if err != nil {
		return err
	}

	// Fetch CITRIX (Nanika) data every 10 minutes, then refresh reports
	_, err = s.cron.AddFunc("0 */10 * * * *", func() {
		s.log.Info("⏰ [CRON TRIGGERED] Fetching Nanika API every 10 minutes...")
		if err := s.RunCitrixSync(); err != nil {
			s.log.Errorf("❌ CITRIX sync failed: %v", err)
			return
		}
		s.log.Info("✅ [CRON FINISHED] Nanika API sync completed")
	})
	if err != nil {
		return err
	}

	// Clean expired cache every day at 3 AM
	_, err = s.cron.AddFunc("0 0 3 * * *", func() {
		s.log.Info("Cleaning expired Apollo API cache...")
		if err := s.ApolloAPIService.CleanExpiredCache(); err != nil {
			s.log.Errorf("Failed to clean expired cache: %v", err)
		} else {
			s.log.Info("Expired cache cleaned successfully")
		}
	})
	if err != nil {
		return err
	}

	// Daily batch auto-close at midnight: snapshot-and-close any active batches whose end_date < today
	// TESTING: every 1 minute (change to "0 0 0 * * *" for production)
	_, err = s.cron.AddFunc("0 0 0 * * *", func() {
		now := time.Now()
		s.log.Infof("[BATCH CRON] Running at %s (local time)", now.Format("2006-01-02 15:04:05 MST"))
		batches, err := s.BatchSnapshotService.BatchRepository.FindActivePastEndDate(
			s.BatchSnapshotService.DB,
		)
		if err != nil {
			s.log.Errorf("[BATCH CRON] Failed to find expired batches: %v", err)
			return
		}
		s.log.Infof("[BATCH CRON] Found %d expired batch(es) to close", len(batches))
		for _, b := range batches {
			s.log.Infof("[BATCH CRON] Processing batch id=%d no=%d endDate=%s status=%s",
				b.ID, b.BatchNo, b.EndDate.Format("2006-01-02"), b.Status)
			if err := s.BatchSnapshotService.SnapshotAndCloseBatch(b.ID); err != nil {
				s.log.Errorf("[BATCH CRON] ❌ Failed to close batch %d: %v", b.ID, err)
			} else {
				s.log.Infof("[BATCH CRON] ✅ Batch %d (no. %d) successfully closed and snapshotted", b.ID, b.BatchNo)
			}
		}
		s.log.Infof("[BATCH CRON] Done. %d batch(es) processed", len(batches))
	})
	if err != nil {
		return err
	}

	s.cron.Start()
	s.log.Info("Cron service started successfully")

	return nil
}

// Stop gracefully stops all cron jobs
func (s *CronService) Stop() {
	s.log.Info("Stopping cron service...")
	s.cron.Stop()
	s.log.Info("Cron service stopped")
}
