package model

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestUserJSONNeverExposesPasswordHash(t *testing.T) {
	encoded, err := json.Marshal(User{PasswordHash: strings.Repeat("x", 12)})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(encoded), "PasswordHash") || strings.Contains(string(encoded), "password_hash") {
		t.Fatalf("password hash exposed in JSON: %s", encoded)
	}
}
