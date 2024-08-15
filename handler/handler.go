package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
)

type ApiError struct {
	Message string `json:"error"`
	Status  int    `json:"-"`
}

func (ae *ApiError) Error() string {
	return ae.Message
}

type APIHandler func(http.ResponseWriter, *http.Request) error

func (ah APIHandler) HandleError(w http.ResponseWriter, req *http.Request, err error) {
	if err != nil {
		var apiErrResponse *ApiError
		switch {
		case errors.As(err, &apiErrResponse):
			w.Header().Add("Content-Type", "application/json")
			w.WriteHeader(apiErrResponse.Status)
			jerr := json.NewEncoder(w).Encode(apiErrResponse)
			if jerr != nil {
				slog.LogAttrs(req.Context(), slog.LevelError, "could not serialize response", slog.String("error", jerr.Error()))
				return
			}
		default:
			w.Header().Add("Content-Type", "text/plain")
			w.WriteHeader(http.StatusInternalServerError)
			_, werr := fmt.Fprint(w, err.Error())
			if werr != nil {
				slog.LogAttrs(req.Context(), slog.LevelError, "could not write error", slog.Any("error", werr))
				return
			}
		}
	}
}

func (ah APIHandler) ToHttpHandlerFunc() http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {

		if err := ah(w, req); err != nil {
			ah.HandleError(w, req, err)
		}
	}
}
