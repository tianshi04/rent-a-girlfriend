package http

import "net/http"

// GatewayOption defines a function signature to customize the HTTP Gateway router.
type GatewayOption func(mux *http.ServeMux)

// WithAdditionalHandler allows registering custom HTTP endpoints on the gateway multiplexer.
func WithAdditionalHandler(pattern string, handler http.Handler) GatewayOption {
	return func(mux *http.ServeMux) {
		mux.Handle(pattern, handler)
	}
}
