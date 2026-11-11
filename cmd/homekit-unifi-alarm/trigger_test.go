package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/brutella/hap/characteristic"
)

func TestTriggerAuthentication(t *testing.T) {
	for _, test := range []struct {
		name        string
		configured  string
		query       string
		bearer      string
		status      int
		shouldAlarm bool
	}{
		{"disabled without configuration", "", "", "", http.StatusServiceUnavailable, false},
		{"missing credentials", "synthetic-token", "", "", http.StatusUnauthorized, false},
		{"incorrect credentials", "synthetic-token", "", "wrong", http.StatusUnauthorized, false},
		{"query token", "synthetic-token", "synthetic-token", "", http.StatusOK, true},
		{"bearer token", "synthetic-token", "", "synthetic-token", http.StatusOK, true},
		{"bearer takes precedence", "synthetic-token", "synthetic-token", "wrong", http.StatusUnauthorized, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			alarm := testSystem(t, Config{})
			if err := alarm.SecuritySystem.SecuritySystemCurrentState.SetValue(characteristic.SecuritySystemCurrentStateAwayArm); err != nil {
				t.Fatal(err)
			}
			request := httptest.NewRequest(http.MethodGet, "/trigger?token="+test.query, nil)
			if test.bearer != "" {
				request.Header.Set("Authorization", "Bearer "+test.bearer)
			}
			recorder := httptest.NewRecorder()
			triggerHandler(alarm, test.configured).ServeHTTP(recorder, request)
			if recorder.Code != test.status {
				t.Fatalf("expected HTTP %d, got %d", test.status, recorder.Code)
			}
			wantState := characteristic.SecuritySystemCurrentStateAwayArm
			if test.shouldAlarm {
				wantState = characteristic.SecuritySystemCurrentStateAlarmTriggered
			}
			if got := alarm.SecuritySystem.SecuritySystemCurrentState.Value(); got != wantState {
				t.Fatalf("expected state %d, got %d", wantState, got)
			}
		})
	}
}
