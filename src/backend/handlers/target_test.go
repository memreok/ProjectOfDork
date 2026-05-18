package handlers

import (
	"net/http/httptest"
	"os"
	"testing"
)

func TestNormalizeTarget(t *testing.T) {
	tests := map[string]string{
		"Example.com":                "example.com",
		" HTTPS://Example.com/path ": "example.com",
		"//Sub.Example.COM?query=1":  "sub.example.com",
		"*.EXAMPLE":                  "*.example",
		"*.Sub.Example.":             "*.sub.example",
	}

	for input, want := range tests {
		t.Run(input, func(t *testing.T) {
			if got := normalizeTarget(input); got != want {
				t.Fatalf("normalizeTarget(%q) = %q, want %q", input, got, want)
			}
		})
	}
}

func TestIsValidTarget(t *testing.T) {
	valid := []string{"example.com", "sub.example.com", "*.example", "*.sub.example"}
	for _, target := range valid {
		t.Run(target, func(t *testing.T) {
			if !isValidTarget(target) {
				t.Fatalf("isValidTarget(%q) = false, want true", target)
			}
		})
	}

	invalid := []string{"example", "*example.com", "*.", ".example", "https://example.com"}
	for _, target := range invalid {
		t.Run(target, func(t *testing.T) {
			if isValidTarget(target) {
				t.Fatalf("isValidTarget(%q) = true, want false", target)
			}
		})
	}
}

func TestCanManageHistory(t *testing.T) {
	t.Setenv("ADMIN_TOKEN", "secret-token")

	allowedReq := httptest.NewRequest("POST", "/", nil)
	allowedReq.Header.Set("X-Admin-Token", "secret-token")
	if !canManageHistory(allowedReq) {
		t.Fatal("canManageHistory() = false, want true")
	}

	blockedReq := httptest.NewRequest("POST", "/", nil)
	blockedReq.Header.Set("X-Admin-Token", "wrong-token")
	if canManageHistory(blockedReq) {
		t.Fatal("canManageHistory() = true, want false")
	}

	os.Unsetenv("ADMIN_TOKEN")
	if canManageHistory(allowedReq) {
		t.Fatal("canManageHistory() with empty ADMIN_TOKEN = true, want false")
	}
}
