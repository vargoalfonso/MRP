package cmd

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	appconf "github.com/ganasa18/go-template/config"
	"github.com/spf13/cobra"
)

var HttpCmd = &cobra.Command{
	Use:   "http",
	Short: "Run the HTTP API server",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := appconf.InitAppConfig()
		appconf.InitLogger(cfg)

		srv, err := initHTTP(cfg)
		if err != nil {
			return err
		}

		// Start server in a goroutine so we can listen for signals.
		errCh := make(chan error, 1)
		go func() {
			if err := srv.Start(); err != nil && !errors.Is(err, http.ErrServerClosed) {
				errCh <- err
			}
		}()

		// Wait for OS signal or server error.
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

		select {
		case sig := <-quit:
			slog.Info("shutdown signal received", slog.String("signal", sig.String()))
		case err := <-errCh:
			return err
		}

		// Graceful shutdown with configured timeout.
		ctx, cancel := context.WithTimeout(context.Background(), cfg.HTTPShutdownTimeout)
		defer cancel()

		if err := srv.Shutdown(ctx); err != nil {
			slog.Error("shutdown error", slog.Any("error", err))
			return err
		}

		slog.Info("server stopped cleanly")
		return nil
	},
}

// init registers HttpCmd with the default os.Exit behaviour on fatal errors.
func init() {
	_ = HttpCmd // registered in root.go
}

// exitOnSignal allows tests to replace os.Exit; not used in production path.
var exitOnSignal = func(code int) { os.Exit(code) }
