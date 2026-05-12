package crypto

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"math/big"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/rent-a-girlfriend/identity-service/internal/application/port"
	"github.com/rent-a-girlfriend/identity-service/internal/infrastructure/persistence"
)

// RSAKeyProvider manages RSA keys and provides JWKS.
type RSAKeyProvider struct {
	db *gorm.DB
}

// NewRSAKeyProvider creates a new key provider.
func NewRSAKeyProvider(db *gorm.DB) *RSAKeyProvider {
	return &RSAKeyProvider{db: db}
}

// EnsureSigningKey creates a signing key if none exists.
func (p *RSAKeyProvider) EnsureSigningKey() error {
	var count int64
	p.db.Model(&persistence.SigningKeyModel{}).Where("is_active = ?", true).Count(&count)
	if count > 0 {
		return nil
	}

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return fmt.Errorf("failed to generate RSA key: %w", err)
	}

	privPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	})

	pubBytes, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	if err != nil {
		return fmt.Errorf("failed to marshal public key: %w", err)
	}
	pubPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: pubBytes,
	})

	kid := fmt.Sprintf("v1.%s", uuid.New().String()[:8])

	model := persistence.SigningKeyModel{
		Kid:           kid,
		PrivateKeyPEM: string(privPEM),
		PublicKeyPEM:  string(pubPEM),
		IsActive:      true,
		CreatedAt:     time.Now(),
	}

	return p.db.Create(&model).Error
}

// GetActiveKey returns the current active private key and its kid.
func (p *RSAKeyProvider) GetActiveKey() (*rsa.PrivateKey, string, error) {
	var model persistence.SigningKeyModel
	if err := p.db.Where("is_active = ?", true).Order("created_at DESC").First(&model).Error; err != nil {
		return nil, "", fmt.Errorf("no active signing key: %w", err)
	}

	block, _ := pem.Decode([]byte(model.PrivateKeyPEM))
	if block == nil {
		return nil, "", fmt.Errorf("failed to decode PEM")
	}

	privKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return nil, "", fmt.Errorf("failed to parse private key: %w", err)
	}

	return privKey, model.Kid, nil
}

// GetPublicKey returns the public key for a given kid.
func (p *RSAKeyProvider) GetPublicKey(kid string) (*rsa.PublicKey, error) {
	var model persistence.SigningKeyModel
	if err := p.db.Where("kid = ?", kid).First(&model).Error; err != nil {
		return nil, fmt.Errorf("public key not found for kid %s: %w", kid, err)
	}

	block, _ := pem.Decode([]byte(model.PublicKeyPEM))
	if block == nil {
		return nil, fmt.Errorf("failed to decode public key PEM")
	}

	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse public key: %w", err)
	}

	rsaPub, ok := pub.(*rsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("not an RSA public key")
	}

	return rsaPub, nil
}

// GetJWKS returns all active public keys in JWKS format.
func (p *RSAKeyProvider) GetJWKS() (*port.JWKSResponse, error) {
	var models []persistence.SigningKeyModel
	if err := p.db.Where("is_active = ?", true).Find(&models).Error; err != nil {
		return nil, err
	}

	keys := make([]port.JWK, 0, len(models))
	for _, m := range models {
		block, _ := pem.Decode([]byte(m.PublicKeyPEM))
		if block == nil {
			continue
		}

		pub, err := x509.ParsePKIXPublicKey(block.Bytes)
		if err != nil {
			continue
		}

		rsaPub, ok := pub.(*rsa.PublicKey)
		if !ok {
			continue
		}

		keys = append(keys, port.JWK{
			Kty: "RSA",
			Use: "sig",
			Kid: m.Kid,
			Alg: "RS256",
			N:   base64.RawURLEncoding.EncodeToString(rsaPub.N.Bytes()),
			E:   base64.RawURLEncoding.EncodeToString(big.NewInt(int64(rsaPub.E)).Bytes()),
		})
	}

	return &port.JWKSResponse{Keys: keys}, nil
}
