package main

import (
	"time"

	"github.com/brutella/hap/characteristic"
)

type Config struct {
	DisarmWebhook   string `env:"DISARM_WEBHOOK,notEmpty"`
	ArmStayWebhook  string `env:"ARM_STAY_WEBHOOK"`
	ArmAwayWebhook  string `env:"ARM_AWAY_WEBHOOK"`
	ArmNightWebhook string `env:"ARM_NIGHT_WEBHOOK"`

	UnifiAPIKey    string        `env:"UNIFI_API_KEY"`
	WebhookMethod  string        `env:"WEBHOOK_METHOD"       envDefault:"POST"`
	WebhookTimeout time.Duration `env:"WEBHOOK_TIMEOUT"      envDefault:"10s"`
	Insecure       bool          `env:"INSECURE_SKIP_VERIFY"`

	TriggerToken string `env:"TRIGGER_TOKEN"`
	Address      string `env:"LISTEN" envDefault:":9009"`
}

// webhook returns the state name and the webhook URL for the given target
// state.
func (c Config) webhook(state int) (string, string) {
	switch state {
	case characteristic.SecuritySystemTargetStateStayArm:
		return "arm stay", c.ArmStayWebhook
	case characteristic.SecuritySystemTargetStateAwayArm:
		return "arm away", c.ArmAwayWebhook
	case characteristic.SecuritySystemTargetStateNightArm:
		return "arm night", c.ArmNightWebhook
	case characteristic.SecuritySystemTargetStateDisarm:
		return "disarm", c.DisarmWebhook
	default:
		return "unknown", ""
	}
}

func stateName(state int) string {
	switch state {
	case characteristic.SecuritySystemCurrentStateStayArm:
		return "armed stay"
	case characteristic.SecuritySystemCurrentStateAwayArm:
		return "armed away"
	case characteristic.SecuritySystemCurrentStateNightArm:
		return "armed night"
	case characteristic.SecuritySystemCurrentStateDisarmed:
		return "disarmed"
	case characteristic.SecuritySystemCurrentStateAlarmTriggered:
		return "triggered"
	default:
		return "unknown"
	}
}
