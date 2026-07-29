package service

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestGoogleIDTokenVerifierAcceptsGoogleClaimsForConfiguredAudience(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	server := newGoogleJWKSFixture(t, key, "key-1")
	defer server.Close()
	verifier := newGoogleIDTokenVerifier([]string{"web-client-id"}, server.URL, server.Client())
	token := signedGoogleToken(t, key, "key-1", map[string]any{
		"iss": "https://accounts.google.com", "aud": "web-client-id", "sub": "subject-1",
		"email": "person@example.com", "email_verified": true, "name": "Person", "exp": time.Now().Add(time.Minute).Unix(),
	})

	identity, err := verifier.Verify(t.Context(), token)
	if err != nil {
		t.Fatal(err)
	}
	if identity.Subject != "subject-1" || identity.Email != "person@example.com" || !identity.EmailVerified {
		t.Fatalf("unexpected identity: %#v", identity)
	}
}

func TestGoogleIDTokenVerifierRejectsWrongAudience(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	server := newGoogleJWKSFixture(t, key, "key-1")
	defer server.Close()
	verifier := newGoogleIDTokenVerifier([]string{"web-client-id"}, server.URL, server.Client())
	token := signedGoogleToken(t, key, "key-1", map[string]any{
		"iss": "accounts.google.com", "aud": "other-client", "sub": "subject-1",
		"email": "person@example.com", "email_verified": true, "exp": time.Now().Add(time.Minute).Unix(),
	})

	if _, err := verifier.Verify(t.Context(), token); err == nil {
		t.Fatal("expected wrong audience to be rejected")
	}
}

func newGoogleJWKSFixture(t *testing.T, key *rsa.PrivateKey, kid string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"keys": []map[string]string{{
			"kid": kid, "kty": "RSA", "alg": "RS256", "use": "sig",
			"n": base64.RawURLEncoding.EncodeToString(key.PublicKey.N.Bytes()),
			"e": base64.RawURLEncoding.EncodeToString(big.NewInt(int64(key.PublicKey.E)).Bytes()),
		}}})
	}))
}

func signedGoogleToken(t *testing.T, key *rsa.PrivateKey, kid string, claims map[string]any) string {
	t.Helper()
	encode := func(v any) string {
		body, err := json.Marshal(v)
		if err != nil {
			t.Fatal(err)
		}
		return base64.RawURLEncoding.EncodeToString(body)
	}
	signingInput := encode(map[string]string{"alg": "RS256", "kid": kid, "typ": "JWT"}) + "." + encode(claims)
	digest := sha256.Sum256([]byte(signingInput))
	signature, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, digest[:])
	if err != nil {
		t.Fatal(err)
	}
	return signingInput + "." + base64.RawURLEncoding.EncodeToString(signature)
}
