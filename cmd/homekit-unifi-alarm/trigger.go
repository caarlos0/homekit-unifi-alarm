package main

import (
	"crypto/subtle"
	"io"
	"net/http"
	"strings"
)

func triggerHandler(alarm *SecuritySystem, configuredToken string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if configuredToken == "" {
			log.Warn("trigger endpoint disabled: no token configured", "addr", r.RemoteAddr)
			http.Error(w, "trigger endpoint disabled", http.StatusServiceUnavailable)
			return
		}
		token := r.URL.Query().Get("token")
		if h := r.Header.Get("Authorization"); strings.HasPrefix(h, "Bearer ") {
			token = strings.TrimPrefix(h, "Bearer ")
		}
		if subtle.ConstantTimeCompare([]byte(token), []byte(configuredToken)) != 1 {
			log.Warn("trigger request with invalid token", "addr", r.RemoteAddr)
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		if !alarm.Trigger() {
			log.Info("ignoring trigger: alarm is disarmed", "addr", r.RemoteAddr)
			_, _ = io.WriteString(w, "ignored: alarm is disarmed\n")
			return
		}
		log.Warn("alarm triggered", "addr", r.RemoteAddr)
		_, _ = io.WriteString(w, "triggered\n")
	}
}
