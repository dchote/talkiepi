package rest

import (
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/dchote/talkiepi/internal/audio"
	"github.com/dchote/talkiepi/internal/controller"
	"github.com/dchote/talkiepi/internal/update"
)

// Options configures the HTTP server.
type Options struct {
	Controller *controller.Controller
	Frontend   fs.FS // nil = API only
	Version    string
	Enumerator audio.Enumerator // optional; nil returns empty device lists
}

// NewRouter builds the chi router.
func NewRouter(opt Options) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))

	api := opt.Controller
	r.Get("/health", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "version": opt.Version})
	})
	r.Get("/docs", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "api/openapi.yaml")
	})
	r.Get("/docs/openapi.yaml", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "api/openapi.yaml")
	})

	r.Route("/api/v1", func(r chi.Router) {
		r.Get("/status", func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusOK, api.Status())
		})
		r.Get("/config", func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusOK, api.Config().Clone())
		})
		r.Patch("/config", func(w http.ResponseWriter, req *http.Request) {
			body, err := io.ReadAll(io.LimitReader(req.Body, 1<<20))
			if err != nil {
				writeErr(w, http.StatusBadRequest, err)
				return
			}
			if err := api.ApplyConfig(body); err != nil {
				writeErr(w, http.StatusBadRequest, err)
				return
			}
			writeJSON(w, http.StatusOK, api.Config().Clone())
		})
		r.Get("/events", func(w http.ResponseWriter, req *http.Request) {
			serveSSE(w, req, api)
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
				err = api.ManualPTT().Press()
			} else {
				err = api.ManualPTT().Release()
			}
			if err != nil {
				writeErr(w, http.StatusConflict, err)
				return
			}
			writeJSON(w, http.StatusOK, map[string]bool{"pressed": body.Pressed})
		})
		r.Post("/communicator/toggle", func(w http.ResponseWriter, _ *http.Request) {
			api.Communicator().Toggle()
			writeJSON(w, http.StatusOK, map[string]string{"state": api.Communicator().State().String()})
		})
		r.Post("/connection/connect", func(w http.ResponseWriter, req *http.Request) {
			if err := api.Connect(req.Context()); err != nil {
				writeErr(w, http.StatusBadGateway, err)
				return
			}
			writeJSON(w, http.StatusOK, api.Status())
		})
		r.Post("/connection/disconnect", func(w http.ResponseWriter, _ *http.Request) {
			api.Disconnect()
			writeJSON(w, http.StatusOK, api.Status())
		})
		r.Get("/channels", func(w http.ResponseWriter, _ *http.Request) {
			cl := api.Client()
			if cl == nil {
				writeJSON(w, http.StatusOK, []any{})
				return
			}
			writeJSON(w, http.StatusOK, cl.Roster().Channels())
		})
		r.Post("/channels/{id}/join", func(w http.ResponseWriter, req *http.Request) {
			cl := api.Client()
			if cl == nil {
				writeErr(w, http.StatusConflict, fmt.Errorf("not connected"))
				return
			}
			id64, err := strconv.ParseUint(chi.URLParam(req, "id"), 10, 32)
			if err != nil {
				writeErr(w, http.StatusBadRequest, err)
				return
			}
			if err := cl.JoinChannelID(uint32(id64)); err != nil {
				writeErr(w, http.StatusBadRequest, err)
				return
			}
			writeJSON(w, http.StatusOK, map[string]uint32{"channel_id": uint32(id64)})
		})
		r.Get("/users", func(w http.ResponseWriter, _ *http.Request) {
			cl := api.Client()
			if cl == nil {
				writeJSON(w, http.StatusOK, []any{})
				return
			}
			writeJSON(w, http.StatusOK, cl.Roster().Users())
		})
		r.Post("/text", func(w http.ResponseWriter, req *http.Request) {
			cl := api.Client()
			if cl == nil {
				writeErr(w, http.StatusConflict, fmt.Errorf("not connected"))
				return
			}
			var body struct {
				Message   string `json:"message"`
				ChannelID uint32 `json:"channel_id"`
			}
			if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
				writeErr(w, http.StatusBadRequest, err)
				return
			}
			if body.ChannelID == 0 {
				body.ChannelID = cl.ChannelID()
			}
			if err := cl.SendTextMessage(body.Message, body.ChannelID); err != nil {
				writeErr(w, http.StatusBadGateway, err)
				return
			}
			writeJSON(w, http.StatusOK, map[string]string{"status": "sent"})
		})
		r.Get("/audio/backends", func(w http.ResponseWriter, _ *http.Request) {
			if opt.Enumerator == nil {
				writeJSON(w, http.StatusOK, []string{"default"})
				return
			}
			writeJSON(w, http.StatusOK, opt.Enumerator.Backends())
		})
		r.Get("/audio/devices", func(w http.ResponseWriter, req *http.Request) {
			if opt.Enumerator == nil {
				writeJSON(w, http.StatusOK, map[string]any{"capture": []any{}, "playback": []any{}})
				return
			}
			backend := req.URL.Query().Get("backend")
			cap, err1 := opt.Enumerator.Devices(backend, audio.DirectionCapture)
			play, err2 := opt.Enumerator.Devices(backend, audio.DirectionPlayback)
			if err1 != nil || err2 != nil {
				err := err1
				if err == nil {
					err = err2
				}
				writeErr(w, http.StatusInternalServerError, err)
				return
			}
			writeJSON(w, http.StatusOK, map[string]any{"capture": cap, "playback": play})
		})
		r.Get("/update", func(w http.ResponseWriter, req *http.Request) {
			u := api.Updater()
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
		r.Handle("/*", spaHandler(opt.Frontend))
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

func spaHandler(frontend fs.FS) http.Handler {
	fileServer := http.FileServer(http.FS(frontend))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/")
		if path != "" {
			if f, err := frontend.Open(path); err == nil {
				_ = f.Close()
				fileServer.ServeHTTP(w, r)
				return
			}
		}
		// SPA fallback to index.html
		r2 := *r
		u := *r.URL
		u.Path = "/index.html"
		r2.URL = &u
		fileServer.ServeHTTP(w, &r2)
	})
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, code int, err error) {
	writeJSON(w, code, map[string]string{"error": err.Error()})
}
