package main

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/brutella/hap"
	"github.com/brutella/hap/accessory"
	"github.com/caarlos0/env/v11"
	logp "github.com/charmbracelet/log"
)

var log = logp.NewWithOptions(os.Stderr, logp.Options{
	ReportTimestamp: true,
	TimeFormat:      time.Kitchen,
	Prefix:          "homekit",
})

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

const manufacturer = "Ubiquiti"

func main() {
	log.Info(
		"homekit-unifi-alarm",
		"version", version,
		"commit", commit,
		"date", date,
		"info", strings.Join([]string{
			"Homekit alarm for UniFi Protect",
			"© Carlos Alexandro Becker",
			"https://becker.software",
		}, "\n"),
	)

	var cfg Config
	if err := env.Parse(&cfg); err != nil {
		log.Fatal(
			"could not parse env",
			"err",
			strings.TrimPrefix(strings.ReplaceAll(err.Error(), "; ", "\n"), "env: ")+"\n",
		)
	}

	fs := hap.NewFsStore("./db")

	alarm := NewSecuritySystem(accessory.Info{
		Name:         "Alarm",
		Manufacturer: manufacturer,
		Model:        "UniFi Protect",
		Firmware:     version,
	}, cfg, fs)

	server, err := hap.NewServer(fs, alarm.A)
	if err != nil {
		log.Fatal("fail to create server", "error", err)
	}
	server.Addr = cfg.Address

	server.ServeMux().HandleFunc("/trigger", func(w http.ResponseWriter, r *http.Request) {
		if cfg.TriggerToken != "" {
			token := r.URL.Query().Get("token")
			if h := r.Header.Get("Authorization"); strings.HasPrefix(h, "Bearer ") {
				token = strings.TrimPrefix(h, "Bearer ")
			}
			if subtle.ConstantTimeCompare([]byte(token), []byte(cfg.TriggerToken)) != 1 {
				log.Warn("trigger request with invalid token", "addr", r.RemoteAddr)
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
		}
		if !alarm.Trigger() {
			log.Info("ignoring trigger: alarm is disarmed", "addr", r.RemoteAddr)
			_, _ = io.WriteString(w, "ignored: alarm is disarmed\n")
			return
		}
		log.Warn("alarm triggered", "addr", r.RemoteAddr)
		_, _ = io.WriteString(w, "triggered\n")
	})

	server.ServeMux().HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{
			"current": stateName(alarm.SecuritySystem.SecuritySystemCurrentState.Value()),
			"target":  stateName(alarm.SecuritySystem.SecuritySystemTargetState.Value()),
		})
	})

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	signal.Notify(c, syscall.SIGTERM)

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		<-c
		log.Info("stopping server")
		signal.Stop(c)
		cancel()
	}()

	log.Info("starting server", "addr", server.Addr)
	if err := server.ListenAndServe(ctx); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Error("failed to close server", "err", err)
	}
}
