package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/golang-jwt/jwt/v5"
)

func TestHealthNoAuth(t *testing.T) {
	secret := []byte("test-secret")
	apiKey := "test-api-key"
	app := newApp(secret, apiKey, "")

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test error: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
}

func TestQRRequiresAuth(t *testing.T) {
	secret := []byte("test-secret")
	apiKey := "test-api-key"
	app := newApp(secret, apiKey, "")

	req := httptest.NewRequest(http.MethodPost, "/qr", strings.NewReader(`{"matrix":[[1],[2]]}`))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test error: %v", err)
	}
	if resp.StatusCode != 401 {
		t.Fatalf("expected 401, got %d", resp.StatusCode)
	}
}

func TestTokenRequiresApiKey(t *testing.T) {
	secret := []byte("test-secret")
	apiKey := "test-api-key"
	app := newApp(secret, apiKey, "")

	req := httptest.NewRequest(http.MethodPost, "/token", strings.NewReader(`{"sub":"julio"}`))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test error: %v", err)
	}
	if resp.StatusCode != 401 {
		t.Fatalf("expected 401, got %d", resp.StatusCode)
	}
}

func TestQRCallsNodeStatsWithServiceToken(t *testing.T) {
	secret := []byte("test-secret")
	apiKey := "test-api-key"

	nodeServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/stats" {
			http.NotFound(w, r)
			return
		}

		auth := r.Header.Get("Authorization")
		if !strings.HasPrefix(strings.ToLower(auth), "bearer ") {
			w.WriteHeader(401)
			_ = json.NewEncoder(w).Encode(map[string]any{"error": "missing bearer"})
			return
		}
		tokenStr := strings.TrimSpace(auth[7:])

		parsed, err := jwt.Parse(tokenStr, func(t *jwt.Token) (any, error) {
			return secret, nil
		}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))
		if err != nil || !parsed.Valid {
			w.WriteHeader(401)
			_ = json.NewEncoder(w).Encode(map[string]any{"error": "invalid jwt"})
			return
		}

		claims, _ := parsed.Claims.(jwt.MapClaims)
		if claims["sub"] != "api-go" {
			w.WriteHeader(401)
			_ = json.NewEncoder(w).Encode(map[string]any{"error": "sub inválido"})
			return
		}

		_ = json.NewEncoder(w).Encode(map[string]any{
			"max":         1.0,
			"min":         0.0,
			"sum":         10.0,
			"avg":         1.25,
			"isDiagonalQ": false,
			"isDiagonalR": false,
		})
	}))
	defer nodeServer.Close()

	app := newApp(secret, apiKey, nodeServer.URL)

	userToken, err := issueToken("tester", secret)
	if err != nil {
		t.Fatalf("issueToken error: %v", err)
	}

	body := []byte(`{"matrix":[[1,2],[3,4],[5,6]]}`)
	req := httptest.NewRequest(http.MethodPost, "/qr", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+userToken)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test error: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var parsed map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if parsed["stats"] == nil {
		t.Fatalf("expected stats in response")
	}
}

func TestQRRejectsWideMatrix(t *testing.T) {
	secret := []byte("test-secret")
	apiKey := "test-api-key"
	app := newApp(secret, apiKey, "")

	userToken, err := issueToken("tester", secret)
	if err != nil {
		t.Fatalf("issueToken error: %v", err)
	}

	body := []byte(`{"matrix":[[1,2,3],[4,5,6]]}`)
	req := httptest.NewRequest(http.MethodPost, "/qr", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+userToken)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test error: %v", err)
	}
	if resp.StatusCode != 400 {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}
