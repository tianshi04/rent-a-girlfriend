package client

import (
	"crypto/rsa"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/url"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/rent-a-girlfriend/identity-service/internal/application/port"
)

// GoogleOAuthClient implements GoogleOAuthProvider with PKCE support.
type GoogleOAuthClient struct {
	clientID     string
	clientSecret string
	redirectURI  string
}

// NewGoogleOAuthClient creates a new Google OAuth client.
func NewGoogleOAuthClient(clientID, clientSecret, redirectURI string) *GoogleOAuthClient {
	return &GoogleOAuthClient{
		clientID:     clientID,
		clientSecret: clientSecret,
		redirectURI:  redirectURI,
	}
}

func (c *GoogleOAuthClient) BuildAuthURL(state, codeChallenge string) string {
	params := url.Values{
		"client_id":             {c.clientID},
		"redirect_uri":         {c.redirectURI},
		"response_type":        {"code"},
		"scope":                {"openid email profile"},
		"state":                {state},
		"code_challenge":       {codeChallenge},
		"code_challenge_method": {"S256"},
		"access_type":          {"offline"},
	}
	return "https://accounts.google.com/o/oauth2/v2/auth?" + params.Encode()
}

func (c *GoogleOAuthClient) ExchangeCode(code, codeVerifier string) (*port.GoogleUserInfo, error) {
	// Exchange authorization code for tokens
	data := url.Values{
		"code":          {code},
		"client_id":    {c.clientID},
		"client_secret": {c.clientSecret},
		"redirect_uri": {c.redirectURI},
		"grant_type":   {"authorization_code"},
		"code_verifier": {codeVerifier},
	}

	resp, err := http.Post("https://oauth2.googleapis.com/token", "application/x-www-form-urlencoded", strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to exchange code: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("google token exchange failed: %s", string(body))
	}

	var tokenResp struct {
		AccessToken string `json:"access_token"`
		IDToken     string `json:"id_token"`
	}
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return nil, fmt.Errorf("failed to parse token response: %w", err)
	}

	// Verify ID Token and extract user info
	return c.verifyIDToken(tokenResp.IDToken)
}

func (c *GoogleOAuthClient) verifyIDToken(idTokenStr string) (*port.GoogleUserInfo, error) {
	// 1. Fetch Google's public keys
	resp, err := http.Get("https://www.googleapis.com/oauth2/v3/certs")
	if err != nil {
		return nil, fmt.Errorf("failed to fetch google certs: %w", err)
	}
	defer resp.Body.Close()

	var jwks struct {
		Keys []struct {
			Kid string `json:"kid"`
			N   string `json:"n"`
			E   string `json:"e"`
		} `json:"keys"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&jwks); err != nil {
		return nil, fmt.Errorf("failed to decode google certs: %w", err)
	}

	// 2. Parse and verify ID Token
	token, err := jwt.Parse(idTokenStr, func(token *jwt.Token) (interface{}, error) {
		// Verify algorithm
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}

		// Get kid from header
		kid, ok := token.Header["kid"].(string)
		if !ok {
			return nil, fmt.Errorf("kid header not found")
		}

		// Find matching key
		for _, key := range jwks.Keys {
			if key.Kid == kid {
				// Convert JWK to RSA Public Key
				nBytes, err := base64.RawURLEncoding.DecodeString(key.N)
				if err != nil {
					return nil, err
				}
				eBytes, err := base64.RawURLEncoding.DecodeString(key.E)
				if err != nil {
					return nil, err
				}

				// Standard exponent is usually 65537 (AQAB), which is 3 bytes.
				// But we should handle it properly.
				var e int
				if len(eBytes) < 4 {
					data := make([]byte, 4)
					copy(data[4-len(eBytes):], eBytes)
					e = int(binary.BigEndian.Uint32(data))
				} else {
					e = int(binary.BigEndian.Uint32(eBytes))
				}

				return &rsa.PublicKey{
					N: new(big.Int).SetBytes(nBytes),
					E: e,
				}, nil
			}
		}

		return nil, fmt.Errorf("kid %s not found in google certs", kid)
	})

	if err != nil {
		return nil, fmt.Errorf("failed to parse id token: %w", err)
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token claims")
	}

	// 3. Validate standard claims
	// Issuer
	iss, _ := claims["iss"].(string)
	if iss != "https://accounts.google.com" && iss != "accounts.google.com" {
		return nil, fmt.Errorf("invalid issuer: %s", iss)
	}

	// Audience
	aud, _ := claims["aud"].(string)
	if aud != c.clientID {
		return nil, fmt.Errorf("invalid audience: %s", aud)
	}

	// 4. Extract user info
	userInfo := &port.GoogleUserInfo{
		GoogleID: claims["sub"].(string),
		Email:    claims["email"].(string),
	}

	if name, ok := claims["name"].(string); ok {
		userInfo.Name = name
	}
	if picture, ok := claims["picture"].(string); ok {
		userInfo.Picture = picture
	}

	return userInfo, nil
}
