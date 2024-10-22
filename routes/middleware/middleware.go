package middleware

import (
	"github.com/danboykis/ishkur/handler"
	"log"
	"log/slog"
	"net/http"
	"time"
)

type ArgGroup struct {
	w   http.ResponseWriter
	r   *http.Request
	err error
}

type Middleware interface {
	Enter(ag *ArgGroup)
	Leave(ag *ArgGroup)
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

func (mh *Handler) ExecutePipeline(w http.ResponseWriter, r *http.Request, pipeline []Middleware) {

	ag := &ArgGroup{w: w, r: r, err: nil}

	for i := 0; i < len(pipeline); i++ {
		pipeline[i].Enter(ag)
	}

	if ag.err != nil {
		slog.LogAttrs(r.Context(), slog.LevelError, "error in middleware pipeline skipping handler", slog.String("error", ag.err.Error()))
		mh.Handler.HandleError(ag.w, ag.r, ag.err)
	} else {
		ag.err = mh.Handler(ag.w, ag.r)
		mh.Handler.HandleError(ag.w, ag.r, ag.err)
	}

	for i := len(pipeline) - 1; i >= 0; i-- {
		pipeline[i].Leave(ag)
	}
}

type AuthMiddleware struct{}

func (amw *AuthMiddleware) Enter(ag *ArgGroup) {
	if ag.err != nil {
		return
	}
	if v, exists := ag.r.Header["Authorization"]; exists {
		slog.LogAttrs(ag.r.Context(), slog.LevelInfo, "auth header", slog.String("auth", v[0]))
		return
	}
	ag.err = &handler.ApiError{Status: 401, Message: "No Authorization header"}
}

func (amw *AuthMiddleware) Leave(_ *ArgGroup) {
	return
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

func (tmw *TimerMiddleware) Enter(mg *ArgGroup) {
	now := time.Now()
	wrw := &wrapperResponseWriter{ResponseWriter: mg.w}
	mg.w = wrw
	tmw.w = wrw
	tmw.startTime = now
	return
}

func (tmw *TimerMiddleware) Leave(ag *ArgGroup) {
	slog.Default().
		With("method", ag.r.Method).
		With("path", ag.r.URL.Path).
		With("query", ag.r.URL.RawQuery).
		LogAttrs(ag.r.Context(), slog.LevelInfo, "timer",
			slog.Int("status", tmw.w.statusCode),
			slog.Duration("took", time.Since(tmw.startTime)))
}
