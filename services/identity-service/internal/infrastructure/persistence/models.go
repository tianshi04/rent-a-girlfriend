package persistence

import (
	"time"

	"github.com/google/uuid"
)

// UserAccountModel is the GORM model for the user_accounts table.
type UserAccountModel struct {
	ID             uuid.UUID `gorm:"type:uuid;primaryKey"`
	Email          string    `gorm:"type:varchar(255);uniqueIndex;not null"`
	GoogleID       string    `gorm:"type:varchar(255);uniqueIndex;not null;column:google_id"`
	Role           string    `gorm:"type:varchar(20);not null;default:CLIENT"`
	Status         string    `gorm:"type:varchar(20);not null;default:ACTIVE"`
	ViolationCount int       `gorm:"not null;default:0;column:violation_count"`
	Version        int       `gorm:"not null;default:1"`
	CreatedAt      time.Time `gorm:"not null;column:created_at"`
	UpdatedAt      time.Time `gorm:"not null;column:updated_at"`
}

func (UserAccountModel) TableName() string { return "user_accounts" }

// RefreshTokenModel is the GORM model for the refresh_tokens table.
type RefreshTokenModel struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey"`
	UserID    uuid.UUID `gorm:"type:uuid;not null;index:idx_rt_user;column:user_id"`
	TokenHash string    `gorm:"type:varchar(255);not null;column:token_hash"`
	FamilyID  uuid.UUID `gorm:"type:uuid;not null;index:idx_rt_family;column:family_id"`
	IsRevoked bool      `gorm:"not null;default:false;column:is_revoked"`
	ExpiresAt time.Time `gorm:"not null;column:expires_at"`
	CreatedAt time.Time `gorm:"not null;column:created_at"`
}

func (RefreshTokenModel) TableName() string { return "refresh_tokens" }

// SigningKeyModel is the GORM model for the signing_keys table.
type SigningKeyModel struct {
	Kid           string     `gorm:"type:varchar(50);primaryKey"`
	PrivateKeyPEM string     `gorm:"type:text;not null;column:private_key_pem"`
	PublicKeyPEM  string     `gorm:"type:text;not null;column:public_key_pem"`
	IsActive      bool       `gorm:"not null;default:true;column:is_active"`
	CreatedAt     time.Time  `gorm:"not null;column:created_at"`
	ExpiresAt     *time.Time `gorm:"column:expires_at"`
}

func (SigningKeyModel) TableName() string { return "signing_keys" }

// UpgradeRequestModel is the GORM model for the upgrade_requests table.
type UpgradeRequestModel struct {
	ID           uuid.UUID  `gorm:"type:uuid;primaryKey"`
	UserID       uuid.UUID  `gorm:"type:uuid;not null;index:idx_upgrade_user;column:user_id"`
	Status       string     `gorm:"type:varchar(20);not null;default:PENDING;index:idx_upgrade_status"`
	Reason       string     `gorm:"type:text"`
	RejectReason string     `gorm:"type:text;column:reject_reason"`
	ReviewedBy   string     `gorm:"type:varchar(255);column:reviewed_by"`
	ReviewedAt   *time.Time `gorm:"column:reviewed_at"`
	CreatedAt    time.Time  `gorm:"not null;column:created_at"`
}

func (UpgradeRequestModel) TableName() string { return "upgrade_requests" }

// SystemConfigModel is the GORM model for the system_configs table.
type SystemConfigModel struct {
	Key       string    `gorm:"type:varchar(100);primaryKey"`
	Value     string    `gorm:"type:text;not null"`
	UpdatedAt time.Time `gorm:"not null;column:updated_at"`
}

func (SystemConfigModel) TableName() string { return "system_configs" }

// OutboxModel is the GORM model for the outbox_events table.
type OutboxModel struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey"`
	EventType string    `gorm:"type:varchar(255);not null;column:event_type"`
	Payload   string    `gorm:"type:jsonb;not null"`
	Published bool      `gorm:"not null;default:false"`
	CreatedAt time.Time `gorm:"not null;column:created_at"`
}

func (OutboxModel) TableName() string { return "outbox_events" }

// PKCEVerifierModel is the GORM model for the pkce_verifiers table.
type PKCEVerifierModel struct {
	State        string    `gorm:"type:varchar(255);primaryKey"`
	CodeVerifier string    `gorm:"type:varchar(255);not null;column:code_verifier"`
	ExpiresAt    time.Time `gorm:"not null;column:expires_at"`
}

func (PKCEVerifierModel) TableName() string { return "pkce_verifiers" }
