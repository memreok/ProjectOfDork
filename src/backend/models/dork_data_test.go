package models

import (
	"strings"
	"testing"
)

func TestSiteQueryPrefixesDomain(t *testing.T) {
	got := SiteQuery("example.com", `intitle:"index of"`)
	want := `site:example.com intitle:"index of"`

	if got != want {
		t.Fatalf("SiteQuery() = %q, want %q", got, want)
	}
}

func TestSiteQueryReplacesDomainToken(t *testing.T) {
	got := SiteQuery("example.com", `site:s3.amazonaws.com "{domain}"`)
	want := `site:s3.amazonaws.com "example.com"`

	if got != want {
		t.Fatalf("SiteQuery() = %q, want %q", got, want)
	}
}

func TestBuildCustomDorkUsesDefaultExpression(t *testing.T) {
	got := BuildCustomDork("example.com", "")

	if got.Title != "Özel Sorgu" {
		t.Fatalf("Title = %q, want %q", got.Title, "Özel Sorgu")
	}
	if !strings.Contains(got.Query, DefaultCustomDork) {
		t.Fatalf("Query = %q, want it to contain default dork expression", got.Query)
	}
	if !strings.Contains(got.URL, "https://www.google.com/search?q=") {
		t.Fatalf("URL = %q, want Google search URL", got.URL)
	}
}

func TestDorksByCategoryReturnsCopyForEmptyCategory(t *testing.T) {
	got := DorksByCategory("")

	if len(got) != len(DorkLibrary) {
		t.Fatalf("len(DorksByCategory(\"\")) = %d, want %d", len(got), len(DorkLibrary))
	}

	got[0].Title = "changed"
	if DorkLibrary[0].Title == "changed" {
		t.Fatal("DorksByCategory returned a slice that mutates DorkLibrary")
	}
}
