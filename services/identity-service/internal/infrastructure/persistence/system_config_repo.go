package persistence

import (
	"context"
	"strconv"

	"gorm.io/gorm"
)

// SystemConfigRepoImpl implements SystemConfigRepository using GORM.
type SystemConfigRepoImpl struct {
	db *gorm.DB
}

// NewSystemConfigRepoImpl creates a new repository implementation.
func NewSystemConfigRepoImpl(db *gorm.DB) *SystemConfigRepoImpl {
	return &SystemConfigRepoImpl{db: db}
}

func (r *SystemConfigRepoImpl) GetInt(ctx context.Context, key string, defaultVal int) (int, error) {
	var model SystemConfigModel
	if err := r.db.WithContext(ctx).Where("`key` = ?", key).First(&model).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return defaultVal, nil
		}
		return defaultVal, err
	}

	val, err := strconv.Atoi(model.Value)
	if err != nil {
		return defaultVal, nil
	}
	return val, nil
}
