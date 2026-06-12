package worker

import (
	"context"
	"log"
	"time"

	"gorm.io/gorm"

	"github.com/rent-a-girlfriend/identity-service/internal/infrastructure/persistence"
)

type DbCleanupWorker struct {
	db              *gorm.DB
	cleanupInterval time.Duration
	retentionDays   int
}

func NewDbCleanupWorker(db *gorm.DB, cleanupInterval time.Duration, retentionDays int) *DbCleanupWorker {
	return &DbCleanupWorker{
		db:              db,
		cleanupInterval: cleanupInterval,
		retentionDays:   retentionDays,
	}
}

func (w *DbCleanupWorker) Start(ctx context.Context) {
	ticker := time.NewTicker(w.cleanupInterval)
	defer ticker.Stop()

	log.Printf("[DB-CLEANUP-WORKER] Started, running every %v, retention=%d days", w.cleanupInterval, w.retentionDays)

	w.runCleanup(ctx)

	for {
		select {
		case <-ctx.Done():
			log.Println("[DB-CLEANUP-WORKER] Stopping...")
			return
		case <-ticker.C:
			w.runCleanup(ctx)
		}
	}
}

func (w *DbCleanupWorker) runCleanup(ctx context.Context) {
	cutoff := time.Now().AddDate(0, 0, -w.retentionDays)
	log.Printf("[DB-CLEANUP-WORKER] Starting cleanup cycle. Cutoff time: %s", cutoff.Format(time.RFC3339))

	resOutbox := w.db.WithContext(ctx).
		Where("published = ? AND created_at < ?", true, cutoff).
		Delete(&persistence.OutboxModel{})
	if resOutbox.Error != nil {
		log.Printf("[DB-CLEANUP-WORKER] Error cleaning up outbox: %v", resOutbox.Error)
	} else if resOutbox.RowsAffected > 0 {
		log.Printf("[DB-CLEANUP-WORKER] Cleaned up %d published outbox records", resOutbox.RowsAffected)
	}

	resPKCE := w.db.WithContext(ctx).
		Where("expires_at < ?", time.Now()).
		Delete(&persistence.PKCEVerifierModel{})
	if resPKCE.Error != nil {
		log.Printf("[DB-CLEANUP-WORKER] Error cleaning up expired PKCE verifiers: %v", resPKCE.Error)
	} else if resPKCE.RowsAffected > 0 {
		log.Printf("[DB-CLEANUP-WORKER] Cleaned up %d expired PKCE verifiers", resPKCE.RowsAffected)
	}

	log.Println("[DB-CLEANUP-WORKER] Cleanup cycle finished")
}
