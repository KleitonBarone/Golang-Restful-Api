package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

const (
	defaultListenAddress  = "localhost:8080"
	requestReadTimeout    = 10 * time.Second
	responseWriteTimeout  = 10 * time.Second
	requestHeaderTimeout  = 5 * time.Second
	maxRequestHeaderBytes = 16 << 10
	idleConnectionTimeout = 60 * time.Second
	shutdownTimeout       = 5 * time.Second
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	listener, err := net.Listen("tcp", listenAddress())
	if err != nil {
		return fmt.Errorf("listen: %w", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	server := newHTTPServer(setupRouter())
	return runHTTPServer(ctx, server, listener, shutdownTimeout)
}

func newHTTPServer(handler http.Handler) *http.Server {
	return &http.Server{
		Handler:           handler,
		ReadTimeout:       requestReadTimeout,
		WriteTimeout:      responseWriteTimeout,
		ReadHeaderTimeout: requestHeaderTimeout,
		MaxHeaderBytes:    maxRequestHeaderBytes,
		IdleTimeout:       idleConnectionTimeout,
	}
}

func runHTTPServer(ctx context.Context, server *http.Server, listener net.Listener, timeout time.Duration) error {
	serveDone := make(chan error, 1)
	go func() {
		serveDone <- server.Serve(listener)
	}()

	select {
	case err := <-serveDone:
		return ignoreServerClosed(err)
	case <-ctx.Done():
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		shutdownErr := fmt.Errorf("shut down HTTP server: %w", err)

		var closeErr error
		if err := server.Close(); err != nil {
			closeErr = fmt.Errorf("close HTTP server: %w", err)
		}

		var serveErr error
		if err := ignoreServerClosed(<-serveDone); err != nil {
			serveErr = fmt.Errorf("serve HTTP: %w", err)
		}
		return errors.Join(shutdownErr, closeErr, serveErr)
	}
	return ignoreServerClosed(<-serveDone)
}

func ignoreServerClosed(err error) error {
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}

func listenAddress() string {
	if address := os.Getenv("LISTEN_ADDRESS"); address != "" {
		return address
	}
	return defaultListenAddress
}
