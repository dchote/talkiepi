package rest

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dchote/talkiepi/internal/audio"
	"github.com/dchote/talkiepi/internal/config"
	"github.com/dchote/talkiepi/internal/controller"
	"github.com/dchote/talkiepi/internal/discovery"
)

type fakeEnum struct{}

func (fakeEnum) Backends() []string { return []string{"alsa", "pulseaudio"} }
func (fakeEnum) Devices(backend string, dir audio.Direction) ([]audio.DeviceInfo, error) {
	return []audio.DeviceInfo{{
		Descriptor: audio.Descriptor{Backend: "alsa", DeviceID: "01", DeviceName: "Test"},
		Name:       "Test",
		IsDefault:  true,
	}}, nil
}

func testController(t *testing.T) *controller.Controller {
	t.Helper()
	cfg := config.Defaults()
	cfg.SetPath(filepath.Join(t.TempDir(), "cfg.yaml"))
	return controller.New(controller.Options{
		Config:  cfg,
		Version: "test",
		Browser: discovery.StaticBrowser{},
	})
}

func TestDiscoverServers(t *testing.T) {
	cfg := config.Defaults()
	cfg.SetPath(filepath.Join(t.TempDir(), "cfg.yaml"))
	ctrl := controller.New(controller.Options{
		Config:  cfg,
		Version: "test",
		Browser: discovery.StaticBrowser{Services: []discovery.Service{{
			Name: "go-mumble-server", Host: "mumble.local", Port: 64738,
			Addresses: []string{"192.168.1.50"},
		}}},
	})
	h := NewRouter(Options{Controller: ctrl, Version: "test"})
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/api/v1/discovery/servers", nil))
	if rr.Code != 200 {
		t.Fatal(rr.Body.String())
	}
	var body struct {
		Servers []discovery.Service `json:"servers"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if len(body.Servers) != 1 || body.Servers[0].Name != "go-mumble-server" {
		t.Fatalf("got %+v", body.Servers)
	}
}

func TestHealth(t *testing.T) {
	h := NewRouter(Options{Controller: testController(t), Version: "1.2.3"})
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/health", nil))
	if rr.Code != 200 {
		t.Fatalf("code=%d", rr.Code)
	}
}

func TestDocsSwaggerUI(t *testing.T) {
	h := NewRouter(Options{Controller: testController(t), Version: "test"})
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/docs", nil))
	if rr.Code != 200 {
		t.Fatalf("code=%d", rr.Code)
	}
	if ct := rr.Header().Get("Content-Type"); !strings.Contains(ct, "text/html") {
		t.Fatalf("content-type=%q", ct)
	}
	if !strings.Contains(rr.Body.String(), "SwaggerUIBundle") {
		t.Fatal("expected swagger UI html")
	}
	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/api/v1/openapi.yaml", nil))
	if rr.Code != 200 || !strings.Contains(rr.Body.String(), "openapi:") {
		t.Fatalf("openapi: code=%d body=%s", rr.Code, rr.Body.String()[:min(80, rr.Body.Len())])
	}
}

func TestGetConfigAndPatch(t *testing.T) {
	h := NewRouter(Options{Controller: testController(t), Version: "test"})
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/api/v1/config", nil))
	if rr.Code != 200 {
		t.Fatal(rr.Body.String())
	}

	rr = httptest.NewRecorder()
	body := bytes.NewBufferString("voice:\n  mode: push_to_talk\n")
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/config", body)
	h.ServeHTTP(rr, req)
	if rr.Code != 200 {
		t.Fatal(rr.Body.String())
	}
	var cfg config.Config
	if err := json.Unmarshal(rr.Body.Bytes(), &cfg); err != nil {
		t.Fatal(err)
	}
	if cfg.Voice.Mode != "push_to_talk" {
		t.Fatalf("mode=%s", cfg.Voice.Mode)
	}
}

func TestStatus(t *testing.T) {
	h := NewRouter(Options{Controller: testController(t), Version: "test"})
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/api/v1/status", nil))
	if rr.Code != 200 {
		t.Fatal(rr.Code)
	}
}

func TestAudioDevices(t *testing.T) {
	h := NewRouter(Options{Controller: testController(t), Enumerator: fakeEnum{}})
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/api/v1/audio/devices", nil))
	if rr.Code != 200 {
		t.Fatal(rr.Body.String())
	}
	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/api/v1/audio/backends", nil))
	if rr.Code != 200 {
		t.Fatal(rr.Body.String())
	}
}

func TestPTTRequiresStart(t *testing.T) {
	h := NewRouter(Options{Controller: testController(t)})
	rr := httptest.NewRecorder()
	body := bytes.NewBufferString(`{"pressed":true}`)
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodPost, "/api/v1/ptt", body))
	// Manual source not started → conflict
	if rr.Code != http.StatusConflict {
		t.Fatalf("code=%d body=%s", rr.Code, rr.Body.String())
	}
}
