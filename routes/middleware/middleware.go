package middleware

import (
	"github.com/danboykis/ishkur/handler"
	"log/slog"
	"net/http"
	"time"
)

type wrapperResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (wrw *wrapperResponseWriter) WriteHeader(statusCode int) {
	wrw.statusCode = statusCode
	wrw.ResponseWriter.WriteHeader(statusCode)
}

type Middleware func(handler handler.APIHandler) handler.APIHandler

func CreateStack(ms ...Middleware) Middleware {
	return func(next handler.APIHandler) handler.APIHandler {
		f := next
		for i := len(ms) - 1; i >= 0; i-- {
			last := ms[i]
			f = last(f)
		}
		return f
	}
}

func TimerMiddleware(next handler.APIHandler) handler.APIHandler {
	return func(rw http.ResponseWriter, req *http.Request) error {
		logger := slog.Default().With("path", req.URL.Path)
		now := time.Now()
		wrw := &wrapperResponseWriter{ResponseWriter: rw, statusCode: http.StatusOK}

		//log.Printf("%s %s got %d took %v\n", req.Method, req.URL.Path, wrw.statusCode, time.Since(now))
		if err := next(wrw, req); err != nil {
			logger.LogAttrs(req.Context(), slog.LevelInfo, "timer with error",
				slog.String("method", req.Method), slog.Int("status", wrw.statusCode), slog.Duration("took", time.Since(now)))
			return err
		}
		logger.LogAttrs(req.Context(), slog.LevelInfo, "timer",
			slog.String("method", req.Method), slog.Int("status", wrw.statusCode), slog.Duration("took", time.Since(now)))
		return nil
	}
}

func AuthInspector(next handler.APIHandler) handler.APIHandler {
	return func(w http.ResponseWriter, req *http.Request) error {
		if v, exists := req.Header["Authorization"]; !exists {
			w.WriteHeader(401)
			return &handler.ApiError{Status: 401, Message: "No Authorization header"}
		} else {
			slog.Info("auth header", "auth", v[0])
			if err := next(w, req); err != nil {
				return err
			}
			return nil
		}
	}
}
