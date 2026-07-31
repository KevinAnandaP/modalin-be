package service

import (
	"context"
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"strings"
	"sync"
	"time"
)

const googleJWKSURL = "https://www.googleapis.com/oauth2/v3/certs"

// GoogleIDTokenVerifier validates Google-issued RS256 ID tokens locally using
// Google's public JWKS. It does not trust user-supplied profile fields.
type GoogleIDTokenVerifier struct {
	audiences map[string]struct{}
	jwksURL   string
	client    *http.Client
	now       func() time.Time

	mu      sync.RWMutex
	keys    map[string]*rsa.PublicKey
	expires time.Time
}

func NewGoogleIDTokenVerifier(clientIDs []string) *GoogleIDTokenVerifier {
	return newGoogleIDTokenVerifier(clientIDs, googleJWKSURL, &http.Client{Timeout: 5 * time.Second})
}

func newGoogleIDTokenVerifier(clientIDs []string, jwksURL string, client *http.Client) *GoogleIDTokenVerifier {
	audiences := make(map[string]struct{}, len(clientIDs))
	for _, clientID := range clientIDs {
		if normalized := strings.TrimSpace(clientID); normalized != "" {
			audiences[normalized] = struct{}{}
		}
	}
	if client == nil {
		client = &http.Client{Timeout: 5 * time.Second}
	}
	return &GoogleIDTokenVerifier{audiences: audiences, jwksURL: jwksURL, client: client, now: time.Now, keys: make(map[string]*rsa.PublicKey)}
}

func (v *GoogleIDTokenVerifier) Verify(ctx context.Context, rawIDToken string) (GoogleIdentity, error) {
	if len(v.audiences) == 0 {
		return GoogleIdentity{}, errors.New("Google OAuth client IDs are not configured")
	}
	parts := strings.Split(rawIDToken, ".")
	if len(parts) != 3 {
		return GoogleIdentity{}, errors.New("invalid ID token format")
	}
	headerBytes, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return GoogleIdentity{}, errors.New("invalid ID token header")
	}
	var header struct {
		Algorithm string `json:"alg"`
		KeyID     string `json:"kid"`
	}
	if err := json.Unmarshal(headerBytes, &header); err != nil || header.Algorithm != "RS256" || header.KeyID == "" {
		return GoogleIdentity{}, errors.New("unsupported ID token signature")
	}
	key, err := v.keyFor(ctx, header.KeyID)
	if err != nil {
		return GoogleIdentity{}, err
	}
	signature, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return GoogleIdentity{}, errors.New("invalid ID token signature")
	}
	digest := sha256.Sum256([]byte(parts[0] + "." + parts[1]))
	if err := rsa.VerifyPKCS1v15(key, crypto.SHA256, digest[:], signature); err != nil {
		return GoogleIdentity{}, errors.New("invalid ID token signature")
	}
	payloadBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return GoogleIdentity{}, errors.New("invalid ID token payload")
	}
	var claims struct {
		Issuer        string          `json:"iss"`
		Audience      json.RawMessage `json:"aud"`
		Subject       string          `json:"sub"`
		Email         string          `json:"email"`
		Name          string          `json:"name"`
		EmailVerified bool            `json:"email_verified"`
		ExpiresAt     int64           `json:"exp"`
	}
	if err := json.Unmarshal(payloadBytes, &claims); err != nil {
		return GoogleIdentity{}, errors.New("invalid ID token claims")
	}
	if claims.Issuer != "accounts.google.com" && claims.Issuer != "https://accounts.google.com" {
		return GoogleIdentity{}, errors.New("invalid ID token issuer")
	}
	if claims.ExpiresAt <= v.now().UTC().Unix() || !v.hasAudience(claims.Audience) {
		return GoogleIdentity{}, errors.New("expired or invalid ID token audience")
	}
	if strings.TrimSpace(claims.Subject) == "" || strings.TrimSpace(claims.Email) == "" || !claims.EmailVerified {
		return GoogleIdentity{}, errors.New("unverified Google identity")
	}
	return GoogleIdentity{Subject: claims.Subject, Email: claims.Email, FullName: claims.Name, EmailVerified: true}, nil
}

func (v *GoogleIDTokenVerifier) hasAudience(raw json.RawMessage) bool {
	var audience string
	if json.Unmarshal(raw, &audience) == nil {
		_, ok := v.audiences[audience]
		return ok
	}
	var audiences []string
	if json.Unmarshal(raw, &audiences) != nil {
		return false
	}
	for _, audience := range audiences {
		if _, ok := v.audiences[audience]; ok {
			return true
		}
	}
	return false
}

func (v *GoogleIDTokenVerifier) keyFor(ctx context.Context, keyID string) (*rsa.PublicKey, error) {
	v.mu.RLock()
	key, valid := v.keys[keyID], v.now().Before(v.expires)
	v.mu.RUnlock()
	if key != nil && valid {
		return key, nil
	}
	if err := v.refreshKeys(ctx); err != nil {
		return nil, err
	}
	v.mu.RLock()
	key = v.keys[keyID]
	v.mu.RUnlock()
	if key == nil {
		return nil, errors.New("Google signing key not found")
	}
	return key, nil
}

func (v *GoogleIDTokenVerifier) refreshKeys(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, v.jwksURL, nil)
	if err != nil {
		return err
	}
	response, err := v.client.Do(req)
	if err != nil {
		return fmt.Errorf("fetch Google signing keys: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("fetch Google signing keys: unexpected status %d", response.StatusCode)
	}
	var payload struct {
		Keys []struct {
			KeyID string `json:"kid"`
			Type  string `json:"kty"`
			Use   string `json:"use"`
			Alg   string `json:"alg"`
			N     string `json:"n"`
			E     string `json:"e"`
		} `json:"keys"`
	}
	if err := json.NewDecoder(io.LimitReader(response.Body, 1<<20)).Decode(&payload); err != nil {
		return fmt.Errorf("decode Google signing keys: %w", err)
	}
	keys := make(map[string]*rsa.PublicKey, len(payload.Keys))
	for _, jwk := range payload.Keys {
		if jwk.Type != "RSA" || jwk.Use != "sig" || jwk.Alg != "RS256" || jwk.KeyID == "" {
			continue
		}
		key, err := parseGoogleRSAPublicKey(jwk.N, jwk.E)
		if err == nil {
			keys[jwk.KeyID] = key
		}
	}
	if len(keys) == 0 {
		return errors.New("Google signing key response contains no valid keys")
	}
	v.mu.Lock()
	v.keys = keys
	v.expires = v.now().Add(time.Hour)
	v.mu.Unlock()
	return nil
}

func parseGoogleRSAPublicKey(modulus, exponent string) (*rsa.PublicKey, error) {
	n, err := base64.RawURLEncoding.DecodeString(modulus)
	if err != nil || len(n) == 0 {
		return nil, errors.New("invalid RSA modulus")
	}
	e, err := base64.RawURLEncoding.DecodeString(exponent)
	if err != nil || len(e) == 0 || len(e) > 4 {
		return nil, errors.New("invalid RSA exponent")
	}
	exponentValue := 0
	for _, value := range e {
		exponentValue = exponentValue<<8 | int(value)
	}
	if exponentValue < 3 || exponentValue%2 == 0 {
		return nil, errors.New("invalid RSA exponent")
	}
	return &rsa.PublicKey{N: new(big.Int).SetBytes(n), E: exponentValue}, nil
}
