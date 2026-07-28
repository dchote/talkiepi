package rest

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"net/http"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	apispec "github.com/dchote/talkiepi/api"
	"github.com/dchote/talkiepi/internal/audio"
	"github.com/dchote/talkiepi/internal/controller"
	"github.com/dchote/talkiepi/internal/discovery"
	"github.com/dchote/talkiepi/internal/update"
)

// Options configures the HTTP server.
type Options struct {
	Controller *controller.Controller
	Frontend   fs.FS // nil = API only
	Version    string
	Enumerator audio.Enumerator // optional; nil returns empty device lists
	Log        *slog.Logger     // optional; enables request logging
}

// NewRouter builds the chi router.
func NewRouter(opt Options) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))
	if opt.Log != nil {
		r.Use(requestLogger(opt.Log))
	}

	ctrl := opt.Controller
	r.Get("/health", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "version": opt.Version})
	})
	r.Get("/docs", swaggerUIHandler)
	r.Get("/docs/openapi.yaml", serveOpenAPI)
	r.Get("/api/v1/openapi.yaml", serveOpenAPI)

	r.Route("/api/v1", func(r chi.Router) {
		r.Get("/status", func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusOK, ctrl.Status())
		})
		r.Get("/config", func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusOK, ctrl.Config().Clone())
		})
		r.Patch("/config", func(w http.ResponseWriter, req *http.Request) {
			body, err := io.ReadAll(io.LimitReader(req.Body, 1<<20))
			if err != nil {
				writeErr(w, http.StatusBadRequest, err)
				return
			}
			if err := ctrl.ApplyConfig(body); err != nil {
				writeErr(w, http.StatusBadRequest, err)
				return
			}
			writeJSON(w, http.StatusOK, ctrl.Config().Clone())
		})
		r.Get("/discovery/servers", func(w http.ResponseWriter, req *http.Request) {
			timeout := 3 * time.Second
			if v := req.URL.Query().Get("timeout_ms"); v != "" {
				if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 15000 {
					timeout = time.Duration(n) * time.Millisecond
				}
			}
			services, err := ctrl.DiscoverServers(req.Context(), timeout)
			if err != nil {
				writeErr(w, http.StatusBadGateway, err)
				return
			}
			if services == nil {
				services = []discovery.Service{}
			}
			writeJSON(w, http.StatusOK, map[string]any{"servers": services})
		})
		r.Get("/events", func(w http.ResponseWriter, req *http.Request) {
			serveSSE(w, req, ctrl)
		})
		r.Post("/ptt", func(w http.ResponseWriter, req *http.Request) {
			var body struct {
				Pressed bool `json:"pressed"`
			}
			if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
				writeErr(w, http.StatusBadRequest, err)
				return
			}
			var err error
			if body.Pressed {
				err = ctrl.ManualPTT().Press()
			} else {
				err = ctrl.ManualPTT().Release()
			}
			if err != nil {
				writeErr(w, http.StatusConflict, err)
				return
			}
			writeJSON(w, http.StatusOK, map[string]bool{"pressed": body.Pressed})
		})
		r.Post("/communicator/toggle", func(w http.ResponseWriter, _ *http.Request) {
			ctrl.ToggleCommunicator()
			writeJSON(w, http.StatusOK, map[string]string{"state": ctrl.Communicator().State().String()})
		})
		r.Post("/connection/connect", func(w http.ResponseWriter, req *http.Request) {
			if err := ctrl.Connect(req.Context()); err != nil {
				writeErr(w, http.StatusBadGateway, err)
				return
			}
			writeJSON(w, http.StatusOK, ctrl.Status())
		})
		r.Post("/connection/disconnect", func(w http.ResponseWriter, _ *http.Request) {
			ctrl.Disconnect()
			writeJSON(w, http.StatusOK, ctrl.Status())
		})
		r.Get("/channels", func(w http.ResponseWriter, _ *http.Request) {
			cl := ctrl.Client()
			if cl == nil {
				writeJSON(w, http.StatusOK, []any{})
				return
			}
			writeJSON(w, http.StatusOK, cl.Roster().Channels())
		})
		r.Post("/channels/{id}/join", func(w http.ResponseWriter, req *http.Request) {
			id64, err := strconv.ParseUint(chi.URLParam(req, "id"), 10, 32)
			if err != nil {
				writeErr(w, http.StatusBadRequest, err)
				return
			}
			if err := ctrl.JoinChannel(uint32(id64)); err != nil {
				writeErr(w, http.StatusBadRequest, err)
				return
			}
			writeJSON(w, http.StatusOK, map[string]uint32{"channel_id": uint32(id64)})
		})
		r.Get("/users", func(w http.ResponseWriter, _ *http.Request) {
			cl := ctrl.Client()
			if cl == nil {
				writeJSON(w, http.StatusOK, []any{})
				return
			}
			writeJSON(w, http.StatusOK, cl.Roster().Users())
		})
		r.Get("/chat/messages", func(w http.ResponseWriter, req *http.Request) {
			var channelID *uint32
			if v := req.URL.Query().Get("channel_id"); v != "" {
				id64, err := strconv.ParseUint(v, 10, 32)
				if err != nil {
					writeErr(w, http.StatusBadRequest, err)
					return
				}
				id := uint32(id64)
				channelID = &id
			}
			limit := 200
			if v := req.URL.Query().Get("limit"); v != "" {
				if n, err := strconv.Atoi(v); err == nil {
					limit = n
				}
			}
			msgs := ctrl.ChatMessages(channelID, limit)
			if msgs == nil {
				msgs = []controller.ChatMessage{}
			}
			writeJSON(w, http.StatusOK, map[string]any{"messages": msgs})
		})
		r.Get("/gpio/pins", func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusOK, map[string]any{"pins": controller.DefaultGPIOPins()})
		})
		r.Post("/text", func(w http.ResponseWriter, req *http.Request) {
			var body struct {
				Message   string `json:"message"`
				ChannelID uint32 `json:"channel_id"`
			}
			if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
				writeErr(w, http.StatusBadRequest, err)
				return
			}
			if strings.TrimSpace(body.Message) == "" {
				writeErr(w, http.StatusBadRequest, fmt.Errorf("message required"))
				return
			}
			msg, err := ctrl.SendChat(body.Message, body.ChannelID)
			if err != nil {
				writeErr(w, http.StatusBadGateway, err)
				return
			}
			writeJSON(w, http.StatusOK, map[string]any{"status": "sent", "message": msg})
		})
		r.Get("/audio/backends", func(w http.ResponseWriter, _ *http.Request) {
			ids := []string{audio.BackendAuto}
			if opt.Enumerator != nil {
				ids = opt.Enumerator.Backends()
			} else {
				ids = append(ids, audio.PlatformBackends()...)
			}
			labels := audio.BackendLabels()
			type item struct {
				ID    string `json:"id"`
				Label string `json:"label"`
			}
			out := make([]item, 0, len(ids))
			for _, id := range ids {
				label := labels[id]
				if label == "" {
					if id == "" {
						label = labels[audio.BackendAuto]
					} else {
						label = id
					}
				}
				out = append(out, item{ID: id, Label: label})
			}
			writeJSON(w, http.StatusOK, out)
		})
		r.Get("/audio/devices", func(w http.ResponseWriter, req *http.Request) {
			if opt.Enumerator == nil {
				writeJSON(w, http.StatusOK, map[string]any{"capture": []any{}, "playback": []any{}})
				return
			}
			q := req.URL.Query()
			// Prefer per-direction backends so the UI can refresh mic/speaker
			// lists independently when the user changes pipeline selection.
			captureBackend := q.Get("capture_backend")
			playbackBackend := q.Get("playback_backend")
			if shared := q.Get("backend"); shared != "" {
				if captureBackend == "" {
					captureBackend = shared
				}
				if playbackBackend == "" {
					playbackBackend = shared
				}
			}
			cap, err1 := opt.Enumerator.Devices(captureBackend, audio.DirectionCapture)
			play, err2 := opt.Enumerator.Devices(playbackBackend, audio.DirectionPlayback)
			resp := map[string]any{
				"capture":          cap,
				"playback":         play,
				"capture_backend":  captureBackend,
				"playback_backend": playbackBackend,
			}
			if err1 != nil {
				resp["capture_error"] = err1.Error()
				resp["capture"] = []any{}
			}
			if err2 != nil {
				resp["playback_error"] = err2.Error()
				resp["playback"] = []any{}
			}
			writeJSON(w, http.StatusOK, resp)
		})
		r.Get("/update", func(w http.ResponseWriter, req *http.Request) {
			u := ctrl.Updater()
			if u == nil {
				writeJSON(w, http.StatusOK, update.UpdateInfo{Current: opt.Version})
				return
			}
			if req.URL.Query().Get("refresh") == "1" {
				info, err := u.Check(req.Context())
				if err != nil {
					writeErr(w, http.StatusBadGateway, err)
					return
				}
				writeJSON(w, http.StatusOK, info)
				return
			}
			info, err := u.LoadCached()
			if err != nil {
				info = update.UpdateInfo{Current: opt.Version}
			}
			writeJSON(w, http.StatusOK, info)
		})
	})

	if opt.Frontend != nil {
		// Explicit / plus catch-all so chi serves the SPA at the site root.
		spa := spaHandler(opt.Frontend)
		r.Get("/", spa.ServeHTTP)
		r.Handle("/*", spa)
	}
	return r
}

