package middleware

import (
	"github.com/danboykis/ishkur/handler"
	"log"
	"log/slog"
	"net/http"
	"time"
)

type Middleware interface {
	Enter(w http.ResponseWriter, r *http.Request, err error) (http.ResponseWriter, *http.Request, error)
	Leave(w http.ResponseWriter, r *http.Request, err error) (http.ResponseWriter, *http.Request, error)
}

type Handler struct {
	MakePipeline func() []Middleware
	Handler      handler.APIHandler
}

func NewMiddlewareHandler(h handler.APIHandler, f func() []Middleware) *Handler {
	if h == nil {
		log.Fatalln("Handler cannot be nil")
	}

	return &Handler{MakePipeline: f, Handler: h}
}

func (mh *Handler) ToHandlerFunc() http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		mh.ExecutePipeline(w, req, mh.MakePipeline())
	}
}

type interceptor func(http.ResponseWriter, *http.Request, error) (http.ResponseWriter, *http.Request, error)

func (mh *Handler) exec(f interceptor, w *http.ResponseWriter, req **http.Request, err *error) {
	nextWriter, nextReq, nextErr := f(*w, *req, *err)
	*w = nextWriter
	*req = nextReq
	*err = nextErr
}

func (mh *Handler) ExecutePipeline(w http.ResponseWriter, r *http.Request, pipeline []Middleware) {

	var err error = nil

	for i := 0; i < len(pipeline); i++ {
		mh.exec(pipeline[i].Enter, &w, &r, &err)
	}

	if err != nil {
		slog.LogAttrs(r.Context(), slog.LevelError, "error in middleware pipeline skipping handler", slog.String("error", err.Error()))
		mh.Handler.HandleError(w, r, err)
	} else {
		err = mh.Handler(w, r)
		mh.Handler.HandleError(w, r, err)
	}

	for i := len(pipeline) - 1; i >= 0; i-- {
		mh.exec(pipeline[i].Leave, &w, &r, &err)
	}
}

type AuthMiddleware struct{}

func (amw *AuthMiddleware) Enter(w http.ResponseWriter, req *http.Request, err error) (http.ResponseWriter, *http.Request, error) {
	if err != nil {
		return w, req, err
	}
	if v, exists := req.Header["Authorization"]; exists {
		slog.Info("auth header", "auth", v[0])
		return w, req, nil
	}
	return w, req, &handler.ApiError{Status: 401, Message: "No Authorization header"}
}

func (amw *AuthMiddleware) Leave(w http.ResponseWriter, r *http.Request, err error) (http.ResponseWriter, *http.Request, error) {
	return w, r, err
}

type TimerMiddleware struct {
	startTime time.Time
	w         *wrapperResponseWriter
}

type wrapperResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (wrw *wrapperResponseWriter) WriteHeader(statusCode int) {
	wrw.statusCode = statusCode
	wrw.ResponseWriter.WriteHeader(statusCode)
}

func (tmw *TimerMiddleware) Enter(w http.ResponseWriter, req *http.Request, err error) (http.ResponseWriter, *http.Request, error) {
	now := time.Now()
	tmw.w = &wrapperResponseWriter{ResponseWriter: w}
	tmw.startTime = now
	return tmw.w, req, err
}

func (tmw *TimerMiddleware) Leave(w http.ResponseWriter, req *http.Request, err error) (http.ResponseWriter, *http.Request, error) {
	logger := slog.Default().
		With("method", req.Method).
		With("path", req.URL.Path).
		With("query", req.URL.RawQuery)

	logger.LogAttrs(req.Context(), slog.LevelInfo, "timer",
		slog.Int("status", tmw.w.statusCode),
		slog.Duration("took", time.Since(tmw.startTime)))

	return w, req, err
}
