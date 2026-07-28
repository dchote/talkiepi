// Package update checks GitHub releases for newer versions.
package update

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// UpdateInfo is the result of a release check.
type UpdateInfo struct {
	Current   string    `json:"current"`
	Latest    string    `json:"latest"`
	URL       string    `json:"url"`
	Available bool      `json:"available"`
	CheckedAt time.Time `json:"checked_at"`
}

// Checker polls GitHub for the latest release.
type Checker struct {
	Owner      string
	Repo       string
	Current    string
	StateFile  string
	HTTPClient *http.Client
	Now        func() time.Time
	BaseURL    string // optional override for tests (default api.github.com)
}

// NewChecker creates a Checker for owner/repo comparing against currentVersion.
func NewChecker(owner, repo, currentVersion, stateFile string) *Checker {
	return &Checker{
		Owner:      owner,
		Repo:       repo,
		Current:    currentVersion,
		StateFile:  stateFile,
		HTTPClient: http.DefaultClient,
		Now:        time.Now,
		BaseURL:    "https://api.github.com",
	}
}

type githubRelease struct {
	TagName string `json:"tag_name"`
	HTMLURL string `json:"html_url"`
}

// Check fetches the latest release, compares SemVer, and caches the result.
func (c *Checker) Check(ctx context.Context) (UpdateInfo, error) {
	if c.HTTPClient == nil {
		c.HTTPClient = http.DefaultClient
	}
	if c.Now == nil {
		c.Now = time.Now
	}
	base := c.BaseURL
	if base == "" {
		base = "https://api.github.com"
	}

	url := fmt.Sprintf("%s/repos/%s/%s/releases/latest", base, c.Owner, c.Repo)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return UpdateInfo{}, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "talkiepi-update-checker")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return UpdateInfo{}, fmt.Errorf("update: fetch latest: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return UpdateInfo{}, fmt.Errorf("update: unexpected status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var rel githubRelease
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return UpdateInfo{}, fmt.Errorf("update: decode: %w", err)
	}

	info := UpdateInfo{
		Current:   c.Current,
		Latest:    rel.TagName,
		URL:       rel.HTMLURL,
		Available: IsNewer(rel.TagName, c.Current),
		CheckedAt: c.Now(),
	}

	if err := c.save(info); err != nil {
		return info, fmt.Errorf("update: cache: %w", err)
	}
	return info, nil
}

// LoadCached reads the last cached UpdateInfo, if any.
func (c *Checker) LoadCached() (UpdateInfo, error) {
	if c.StateFile == "" {
		return UpdateInfo{}, fmt.Errorf("update: no state file")
	}
	data, err := os.ReadFile(c.StateFile)
	if err != nil {
		return UpdateInfo{}, err
	}
	var info UpdateInfo
	if err := json.Unmarshal(data, &info); err != nil {
		return UpdateInfo{}, err
	}
	return info, nil
}

func (c *Checker) save(info UpdateInfo) error {
	if c.StateFile == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(c.StateFile), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(info, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(c.StateFile, data, 0o644)
}

// SemVer is a parsed X.Y.Z version (pre-release/build metadata ignored for compare).
type SemVer struct {
	Major, Minor, Patch int
}

// ParseSemVer strips an optional leading 'v' and parses X.Y.Z.
// Extra suffixes after Patch (e.g. -rc.1) are ignored for numeric compare.
func ParseSemVer(s string) (SemVer, error) {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "v")
	s = strings.TrimPrefix(s, "V")
	if s == "" {
		return SemVer{}, fmt.Errorf("empty version")
	}
	// Drop pre-release / build metadata for core compare.
	if i := strings.IndexAny(s, "-+"); i >= 0 {
		s = s[:i]
	}
	parts := strings.Split(s, ".")
	if len(parts) < 1 || len(parts) > 3 {
		return SemVer{}, fmt.Errorf("invalid semver %q", s)
	}
	var v SemVer
	var err error
	v.Major, err = strconv.Atoi(parts[0])
	if err != nil {
		return SemVer{}, fmt.Errorf("invalid major in %q", s)
	}
	if len(parts) > 1 {
		v.Minor, err = strconv.Atoi(parts[1])
		if err != nil {
			return SemVer{}, fmt.Errorf("invalid minor in %q", s)
		}
	}
	if len(parts) > 2 {
		v.Patch, err = strconv.Atoi(parts[2])
		if err != nil {
			return SemVer{}, fmt.Errorf("invalid patch in %q", s)
		}
	}
	return v, nil
}

// Compare returns -1 if a<b, 0 if equal, 1 if a>b.
func Compare(a, b SemVer) int {
	if a.Major != b.Major {
		if a.Major < b.Major {
			return -1
		}
		return 1
	}
	if a.Minor != b.Minor {
		if a.Minor < b.Minor {
			return -1
		}
		return 1
	}
	if a.Patch != b.Patch {
		if a.Patch < b.Patch {
			return -1
		}
		return 1
	}
	return 0
}

// IsNewer reports whether latest is a higher SemVer than current.
func IsNewer(latest, current string) bool {
	l, errL := ParseSemVer(latest)
	c, errC := ParseSemVer(current)
	if errL != nil || errC != nil {
		// Fall back to string compare of normalized tags if parse fails.
		return normalizeTag(latest) != normalizeTag(current) && normalizeTag(latest) > normalizeTag(current)
	}
	return Compare(l, c) > 0
}

func normalizeTag(s string) string {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "v")
	s = strings.TrimPrefix(s, "V")
	return s
}
