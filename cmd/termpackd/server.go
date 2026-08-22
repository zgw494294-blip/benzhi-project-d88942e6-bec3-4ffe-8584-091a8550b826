package main

import (
	"errors"
	"log"
	"net"
	"net/http"
	"time"

	"termpack/internal/application"
	"termpack/internal/httpapi"
	"termpack/internal/store/sqlite"
	"termpack/internal/webui"
)

type runtime struct {
	store    *sqlite.Store
	server   *http.Server
	listener net.Listener
}

func buildRuntime(cfg config) (*runtime, error) {
	store, err := sqlite.Open(cfg.database)
	if err != nil {
		return nil, err
	}
	service := application.NewService(store)
	mux := http.NewServeMux()
	httpapi.New(service).Register(mux)
	mux.Handle("GET /", webui.Handler())
	listener, err := net.Listen("tcp", cfg.address)
	if err != nil {
		store.Close()
		return nil, err
	}
	server := &http.Server{Addr: cfg.address, Handler: httpapi.SecurityHeaders(mux), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 15 * time.Second, WriteTimeout: 20 * time.Second, IdleTimeout: 60 * time.Second}
	return &runtime{store: store, server: server, listener: listener}, nil
}

func (r *runtime) serve() <-chan error {
	done := make(chan error, 1)
	go func() {
		err := r.server.Serve(r.listener)
		if errors.Is(err, http.ErrServerClosed) {
			err = nil
		}
		done <- err
	}()
	return done
}

func (r *runtime) close() {
	if err := r.store.Close(); err != nil {
		log.Printf("关闭数据库: %v", err)
	}
}
