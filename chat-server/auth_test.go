package main

import "testing"

func TestGenerateAndParseToken(t *testing.T) {
	user := &User{
		UserID:   42,
		Username: "alice",
	}

	token, err := GenerateToken(user)
	if err != nil {
		t.Fatalf("GenerateToken() error = %v", err)
	}
	if token == "" {
		t.Fatal("GenerateToken() returned empty token")
	}

	claims, err := ParseToken(token)
	if err != nil {
		t.Fatalf("ParseToken() error = %v", err)
	}
	if claims.UserID != user.UserID {
		t.Fatalf("claims.UserID = %d, want %d", claims.UserID, user.UserID)
	}
	if claims.Username != user.Username {
		t.Fatalf("claims.Username = %q, want %q", claims.Username, user.Username)
	}
}

func TestParseTokenRejectsInvalidToken(t *testing.T) {
	if _, err := ParseToken("not-a-jwt"); err == nil {
		t.Fatal("ParseToken() error = nil, want non-nil")
	}
}
