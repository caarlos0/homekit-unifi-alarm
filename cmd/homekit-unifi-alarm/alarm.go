package main

import (
	"crypto/tls"
	"fmt"
	"net/http"

	"github.com/brutella/hap"
	"github.com/brutella/hap/accessory"
	"github.com/brutella/hap/characteristic"
	"github.com/brutella/hap/service"
)

// stateKey is the key used to persist the alarm state in the hap store, so
// the state survives restarts.
const stateKey = "unifi-alarm-state"

type SecuritySystem struct {
	*accessory.A
	SecuritySystem *service.SecuritySystem

	cfg    Config
	client *http.Client
	store  hap.Store
}

func NewSecuritySystem(info accessory.Info, cfg Config, store hap.Store) *SecuritySystem {
	a := &SecuritySystem{
		cfg:   cfg,
		store: store,
		client: &http.Client{
			Timeout: cfg.WebhookTimeout,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: cfg.Insecure, //nolint:gosec
				},
			},
		},
	}
	a.A = accessory.New(info, accessory.TypeSecuritySystem)

	a.SecuritySystem = service.NewSecuritySystem()
	a.AddS(a.SecuritySystem.S)

	a.SecuritySystem.SecuritySystemTargetState.SetValueRequestFunc = a.updateHandler

	a.restore()

	return a
}

// restore loads the last known state from the store, defaulting to disarmed,
// as the characteristics otherwise default to armed stay.
func (a *SecuritySystem) restore() {
	state := characteristic.SecuritySystemCurrentStateDisarmed
	if b, err := a.store.Get(stateKey); err == nil && len(b) == 1 &&
		int(b[0]) >= characteristic.SecuritySystemCurrentStateStayArm &&
		int(b[0]) <= characteristic.SecuritySystemCurrentStateAlarmTriggered {
		state = int(b[0])
	}
	if state != characteristic.SecuritySystemCurrentStateAlarmTriggered {
		_ = a.SecuritySystem.SecuritySystemTargetState.SetValue(state)
	}
	_ = a.SecuritySystem.SecuritySystemCurrentState.SetValue(state)
	log.Info("restored state", "state", stateName(state))
}

// Trigger sets the current state to triggered, unless the alarm is disarmed.
func (a *SecuritySystem) Trigger() bool {
	if a.SecuritySystem.SecuritySystemCurrentState.Value() ==
		characteristic.SecuritySystemCurrentStateDisarmed {
		return false
	}
	a.setCurrentState(characteristic.SecuritySystemCurrentStateAlarmTriggered)
	return true
}

func (a *SecuritySystem) setCurrentState(state int) {
	if err := a.SecuritySystem.SecuritySystemCurrentState.SetValue(state); err != nil {
		log.Error("could not set current state", "state", stateName(state), "err", err)
		return
	}
	if err := a.store.Set(stateKey, []byte{byte(state)}); err != nil {
		log.Error("could not persist state", "state", stateName(state), "err", err)
	}
	log.Info("set current state", "state", stateName(state))
}

func (a *SecuritySystem) updateHandler(
	v interface{},
	_ *http.Request,
) (response interface{}, code int) {
	state := v.(int)
	name, url := a.cfg.webhook(state)
	if url == "" {
		log.Error("no webhook configured for state", "state", name)
		return nil, hap.JsonStatusResourceDoesNotExist
	}
	log.Info("state change requested", "state", name)
	if err := a.call(url); err != nil {
		log.Error("webhook failed", "state", name, "err", err)
		return nil, hap.JsonStatusResourceBusy
	}
	a.setCurrentState(state)
	return nil, hap.JsonStatusSuccess
}

func (a *SecuritySystem) call(url string) error {
	req, err := http.NewRequest(a.cfg.WebhookMethod, url, nil)
	if err != nil {
		return fmt.Errorf("could not create request: %w", err)
	}
	if a.cfg.UnifiAPIKey != "" {
		req.Header.Set("X-API-KEY", a.cfg.UnifiAPIKey)
	}
	resp, err := a.client.Do(req)
	if err != nil {
		return fmt.Errorf("could not call webhook: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("webhook returned %s", resp.Status)
	}
	return nil
}
