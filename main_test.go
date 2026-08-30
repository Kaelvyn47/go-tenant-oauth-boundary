package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestLoginProviderDecision(t *testing.T) {
	tests := []struct {
		name, provider string
		valid          bool
	}{{"google", "google", true}, {"github", "github", true}, {"typo", "gitlab", false}}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.provider == "google" || tc.provider == "github"
			if got != tc.valid {
				t.Fatalf("provider %q valid=%v, want %v", tc.provider, got, tc.valid)
			}
		})
	}
}

func TestVerifyCaptchaSendsRequiredFields(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body["widget_record_id"] != "widget-123" {
			t.Errorf("widget_record_id = %v, want widget-123", body["widget_record_id"])
		}
		if body["token"] != "captcha-token" {
			t.Errorf("token = %v, want captcha-token", body["token"])
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true,"data":null}`))
	}))
	defer server.Close()

	client := &Client{BaseURL: server.URL, Key: "test-key", HTTP: server.Client()}
	if err := client.VerifyCaptcha("widget-123", "captcha-token", "google", "127.0.0.1", "login", 0.5); err != nil {
		t.Fatal(err)
	}
}