func serveSSE(w http.ResponseWriter, req *http.Request, api *controller.Controller) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	ch := api.Subscribe()
	defer api.Unsubscribe(ch)

	// Initial status
	writeSSE(w, flusher, api.Status())

	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-req.Context().Done():
			return
		case <-ticker.C:
			writeSSE(w, flusher, map[string]any{"type": "status", "status": api.Status()})
		case ev, ok := <-ch:
			if !ok {
				return
			}
			writeSSE(w, flusher, ev)
		}
	}
}

func writeSSE(w http.ResponseWriter, flusher http.Flusher, v any) {
	data, _ := json.Marshal(v)
	fmt.Fprintf(w, "data: %s\n\n", data)
	flusher.Flush()
}

// appBaseHref returns the base href for the SPA so relative asset URLs
// (Vite base: './') resolve correctly on deep-link reloads and under HA ingress.
func appBaseHref(requestPath string) string {
	p := strings.TrimPrefix(requestPath, "/")
	parts := strings.Split(p, "/")
	if len(parts) >= 3 && parts[0] == "api" && parts[1] == "hassio_ingress" && parts[2] != "" {
		return "/" + strings.Join(parts[0:3], "/") + "/"
	}
	return "/"
}

func isAssetPath(rel string) bool {
	ext := strings.ToLower(path.Ext(rel))
	switch ext {
	case ".js", ".mjs", ".css", ".woff2", ".woff", ".ttf", ".eot", ".ico", ".png", ".jpg", ".jpeg", ".gif", ".svg", ".webp", ".json", ".map":
		return true
	}
	return strings.HasPrefix(rel, "assets/")
}

