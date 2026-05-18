package integration

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/rent-a-girlfriend/identity-service/internal/infrastructure/cache"
	"github.com/rent-a-girlfriend/identity-service/internal/infrastructure/persistence"
	"github.com/rent-a-girlfriend/identity-service/tests/testhelper"
)

func TestRedisCaching_Integration(t *testing.T) {
	db := testhelper.StartPostgresContainer(t)
	redisURL := testhelper.StartRedisContainer(t)

	redisAdapter, err := cache.NewRedisAdapter(redisURL)
	require.NoError(t, err)
	defer redisAdapter.Close()

	testhelper.WithTx(t, db, func(tx *gorm.DB) {
		ctx := context.Background()

		configRepo := persistence.NewSystemConfigRepoImpl(tx, redisAdapter)

		testKey := "test_lock_threshold"
		cacheKey := "config:" + testKey

		// Đảm bảo cache sạch trước khi test
		redisAdapter.Delete(ctx, cacheKey)

		// Seed dữ liệu trong transaction
		err := tx.Exec(
			"INSERT INTO system_configs (key, value, updated_at) VALUES (?, '5', NOW()) ON CONFLICT DO NOTHING",
			testKey,
		).Error
		require.NoError(t, err)

		// Lần 1: cache miss → đọc DB, ghi cache
		start := time.Now()
		val1, err := configRepo.GetInt(ctx, testKey, 0)
		require.NoError(t, err)
		assert.Equal(t, 5, val1)
		duration1 := time.Since(start)

		// Xác nhận Redis đã có cache
		var cachedVal int
		found, err := redisAdapter.Get(ctx, cacheKey, &cachedVal)
		require.NoError(t, err)
		assert.True(t, found, "Dữ liệu phải có trong Redis sau lần gọi đầu tiên")
		assert.Equal(t, 5, cachedVal)

		// Lần 2: cache hit → nhanh hơn
		start = time.Now()
		val2, err := configRepo.GetInt(ctx, testKey, 0)
		require.NoError(t, err)
		assert.Equal(t, 5, val2)
		duration2 := time.Since(start)

		t.Logf("Cache miss (DB): %v | Cache hit: %v", duration1, duration2)

		// Dọn cache (DB sẽ tự rollback)
		redisAdapter.Delete(ctx, cacheKey)
	})
	// TX rollback — system_configs không bị bẩn
}
