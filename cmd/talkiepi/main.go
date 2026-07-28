package main

import (
	"context"
	"flag"
	"fmt"
	"io/fs"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/dchote/talkiepi/internal/audio"
	"github.com/dchote/talkiepi/internal/config"
	"github.com/dchote/talkiepi/internal/controller"
	"github.com/dchote/talkiepi/internal/identity"
	"github.com/dchote/talkiepi/internal/led"
	"github.com/dchote/talkiepi/internal/rest"
	"github.com/dchote/talkiepi/internal/update"
)

var (
	version   = "dev"
	commit    = "unknown"
	buildTime = "unknown"
)

func main() {
	configPath := flag.String("config", defaultConfigPath(), "path to talkiepi.yaml")
	showVersion := flag.Bool("version", false, "print version and exit")
	frontendEmbed := flag.Bool("frontend-embed", true, "serve embedded frontend when built with embed_frontend")
	frontendDir := flag.String("frontend-dir", "frontend/dist", "directory to serve when -frontend-embed=false (or embed unavailable)")
	flag.Parse()

	if *showVersion {
		fmt.Printf("talkiepi %s (commit %s, built %s)\n", version, commit, buildTime)
		os.Exit(0)
	}

	cfg, err := config.Load(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "config: %v\n", err)
		os.Exit(1)
	}
	if cfg.SeedUsernameIfEmpty(identity.DefaultUsername()) {
		if err := cfg.Save(""); err != nil {
			fmt.Fprintf(os.Stderr, "config: seed username: %v\n", err)
		}
	}

	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: parseLevel(cfg.Logging.Level)}))
	slog.SetDefault(log)

	stateDir := filepath.Join(filepath.Dir(*configPath), "..")
	if filepath.Base(filepath.Dir(*configPath)) == "talkiepi" {
		stateDir = "/var/lib/talkiepi"
	}
	if d := os.Getenv("TALKIEPI_STATE_DIR"); d != "" {
		stateDir = d
	}
	_ = os.MkdirAll(stateDir, 0o750)

	var updater *update.Checker
	if cfg.Updates.Enabled {
		owner, repo := "dchote", "talkiepi"
		if cfg.Updates.Repo != "" {
			parts := splitRepo(cfg.Updates.Repo)
			owner, repo = parts[0], parts[1]
		}
		updater = update.NewChecker(owner, repo, version, filepath.Join(stateDir, "update-cache.json"))
	}

	leds := led.Indicator(led.NewMemory())
	if cfg.LEDs.Enabled {
		leds = led.NewGPIO(led.GPIOConfig{
			Enabled:          true,
			Chip:             cfg.LEDs.Chip,
			OnlineLine:       cfg.LEDs.OnlineLine,
			ParticipantsLine: cfg.LEDs.ParticipantsLine,
			TransmitLine:     cfg.LEDs.TransmitLine,
		})
	}

	enumerator := audio.NewMalgoEnumerator()
	ctrl := controller.New(controller.Options{
		Config:     cfg,
		Logger:     log,
		Version:    version,
		LEDs:       leds,
		Updater:    updater,
		Enumerator: enumerator,
	})

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if err := ctrl.Start(ctx); err != nil {
		log.Error("controller start", "err", err)
		os.Exit(1)
	}

	frontend, frontendSrc := resolveFrontendFS(*frontendEmbed, *frontendDir, log)
	handler := rest.NewRouter(rest.Options{
		Controller: ctrl,
		Frontend:   frontend,
		Version:    version,
		Enumerator: enumerator,
		Log:        log,
	})

	addr := net.JoinHostPort(cfg.Web.Host, fmt.Sprintf("%d", cfg.Web.Port))
	srv := &http.Server{Addr: addr, Handler: handler, ReadHeaderTimeout: 10 * time.Second}
	httpErr := make(chan error, 1)
	go func() {
		log.Info("listening", "addr", addr, "version", version, "frontend", frontendSrc)
		err := srv.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
			httpErr <- err
			return
		}
		httpErr <- nil
	}()

	if updater != nil {
		go func() {
			ticker := time.NewTicker(time.Duration(cfg.Updates.IntervalHours) * time.Hour)
			defer ticker.Stop()
			_, _ = updater.Check(ctx)
			for {
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
					_, _ = updater.Check(ctx)
				}
			}
		}()
	}

	exitCode := 0
	select {
	case <-ctx.Done():
		log.Info("shutting down")
	case err := <-httpErr:
		if err != nil {
			log.Error("http", "err", err)
			exitCode = 1
		}
	}

	// Restore default signal handling so a second Ctrl+C force-exits if shutdown hangs.
	stop()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 8*time.Second)
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Warn("http shutdown", "err", err)
	}
	shutdownCancel()
	ctrl.Stop()
	log.Info("stopped")
	if exitCode != 0 {
		os.Exit(exitCode)
	}
}

func defaultConfigPath() string {
	if v := os.Getenv("TALKIEPI_CONFIG"); v != "" {
		return v
	}
	return "configs/talkiepi.yaml"
}

func parseLevel(s string) slog.Level {
	switch s {
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

func splitRepo(repo string) [2]string {
	for i := 0; i < len(repo); i++ {
		if repo[i] == '/' {
			return [2]string{repo[:i], repo[i+1:]}
		}
	}
	return [2]string{"dchote", repo}
}

// resolveFrontendFS picks an embedded FS, then a directory on disk.
// Returns nil FS when neither is available (API-only mode).
func resolveFrontendFS(embed bool, dir string, log *slog.Logger) (fs.FS, string) {
	if embed {
		if fsys, err := getFrontendFS(); err == nil && fsys != nil {
			if _, err := fs.Stat(fsys, "index.html"); err == nil {
				return fsys, "embedded"
			}
		}
	}
	if dir == "" {
		dir = "frontend/dist"
	}
	index := filepath.Join(dir, "index.html")
	if st, err := os.Stat(index); err == nil && !st.IsDir() {
		return os.DirFS(dir), dir
	}
	log.Warn("frontend not available; serving API only",
		"hint", "run: cd frontend && yarn build  (or yarn dev on :3000)",
		"frontend_dir", dir,
		"frontend_embed", embed,
	)
	return nil, "none"
}
