//go:build production

package bootstrap

import gateway "github.com/rent-a-girlfriend/identity-service/internal/interfaces/http"

func (s *Server) getTestGatewayOptions() []gateway.GatewayOption {
	return nil
}
