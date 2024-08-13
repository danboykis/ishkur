package routes

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/danboykis/ishkur/config"
	"github.com/danboykis/ishkur/db"
	"github.com/danboykis/ishkur/handler"
	"github.com/danboykis/ishkur/routes/middleware"
	"log/slog"
	"net"
	"net/http"
	"time"
)

func SetupHttpServer(conf *config.Config, ver config.Version, dbConn db.Db) *http.Server {
	h := SetupHttpRoutes(conf, ver, dbConn)

	srv := &http.Server{
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		Addr:         net.JoinHostPort(conf.Host, fmt.Sprintf("%d", conf.Port)),
		Handler:      h,
	}
	return srv
}

func VersionEndpoint(v config.Version) handler.APIHandler {
	return func(w http.ResponseWriter, req *http.Request) error {
		w.Header().Set("Content-Type", "application/json")
		return json.NewEncoder(w).Encode(v)
	}
}

func ConfigEndpoint(c config.Config) handler.APIHandler {
	return func(w http.ResponseWriter, req *http.Request) error {
		w.Header().Set("Content-Type", "application/json")
		return json.NewEncoder(w).Encode(c)
	}
}

func LookupEndpoint(dbConn db.Db) handler.APIHandler {
	return func(w http.ResponseWriter, req *http.Request) error {
		ctx, cancel := context.WithTimeout(req.Context(), 3*time.Second)
		defer cancel()
		k := req.PathValue("key")
		value, err := dbConn.Get(ctx, k)
		if err != nil {
			switch {
			case errors.Is(err, db.NotFoundError):
				return &handler.ApiError{Status: 404, Message: fmt.Sprintf("could not find %s", k)}
			default:
				slog.Warn("redis value", "key", k, "error", err)
				return err
			}
		}
		w.Header().Set("Content-Type", "application/json")
		response := struct {
			Value string `json:"value"`
		}{}
		response.Value = value
		return json.NewEncoder(w).Encode(response)
	}
}

func WriteEndpoint(db db.Db) handler.APIHandler {
	return func(w http.ResponseWriter, req *http.Request) error {
		ctx, cancel := context.WithTimeout(req.Context(), 3*time.Second)
		defer cancel()

		k := req.PathValue("key")
		v := struct {
			Value string `json:"value"`
		}{}
		if err := json.NewDecoder(req.Body).Decode(&v); err != nil {
			return err
		}
		if err := db.Set(ctx, k, v.Value); err != nil {
			return err
		}
		w.WriteHeader(http.StatusCreated)
		return nil
	}
}

func SetupHttpRoutes(conf *config.Config, ver config.Version, dbConn db.Db) *http.ServeMux {
	wrap := func(ah handler.APIHandler) handler.APIHandler {
		return middleware.CreateStack(middleware.TimerMiddleware, middleware.AuthInspector)(ah)
	}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /version", wrap(VersionEndpoint(ver)).ToHttpHandlerFunc())
	mux.HandleFunc("GET /config", wrap(ConfigEndpoint(*conf)).ToHttpHandlerFunc())
	mux.HandleFunc("GET /lookup/{key}", wrap(LookupEndpoint(dbConn)).ToHttpHandlerFunc())
	mux.HandleFunc("POST /lookup/{key}", wrap(WriteEndpoint(dbConn)).ToHttpHandlerFunc())
	return mux
}
