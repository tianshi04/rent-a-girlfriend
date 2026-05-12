package integration

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rent-a-girlfriend/identity-service/internal/bootstrap"
	"github.com/rent-a-girlfriend/identity-service/internal/infrastructure/cache"
	"github.com/rent-a-girlfriend/identity-service/internal/infrastructure/persistence"
)

func TestRedisCaching_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Bỏ qua Redis integration test trong chế độ short mode")
	}

	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" || redisURL == "rediss://default:password@name.upstash.io:6379" {
		t.Skip("Bỏ qua test vì chưa cấu hình REDIS_URL thật")
	}

	cfg := bootstrap.LoadConfig()
	db, err := bootstrap.InitDatabase(cfg.Database)
	require.NoError(t, err)

	redisAdapter, err := cache.NewRedisAdapter(cfg.Redis.URL)
	require.NoError(t, err)
	defer redisAdapter.Close()

	ctx := context.Background()

	// Khởi tạo Repo với Cache
	configRepo := persistence.NewSystemConfigRepoImpl(db, redisAdapter)

	// Xóa key cấu hình để đảm bảo test sạch
	testKey := "test_lock_threshold"
	cacheKey := "config:" + testKey
	redisAdapter.Delete(ctx, cacheKey)

	// Đảm bảo có dữ liệu trong DB
	db.Exec("INSERT INTO system_configs (key, value, updated_at) VALUES (?, '5', NOW()) ON CONFLICT DO NOTHING", testKey)

	// Lần 1: Gọi hàm GetInt -> Phải gọi DB và lưu vào Cache
	start := time.Now()
	val1, err := configRepo.GetInt(ctx, testKey, 0)
	require.NoError(t, err)
	assert.Equal(t, 5, val1)
	duration1 := time.Since(start)

	// Kiểm tra xem Redis đã lưu cache chưa
	var cachedVal int
	found, err := redisAdapter.Get(ctx, cacheKey, &cachedVal)
	require.NoError(t, err)
	assert.True(t, found, "Dữ liệu phải có mặt trong Redis sau lần gọi đầu tiên")
	assert.Equal(t, 5, cachedVal)

	// Lần 2: Gọi hàm GetInt -> Phải lấy từ Cache, tốc độ phải nhanh hơn
	start = time.Now()
	val2, err := configRepo.GetInt(ctx, testKey, 0)
	require.NoError(t, err)
	assert.Equal(t, 5, val2)
	duration2 := time.Since(start)

	t.Logf("Lần 1 (Miss DB): %v", duration1)
	t.Logf("Lần 2 (Hit Cache): %v", duration2)

	// Xóa dữ liệu test
	redisAdapter.Delete(ctx, cacheKey)
	db.Exec("DELETE FROM system_configs WHERE key = ?", testKey)
}
