package handler

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/bak1an/artf/version"
)

func Version(w http.ResponseWriter, r *http.Request) {
	vv := version.GetBuildInfo()
	buf := new(bytes.Buffer)
	if err := json.NewEncoder(buf).Encode(vv); err != nil {
		slog.Error("cannot encode version", "error", err)
		w.WriteHeader(http.StatusInternalServerError)
	} else {
		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "application/json")
		if _, err := w.Write(buf.Bytes()); err != nil {
			slog.Error("cannot write version", "error", err)
		}
	}
}
