package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/bak1an/artf/version"
)

func Version(w http.ResponseWriter, r *http.Request) {
	vv := version.GetBuildInfo()
	resp, err := json.Marshal(&vv)
	if err != nil {
		slog.Error("cannot marshal version info", "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	w.Write(resp)
}
