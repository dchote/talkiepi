package rest

import (
	"io/fs"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"
)

func TestSPAServesIndexAtRoot(t *testing.T) {
	fsys := fstest.MapFS{
		"index.html":    &fstest.MapFile{Data: []byte("<html><head></head>ok</html>")},
		"assets/app.js": &fstest.MapFile{Data: []byte("console.log(1)")},
	}
	h := spaHandler(fsys)

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/", nil))
	if rr.Code != 200 {
		t.Fatalf("code=%d", rr.Code)
	}
	got := rr.Body.String()
	if !strings.Contains(got, "ok") {
		t.Fatalf("body=%q", got)
	}
	// Root should not inject <base> (preserves HA ingress relative assets).
	if strings.Contains(got, "<base ") {
		t.Fatalf("unexpected base tag at root: %q", got)
	}
	if ct := rr.Header().Get("Content-Type"); ct != "text/html; charset=utf-8" {
		t.Fatalf("content-type=%q", ct)
	}
}

func TestSPAServesAssetAndFallsBack(t *testing.T) {
	fsys := fstest.MapFS{
		"index.html":    &fstest.MapFile{Data: []byte("<html><head></head>index</html>")},
		"assets/app.js": &fstest.MapFile{Data: []byte("js")},
	}
	h := spaHandler(fs.FS(fsys))

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/assets/app.js", nil))
	if rr.Code != 200 || rr.Body.String() != "js" {
		t.Fatalf("asset: %d %q", rr.Code, rr.Body.String())
	}

	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/channels", nil))
	if rr.Code != 200 {
		t.Fatalf("fallback code=%d", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), "index") {
		t.Fatalf("fallback body=%q", rr.Body.String())
	}
}

func TestSPADeepLinkInjectsBaseHref(t *testing.T) {
	fsys := fstest.MapFS{
		"index.html": &fstest.MapFile{Data: []byte("<html><head><title>t</title></head></html>")},
	}
	h := spaHandler(fsys)

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/settings/server", nil))
	if rr.Code != 200 {
		t.Fatalf("code=%d", rr.Code)
	}
	got := rr.Body.String()
	if !strings.Contains(got, `<base href="/">`) {
		t.Fatalf("missing base href: %q", got)
	}
}

func TestSPAIngressBaseHref(t *testing.T) {
	fsys := fstest.MapFS{
		"index.html": &fstest.MapFile{Data: []byte("<html><head></head></html>")},
	}
	h := spaHandler(fsys)

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/api/hassio_ingress/token123/settings/server", nil))
	if rr.Code != 200 {
		t.Fatalf("code=%d", rr.Code)
	}
	got := rr.Body.String()
	if !strings.Contains(got, `<base href="/api/hassio_ingress/token123/">`) {
		t.Fatalf("missing ingress base: %q", got)
	}
}

func TestSPAMissingAssetIs404(t *testing.T) {
	fsys := fstest.MapFS{
		"index.html": &fstest.MapFile{Data: []byte("<html><head></head></html>")},
	}
	h := spaHandler(fsys)

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/assets/missing.js", nil))
	if rr.Code != 404 {
		t.Fatalf("code=%d want 404", rr.Code)
	}
}

func TestAppBaseHref(t *testing.T) {
	if got := appBaseHref("/settings/server"); got != "/" {
		t.Fatalf("got %q", got)
	}
	if got := appBaseHref("/api/hassio_ingress/abc/channels"); got != "/api/hassio_ingress/abc/" {
		t.Fatalf("got %q", got)
	}
}
