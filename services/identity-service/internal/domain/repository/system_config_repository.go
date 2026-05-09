package repository

import "context"

// SystemConfigRepository is the port interface for reading system configuration from DB.
type SystemConfigRepository interface {
	// GetInt retrieves an integer configuration value by key.
	// Returns defaultVal if the key is not found.
	GetInt(ctx context.Context, key string, defaultVal int) (int, error)
}
