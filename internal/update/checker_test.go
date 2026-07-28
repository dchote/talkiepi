package update

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestParseSemVer(t *testing.T) {
	cases := []struct {
		in      string
		want    SemVer
		wantErr bool
	}{
		{"1.2.3", SemVer{1, 2, 3}, false},
		{"v1.2.3", SemVer{1, 2, 3}, false},
		{"V2.0.0", SemVer{2, 0, 0}, false},
		{"1.2", SemVer{1, 2, 0}, false},
		{"3", SemVer{3, 0, 0}, false},
		{"v1.2.3-rc.1", SemVer{1, 2, 3}, false},
		{"1.0.0+build", SemVer{1, 0, 0}, false},
		{"", SemVer{}, true},
		{"abc", SemVer{}, true},
	}
	for _, tc := range cases {
		got, err := ParseSemVer(tc.in)
		if tc.wantErr {
			if err == nil {
				t.Errorf("ParseSemVer(%q) expected error", tc.in)
			}
			continue
		}
		if err != nil {
			t.Errorf("ParseSemVer(%q) unexpected error: %v", tc.in, err)
			continue
		}
		if got != tc.want {
			t.Errorf("ParseSemVer(%q) = %+v, want %+v", tc.in, got, tc.want)
		}
	}
}

func TestIsNewer(t *testing.T) {
	cases := []struct {
		latest, current string
		want            bool
	}{
		{"v1.2.0", "v1.1.9", true},
		{"1.2.0", "1.2.0", false},
		{"v1.0.0", "v1.0.1", false},
		{"v2.0.0", "1.9.9", true},
		{"v1.10.0", "v1.9.0", true},
	}
	for _, tc := range cases {
		if got := IsNewer(tc.latest, tc.current); got != tc.want {
			t.Errorf("IsNewer(%q, %q) = %v, want %v", tc.latest, tc.current, got, tc.want)
		}
	}
}

func TestCompare(t *testing.T) {
	if Compare(SemVer{1, 0, 0}, SemVer{1, 0, 1}) != -1 {
		t.Fatal("expected -1")
	}
	if Compare(SemVer{1, 1, 0}, SemVer{1, 0, 9}) != 1 {
		t.Fatal("expected 1")
	}
	if Compare(SemVer{1, 2, 3}, SemVer{1, 2, 3}) != 0 {
		t.Fatal("expected 0")
	}
}

func TestCheckAvailable(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/repos/dchote/talkiepi/releases/latest" {
			http.NotFound(w, r)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]string{
			"tag_name": "v1.2.0",
			"html_url": "https://github.com/dchote/talkiepi/releases/tag/v1.2.0",
		})
	}))
	defer srv.Close()

	dir := t.TempDir()
	state := filepath.Join(dir, "update.json")
	now := time.Date(2026, 7, 28, 12, 0, 0, 0, time.UTC)

	c := NewChecker("dchote", "talkiepi", "v1.0.0", state)
	c.HTTPClient = srv.Client()
	c.BaseURL = srv.URL
	c.Now = func() time.Time { return now }

	info, err := c.Check(context.Background())
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if !info.Available {
		t.Fatal("expected Available")
	}
	if info.Latest != "v1.2.0" {
		t.Fatalf("Latest = %q", info.Latest)
	}
	if info.Current != "v1.0.0" {
		t.Fatalf("Current = %q", info.Current)
	}
	if info.URL == "" {
		t.Fatal("expected URL")
	}
	if !info.CheckedAt.Equal(now) {
		t.Fatalf("CheckedAt = %v, want %v", info.CheckedAt, now)
	}

	cached, err := c.LoadCached()
	if err != nil {
		t.Fatalf("LoadCached: %v", err)
	}
	if cached.Latest != info.Latest || !cached.Available {
		t.Fatalf("cached mismatch: %+v", cached)
	}
	raw, err := os.ReadFile(state)
	if err != nil {
		t.Fatal(err)
	}
	if len(raw) == 0 {
		t.Fatal("empty state file")
	}
}

func TestCheckUpToDate(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]string{
			"tag_name": "v1.0.0",
			"html_url": "https://example.com/r",
		})
	}))
	defer srv.Close()

	c := NewChecker("o", "r", "1.0.0", filepath.Join(t.TempDir(), "s.json"))
	c.HTTPClient = srv.Client()
	c.BaseURL = srv.URL

	info, err := c.Check(context.Background())
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if info.Available {
		t.Fatal("expected not Available")
	}
}

func TestCheckHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "nope", http.StatusNotFound)
	}))
	defer srv.Close()

	c := NewChecker("o", "r", "1.0.0", "")
	c.HTTPClient = srv.Client()
	c.BaseURL = srv.URL

	_, err := c.Check(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestLoadCachedMissing(t *testing.T) {
	c := NewChecker("o", "r", "1.0.0", filepath.Join(t.TempDir(), "missing.json"))
	_, err := c.LoadCached()
	if err == nil {
		t.Fatal("expected error")
	}
}
