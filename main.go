package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/danboykis/ishkur/config"
	"github.com/danboykis/ishkur/state"
	"gopkg.in/natefinch/lumberjack.v2"
	"io"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"runtime/debug"
	"sync"
	"syscall"
	"time"
)

func gitVersion() (*config.Version, error) {
	v := &config.Version{}
	if bi, ok := debug.ReadBuildInfo(); ok {
		for _, kv := range bi.Settings {
			switch kv.Key {
			case "vcs.revision":
				v.Checksum = kv.Value
			case "vcs.time":
				LastCommit, err := time.Parse(time.RFC3339, kv.Value)
				if err == nil {
					v.DateTime = LastCommit
				}
			}
		}
		return v, nil
	}
	return v, errors.New("could not find git info")
}

func main() {
	version, verr := gitVersion()
	if verr != nil {
		log.Fatalf("%v\n", verr)
	}
	fmt.Printf("Starting ishkur: %+v\n", version)

	s := &state.States{Version: version}

	if err := run(context.Background(), s); err != nil {
		log.Fatalf("%v\n", err)
	}
}

func run(parentCtx context.Context, s *state.States) error {
	ctx, cancel := signal.NotifyContext(parentCtx, os.Interrupt, syscall.SIGTERM)
	defer cancel()

	if err := s.InitConfig(); err != nil {
		return err
	}

	setupLogging(s.Config, s.Version)

	if err := s.InitDb(); err != nil {
		return err
	}

	if err := s.InitHttpServer(); err != nil {
		return err
	}

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		log.Printf("listening on %s\n", s.HttpServer.Addr)
		if err := s.HttpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			fmt.Fprintf(os.Stderr, "error listening and serving: %s\n", err)
		}
	}()

	go func() {
		defer wg.Done()
		<-ctx.Done()

		stopFns := []func(context.Context) error{s.StopHttpServer, s.StopDb}

		for _, fn := range stopFns {
			shutdownCtx, cancelCtx := context.WithTimeout(parentCtx, 5*time.Second)

			if err := fn(shutdownCtx); err != nil {
				fmt.Fprintf(os.Stderr, "%v\n", err)
			}

			cancelCtx()
		}
	}()

	wg.Wait()

	return nil
}

func setupLogging(conf *config.Config, v *config.Version) {
	llog := &lumberjack.Logger{
		Filename:   conf.LogPath,
		MaxSize:    1024, // megabytes
		MaxBackups: 1,
		MaxAge:     30,    //days
		Compress:   false, // disabled by default
	}

	logger := slog.New(slog.NewJSONHandler(io.MultiWriter(llog, os.Stdout), nil))
	slog.SetDefault(logger)

	slog.Info("ishkur", "version", v)
}
