// Package testhelper cung cấp các tiện ích dùng chung cho integration tests.
// Tất cả containers được tự động destroy khi test kết thúc qua t.Cleanup().
package testhelper

import (
	"context"
	"fmt"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	gormpostgres "gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// TestDB holds connection info for the ephemeral test database.
type TestDB struct {
	DB     *gorm.DB
	DSN    string // postgres://... URL
}

// StartPostgresContainer khởi động một PostgreSQL ephemeral container,
// chạy toàn bộ migrations, và trả về *gorm.DB đã connect.
// Container sẽ tự bị destroy khi test kết thúc.
func StartPostgresContainer(t *testing.T) *gorm.DB {
	t.Helper()
	ctx := context.Background()

	req := testcontainers.ContainerRequest{
		Image:        "postgres:16-alpine",
		ExposedPorts: []string{"5432/tcp"},
		Env: map[string]string{
			"POSTGRES_DB":       "identity_test",
			"POSTGRES_USER":     "test",
			"POSTGRES_PASSWORD": "test",
		},
		WaitingFor: wait.ForAll(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(60*time.Second),
			wait.ForListeningPort("5432/tcp"),
		),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	require.NoError(t, err, "failed to start postgres container")

	t.Cleanup(func() {
		if err := container.Terminate(ctx); err != nil {
			t.Logf("warning: failed to terminate postgres container: %v", err)
		}
	})

	host, err := container.Host(ctx)
	require.NoError(t, err)
	port, err := container.MappedPort(ctx, "5432")
	require.NoError(t, err)

	pgURL := fmt.Sprintf(
		"postgres://test:test@%s:%s/identity_test?sslmode=disable",
		host, port.Port(),
	)

	runMigrations(t, pgURL)

	db, err := gorm.Open(gormpostgres.Open(pgURL), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err, "failed to open gorm connection to test postgres")

	return db
}

// WithTx chạy fn bên trong một transaction và luôn ROLLBACK sau khi xong,
// đảm bảo test không để lại dữ liệu bẩn trong DB.
func WithTx(t *testing.T, db *gorm.DB, fn func(tx *gorm.DB)) {
	t.Helper()
	tx := db.Begin()
	require.NoError(t, tx.Error, "failed to begin transaction")
	defer func() {
		if rbErr := tx.Rollback().Error; rbErr != nil && rbErr != gorm.ErrInvalidTransaction {
			t.Logf("warning: rollback error: %v", rbErr)
		}
	}()
	fn(tx)
}

// runMigrations chạy tất cả file *.up.sql trong thư mục migrations/ của service.
func runMigrations(t *testing.T, pgURL string) {
	t.Helper()

	// Tính đường dẫn đến migrations/ tương đối từ file này trong source tree:
	// tests/testhelper/postgres.go → ../../migrations
	_, thisFile, _, ok := runtime.Caller(0)
	require.True(t, ok, "cannot determine source file path")

	migrationsDir := filepath.Join(filepath.Dir(thisFile), "..", "..", "migrations")
	migrationsURL := "file://" + filepath.ToSlash(migrationsDir)

	m, err := migrate.New(migrationsURL, pgURL)
	require.NoError(t, err, "failed to create migrate instance; migrations path: %s", migrationsDir)

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		require.NoError(t, err, "failed to run migrations")
	}
}
