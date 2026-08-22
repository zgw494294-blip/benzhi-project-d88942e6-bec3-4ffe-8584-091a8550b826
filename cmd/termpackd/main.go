package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		log.Fatal(err)
	}
}

func run(args []string) error {
	cfg, err := parseConfig(args)
	if err != nil {
		return err
	}
	runtime, err := buildRuntime(cfg)
	if err != nil {
		return err
	}
	defer runtime.close()
	done := runtime.serve()
	if cfg.selfcheck {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		checkErr := runSelfcheck(ctx, cfg.address)
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer shutdownCancel()
		shutdownErr := runtime.server.Shutdown(shutdownCtx)
		serveErr := <-done
		if checkErr != nil {
			return fmt.Errorf("selfcheck 失败: %w", checkErr)
		}
		if shutdownErr != nil {
			return shutdownErr
		}
		if serveErr != nil {
			return serveErr
		}
		fmt.Println("selfcheck 通过：术语包已完成纠错修订并生成不可变发布凭据")
		return nil
	}
	log.Printf("口译术语包发布工作台监听于 http://%s", cfg.address)
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)
	select {
	case err := <-done:
		return err
	case <-signals:
		ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
		defer cancel()
		return runtime.server.Shutdown(ctx)
	}
}