func spaHandler(frontend fs.FS) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rel := strings.TrimPrefix(path.Clean("/"+r.URL.Path), "/")
		if rel == "." || rel == "" || strings.HasSuffix(r.URL.Path, "/") {
			rel = "index.html"
		}
		if _, err := fs.Stat(frontend, rel); err != nil {
			if isAssetPath(rel) {
				http.NotFound(w, r)
				return
			}
			rel = "index.html"
		}
		if rel == "index.html" {
			serveSPAIndex(w, r, frontend)
			return
		}
		if !serveSPAFile(w, r, frontend, rel) {
			http.NotFound(w, r)
		}
	})
}

func serveSPAIndex(w http.ResponseWriter, r *http.Request, fsys fs.FS) {
	data, err := fs.ReadFile(fsys, "index.html")
	if err != nil {
		http.NotFound(w, r)
		return
	}
	// Inject <base href="..."> on non-root paths so ./assets/* resolves to the app root
	// (e.g. reload of /settings/server). Skip at "/" so HA ingress relative assets still work.
	requestPath := strings.TrimSuffix(r.URL.Path, "/")
	if requestPath == "" {
		requestPath = "/"
	}
	if requestPath != "/" {
		base := appBaseHref(r.URL.Path)
		baseTag := []byte(`<base href="` + base + `">`)
		head := []byte("<head>")
		if idx := bytes.Index(data, head); idx >= 0 {
			data = bytes.Join([][]byte{data[:idx+len(head)], baseTag, data[idx+len(head):]}, nil)
		}
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	http.ServeContent(w, r, "index.html", time.Time{}, bytes.NewReader(data))
}

func serveSPAFile(w http.ResponseWriter, r *http.Request, fsys fs.FS, name string) bool {
	data, err := fs.ReadFile(fsys, name)
	if err != nil {
		return false
	}
	ctype := contentTypeFor(name)
	if ctype != "" {
		w.Header().Set("Content-Type", ctype)
	}
	http.ServeContent(w, r, name, time.Time{}, bytes.NewReader(data))
	return true
}

func contentTypeFor(name string) string {
	switch strings.ToLower(path.Ext(name)) {
	case ".html":
		return "text/html; charset=utf-8"
	case ".js":
		return "text/javascript; charset=utf-8"
	case ".css":
		return "text/css; charset=utf-8"
	case ".svg":
		return "image/svg+xml"
	case ".json":
		return "application/json"
	case ".png":
		return "image/png"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".woff2":
		return "font/woff2"
	case ".woff":
		return "font/woff"
	default:
		return ""
	}
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

// requestLogger logs mutating API calls and errors; GETs only at Debug (except failures).
func requestLogger(log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
			next.ServeHTTP(ww, r)
			status := ww.Status()
			if status == 0 {
				status = http.StatusOK
			}
			path := r.URL.Path
			// Skip noisy SPA asset / health polls at Info.
			if r.Method == http.MethodGet && status < 400 &&
				(path == "/health" || path == "/api/v1/status" || strings.HasPrefix(path, "/assets/") ||
					path == "/" || strings.HasPrefix(path, "/settings") || path == "/status" || path == "/docs" ||
					strings.HasPrefix(path, "/api/v1/events")) {
				log.Debug("http", "method", r.Method, "path", path, "status", status,
					"ms", time.Since(start).Milliseconds())
				return
			}
			level := slog.LevelInfo
			if status >= 500 {
				level = slog.LevelError
			} else if status >= 400 {
				level = slog.LevelWarn
			}
			log.Log(r.Context(), level, "http",
				"method", r.Method,
				"path", path,
				"status", status,
				"ms", time.Since(start).Milliseconds(),
			)
		})
	}
}

func writeErr(w http.ResponseWriter, code int, err error) {
	writeJSON(w, code, map[string]string{"error": err.Error()})
}

func serveOpenAPI(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/x-yaml; charset=utf-8")
	_, _ = w.Write(apispec.OpenAPI)
}

// swaggerUIHTML loads Swagger UI from CDN. Spec URL is resolved relative to the
// current path so HA ingress prefixes continue to work.
const swaggerUIHTML = `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>talkiepi API Docs</title>
  <link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@5/swagger-ui.css">
  <style>body { margin: 0; }</style>
</head>
<body>
  <div id="swagger-ui"></div>
  <script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-bundle.js"></script>
  <script>
    (function () {
      var path = window.location.pathname.replace(/\/docs\/?$/, '');
      SwaggerUIBundle({
        url: path + '/api/v1/openapi.yaml',
        dom_id: '#swagger-ui',
        deepLinking: true,
      });
    })();
  </script>
</body>
</html>
`

func swaggerUIHandler(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(swaggerUIHTML))
}
