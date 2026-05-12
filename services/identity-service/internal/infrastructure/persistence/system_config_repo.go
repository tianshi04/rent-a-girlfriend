package persistence

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/rent-a-girlfriend/identity-service/internal/infrastructure/cache"
	"gorm.io/gorm"
)

// SystemConfigRepoImpl implements SystemConfigRepository using GORM.
type SystemConfigRepoImpl struct {
	db    *gorm.DB
	cache *cache.RedisAdapter
}

// NewSystemConfigRepoImpl creates a new repository implementation.
func NewSystemConfigRepoImpl(db *gorm.DB, cache *cache.RedisAdapter) *SystemConfigRepoImpl {
	return &SystemConfigRepoImpl{
		db:    db,
		cache: cache,
	}
}

func (r *SystemConfigRepoImpl) GetInt(ctx context.Context, key string, defaultVal int) (int, error) {
	cacheKey := fmt.Sprintf("config:%s", key)
	var cachedVal int

	// Try cache
	found, _ := r.cache.Get(ctx, cacheKey, &cachedVal)
	if found {
		return cachedVal, nil
	}

	var model SystemConfigModel
	if err := r.db.WithContext(ctx).Where("key = ?", key).First(&model).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return defaultVal, nil
		}
		return defaultVal, err
	}

	val, err := strconv.Atoi(model.Value)
	if err != nil {
		return defaultVal, nil
	}

	// Save to cache for 24 hours
	_ = r.cache.Set(ctx, cacheKey, val, 24*time.Hour)

	return val, nil
}
