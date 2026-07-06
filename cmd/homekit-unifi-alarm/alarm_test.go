package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/brutella/hap"
	"github.com/brutella/hap/accessory"
	"github.com/brutella/hap/characteristic"
)

func testSystem(tb testing.TB, cfg Config) *SecuritySystem {
	tb.Helper()
	if cfg.WebhookMethod == "" {
		cfg.WebhookMethod = http.MethodGet
	}
	if cfg.WebhookTimeout == 0 {
		cfg.WebhookTimeout = time.Second
	}
	return NewSecuritySystem(accessory.Info{
		Name: "Alarm",
	}, cfg, hap.NewFsStore(tb.TempDir()))
}

func TestDefaultsToDisarmed(t *testing.T) {
	alarm := testSystem(t, Config{})
	if v := alarm.SecuritySystem.SecuritySystemCurrentState.Value(); v != characteristic.SecuritySystemCurrentStateDisarmed {
		t.Errorf("expected disarmed, got %s", stateName(v))
	}
	if alarm.Trigger() {
		t.Error("expected trigger to be ignored while disarmed")
	}
}

func TestArmTriggerDisarm(t *testing.T) {
	var gotPaths []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-API-KEY") != "test-key" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		gotPaths = append(gotPaths, r.URL.Path)
	}))
	t.Cleanup(srv.Close)

	alarm := testSystem(t, Config{
		DisarmWebhook:  srv.URL + "/disarm",
		ArmAwayWebhook: srv.URL + "/away",
		UnifiAPIKey:    "test-key",
	})

	if _, code := alarm.updateHandler(characteristic.SecuritySystemTargetStateAwayArm, nil); code != hap.JsonStatusSuccess {
		t.Fatalf("expected success, got %d", code)
	}
	if v := alarm.SecuritySystem.SecuritySystemCurrentState.Value(); v != characteristic.SecuritySystemCurrentStateAwayArm {
		t.Errorf("expected armed away, got %s", stateName(v))
	}

	if !alarm.Trigger() {
		t.Error("expected trigger to fire while armed")
	}
	if v := alarm.SecuritySystem.SecuritySystemCurrentState.Value(); v != characteristic.SecuritySystemCurrentStateAlarmTriggered {
		t.Errorf("expected triggered, got %s", stateName(v))
	}

	if _, code := alarm.updateHandler(characteristic.SecuritySystemTargetStateDisarm, nil); code != hap.JsonStatusSuccess {
		t.Fatalf("expected success, got %d", code)
	}
	if v := alarm.SecuritySystem.SecuritySystemCurrentState.Value(); v != characteristic.SecuritySystemCurrentStateDisarmed {
		t.Errorf("expected disarmed, got %s", stateName(v))
	}

	if want := []string{"/away", "/disarm"}; len(gotPaths) != 2 || gotPaths[0] != want[0] || gotPaths[1] != want[1] {
		t.Errorf("expected webhooks %v, got %v", want, gotPaths)
	}
}

func TestRestoresState(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {}))
	t.Cleanup(srv.Close)

	cfg := Config{
		DisarmWebhook:   srv.URL + "/disarm",
		ArmNightWebhook: srv.URL + "/night",
		WebhookMethod:   http.MethodGet,
		WebhookTimeout:  time.Second,
	}
	store := hap.NewFsStore(t.TempDir())

	alarm := NewSecuritySystem(accessory.Info{Name: "Alarm"}, cfg, store)
	if _, code := alarm.updateHandler(characteristic.SecuritySystemTargetStateNightArm, nil); code != hap.JsonStatusSuccess {
		t.Fatalf("expected success, got %d", code)
	}

	restored := NewSecuritySystem(accessory.Info{Name: "Alarm"}, cfg, store)
	if v := restored.SecuritySystem.SecuritySystemCurrentState.Value(); v != characteristic.SecuritySystemCurrentStateNightArm {
		t.Errorf("expected armed night, got %s", stateName(v))
	}
}

func TestMissingWebhook(t *testing.T) {
	alarm := testSystem(t, Config{})
	if _, code := alarm.updateHandler(characteristic.SecuritySystemTargetStateStayArm, nil); code != hap.JsonStatusResourceDoesNotExist {
		t.Errorf("expected resource does not exist, got %d", code)
	}
	if v := alarm.SecuritySystem.SecuritySystemCurrentState.Value(); v != characteristic.SecuritySystemCurrentStateDisarmed {
		t.Errorf("expected disarmed, got %s", stateName(v))
	}
}

func TestWebhookFailure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	t.Cleanup(srv.Close)

	alarm := testSystem(t, Config{ArmStayWebhook: srv.URL + "/stay"})
	if _, code := alarm.updateHandler(characteristic.SecuritySystemTargetStateStayArm, nil); code != hap.JsonStatusResourceBusy {
		t.Errorf("expected resource busy, got %d", code)
	}
	if v := alarm.SecuritySystem.SecuritySystemCurrentState.Value(); v != characteristic.SecuritySystemCurrentStateDisarmed {
		t.Errorf("expected disarmed, got %s", stateName(v))
	}
}
