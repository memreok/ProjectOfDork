package handlers

import "testing"

func TestNormalizeTarget(t *testing.T) {
	tests := map[string]string{
		"Tesla.com":                 "tesla.com",
		" HTTPS://Tesla.com/path ":  "tesla.com",
		"//Sub.Example.COM?query=1": "sub.example.com",
		"*.HK":                      "*.hk",
		"*.gov.tr.":                 "*.gov.tr",
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
	valid := []string{"tesla.com", "sub.example.com", "*.hk", "*.gov.tr"}
	for _, target := range valid {
		t.Run(target, func(t *testing.T) {
			if !isValidTarget(target) {
				t.Fatalf("isValidTarget(%q) = false, want true", target)
			}
		})
	}

	invalid := []string{"tesla", "*tesla.com", "*.", ".hk", "https://tesla.com"}
	for _, target := range invalid {
		t.Run(target, func(t *testing.T) {
			if isValidTarget(target) {
				t.Fatalf("isValidTarget(%q) = true, want false", target)
			}
		})
	}
}
