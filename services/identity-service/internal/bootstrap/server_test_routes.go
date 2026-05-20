//go:build !production

package bootstrap

import (
	"encoding/json"
	"log"
	"net/http"
	"os"

	"github.com/rent-a-girlfriend/identity-service/internal/application/command"
	gateway "github.com/rent-a-girlfriend/identity-service/internal/interfaces/http"
)

func (s *Server) getTestGatewayOptions() []gateway.GatewayOption {
	if os.Getenv("ENABLE_TEST_ROUTES") != "true" {
		return nil
	}

	log.Println("[BOOTSTRAP] Registering E2E test-only endpoints")

	var gatewayOpts []gateway.GatewayOption

	// Mock Login Handler
	gatewayOpts = append(gatewayOpts, gateway.WithAdditionalHandler("/api/v1/test/mock-login", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		var req struct {
			Email    string `json:"email"`
			GoogleID string `json:"google_id"`
			Role     string `json:"role"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
			return
		}

		tokenPair, err := s.mockLoginHandler.Handle(r.Context(), command.MockLoginCommand{
			Email:    req.Email,
			GoogleID: req.GoogleID,
			Role:     req.Role,
		})
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"accessToken":  tokenPair.AccessToken,
			"refreshToken": tokenPair.RefreshToken,
			"expiresIn":    tokenPair.ExpiresIn,
		})
	})))

	// Truncate Handler
	gatewayOpts = append(gatewayOpts, gateway.WithAdditionalHandler("/api/v1/test/truncate", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})))

	return gatewayOpts
}
