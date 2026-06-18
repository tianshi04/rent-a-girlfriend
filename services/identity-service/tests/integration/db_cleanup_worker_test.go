package integration

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rent-a-girlfriend/identity-service/internal/infrastructure/persistence"
	"github.com/rent-a-girlfriend/identity-service/internal/infrastructure/worker"
	"github.com/rent-a-girlfriend/identity-service/tests/testhelper"
)

func TestDbCleanupWorker_Integration(t *testing.T) {
	db := testhelper.StartPostgresContainer(t)

	// Clean tables before running
	err := db.Exec("DELETE FROM outbox_events").Error
	require.NoError(t, err)
	err = db.Exec("DELETE FROM pkce_verifiers").Error
	require.NoError(t, err)

	// Setup Test Data
	now := time.Now().UTC()
	retentionDays := 7

	// 1. Outbox events:
	// Event A: Published and older than retention (should be deleted)
	eventA := persistence.OutboxModel{
		ID:        uuid.New(),
		EventType: "test.event.v1",
		Payload:   `{"data": "A"}`,
		Published: true,
		CreatedAt: now.AddDate(0, 0, -(retentionDays + 2)), // 9 days ago
	}
	// Event B: Published and within retention (should be preserved)
	eventB := persistence.OutboxModel{
		ID:        uuid.New(),
		EventType: "test.event.v1",
		Payload:   `{"data": "B"}`,
		Published: true,
		CreatedAt: now.AddDate(0, 0, -(retentionDays - 2)), // 5 days ago
	}
	// Event C: Unpublished and older than retention (should be preserved as a safety safeguard)
	eventC := persistence.OutboxModel{
		ID:        uuid.New(),
		EventType: "test.event.v1",
		Payload:   `{"data": "C"}`,
		Published: false,
		CreatedAt: now.AddDate(0, 0, -(retentionDays + 2)), // 9 days ago
	}

	require.NoError(t, db.Create(&eventA).Error)
	require.NoError(t, db.Create(&eventB).Error)
	require.NoError(t, db.Create(&eventC).Error)

	// 2. PKCE Verifiers:
	// Verifier X: Expired (expires_at in the past) (should be deleted)
	verifierX := persistence.PKCEVerifierModel{
		State:        "state-expired",
		CodeVerifier: "verifier-expired",
		ExpiresAt:    now.Add(-5 * time.Minute),
	}
	// Verifier Y: Active (expires_at in the future) (should be preserved)
	verifierY := persistence.PKCEVerifierModel{
		State:        "state-active",
		CodeVerifier: "verifier-active",
		ExpiresAt:    now.Add(10 * time.Minute),
	}

	require.NoError(t, db.Create(&verifierX).Error)
	require.NoError(t, db.Create(&verifierY).Error)

	// Initialize the worker
	// Interval is long (e.g. 1 hour) because we only want to test the initial startup run.
	workerInstance := worker.NewDbCleanupWorker(db, 1*time.Hour, retentionDays)

	// Run the worker and stop it after a brief moment
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	workerInstance.Start(ctx)

	// Assertions
	// Check Outbox events
	var outboxEvents []persistence.OutboxModel
	err = db.Find(&outboxEvents).Error
	require.NoError(t, err)

	assert.Len(t, outboxEvents, 2, "Should preserve exactly 2 outbox events")

	var foundB, foundC bool
	for _, evt := range outboxEvents {
		if evt.ID == eventB.ID {
			foundB = true
		}
		if evt.ID == eventC.ID {
			foundC = true
		}
	}
	assert.True(t, foundB, "Recent published event B should be preserved")
	assert.True(t, foundC, "Old unpublished event C should be preserved")

	// Check PKCE Verifiers
	var verifiers []persistence.PKCEVerifierModel
	err = db.Find(&verifiers).Error
	require.NoError(t, err)

	assert.Len(t, verifiers, 1, "Should preserve exactly 1 PKCE verifier")
	assert.Equal(t, verifierY.State, verifiers[0].State, "Active PKCE verifier should be preserved")
}
