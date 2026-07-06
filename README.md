# homekit-unifi-alarm

A HomeKit security system for UniFi Protect.

UniFi doesn't have a good public API to arm/disarm anything, so this bridges
the gap using webhooks in both directions:

- **HomeKit → UniFi**: when you change the alarm state in the Home app, the
  corresponding webhook is called — point them at UniFi Alarm Manager alarms
  with a *Webhook* trigger, and use those alarms to enable/disable whatever
  automations you want.
- **UniFi → HomeKit**: an Alarm Manager alarm with a *Webhook* action can call
  the `/trigger` endpoint to set the HomeKit alarm to "triggered" (e.g. on
  motion or smart detections while armed).

Sensors, cameras, and everything else are expected to be exposed to HomeKit
by other means (e.g. [Homebridge][homebridge]); this only provides the core
alarm accessory for home automations.

[homebridge]: https://homebridge.io

## Configuration

Everything is set via environment variables:

| Variable               | Required | Default | Description                                             |
| ---------------------- | -------- | ------- | ------------------------------------------------------- |
| `DISARM_WEBHOOK`       | yes      |         | URL called when the alarm is disarmed                   |
| `ARM_AWAY_WEBHOOK`     | no       |         | URL called when arming in away mode                     |
| `ARM_STAY_WEBHOOK`     | no       |         | URL called when arming in stay/home mode                |
| `ARM_NIGHT_WEBHOOK`    | no       |         | URL called when arming in night mode                    |
| `UNIFI_API_KEY`        | no       |         | Sent as `X-API-KEY`; required by the Integration API    |
| `WEBHOOK_METHOD`       | no       | `POST`  | HTTP method used to call the webhooks                   |
| `WEBHOOK_TIMEOUT`      | no       | `10s`   | Timeout for webhook calls                               |
| `INSECURE_SKIP_VERIFY` | no       | `false` | Skip TLS verification (UniFi self-signed certificates)  |
| `TRIGGER_TOKEN`        | no       |         | If set, `/trigger` requires this token                  |
| `LISTEN`               | no       | `:9009` | Address to listen on                                    |

Arm modes without a webhook configured are rejected in the Home app.

The Alarm Manager webhook URLs look like
`https://<console>/proxy/protect/integration/v1/alarm-manager/webhook/<id>`
and require an API key: create one in UniFi under **Settings → Control Plane →
Integrations** and set it as `UNIFI_API_KEY`.

## Trigger endpoint

Point an Alarm Manager *Webhook* action at:

```
http://<host>:9009/trigger?token=<TRIGGER_TOKEN>
```

The token can also be sent as an `Authorization: Bearer` header.

Triggers are ignored while the alarm is disarmed, so you can let UniFi send
detections unconditionally and gate them here.

Setting the alarm to disarmed in the Home app clears the triggered state (and
calls `DISARM_WEBHOOK`).

## Status

`GET /` returns the current and target states as JSON:

```json
{ "current": "armed away", "target": "armed away" }
```

## Running

```console
$ DISARM_WEBHOOK="https://..." ARM_AWAY_WEBHOOK="https://..." ./homekit-unifi-alarm
```

Then add the accessory to the Home app using the pin `001-02-003`.

State (pairings and last alarm state) is kept in `./db`.
