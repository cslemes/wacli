#!/bin/bash
# Test wacli Grafana webhook with realistic Uptime Kuma alert data
#
# Usage: ./test-webhook.sh [HOST:PORT] [PHONE] [API_KEY]

HOST="${1:-172.29.0.11:8090}"
PHONE="${2:-5521998940168}"
API_KEY="${3:-XsVREsdriPvQG8BvyEW7WGdyb3FYQXh5NLhIQJPgw5h3UkMpuIbA}"

echo "=== Testing wacli Grafana Webhook ==="
echo "Host:  $HOST"
echo "Phone: $PHONE"
echo ""

# Test 1: Firing alert with monitor_name labels
echo "--- Test 1: FIRING alert (2 monitors down) ---"
curl -s -w "\nHTTP Status: %{http_code}\n" \
  "http://${HOST}/api/v1/webhook/grafana?to=${PHONE}" \
  -H "Authorization: Bearer ${API_KEY}" \
  -H "Content-Type: application/json" \
  -d '{
  "receiver": "wacli-whatsapp",
  "status": "firing",
  "state": "alerting",
  "title": "[FIRING:2] Link Fora",
  "message": "",
  "externalURL": "http://172.29.0.11:3000/",
  "version": "1",
  "orgId": 1,
  "groupKey": "{}:{alertname=\"Link Fora\"}",
  "groupLabels": {"alertname": "Link Fora"},
  "commonLabels": {
    "alertname": "Link Fora",
    "job": "uptime",
    "severity": "critical"
  },
  "commonAnnotations": {
    "summary": "Monitores offline detectados"
  },
  "truncatedAlerts": 0,
  "alerts": [
    {
      "status": "firing",
      "labels": {
        "alertname": "Link Fora",
        "instance": "172.29.0.11:3001",
        "job": "uptime",
        "monitor_hostname": "172.19.2.51",
        "monitor_name": "01-INHOAIBA-TEF",
        "monitor_port": "null",
        "monitor_type": "push",
        "monitor_url": "https://"
      },
      "annotations": {
        "summary": "Monitor 01-INHOAIBA-TEF está offline"
      },
      "startsAt": "2026-02-09T17:00:00Z",
      "endsAt": "0001-01-01T00:00:00Z",
      "generatorURL": "http://172.29.0.11:3000/alerting/1/view",
      "fingerprint": "abc123",
      "silenceURL": "http://172.29.0.11:3000/alerting/silence/new",
      "dashboardURL": "",
      "panelURL": "",
      "values": {"A": 0, "C": 1}
    },
    {
      "status": "firing",
      "labels": {
        "alertname": "Link Fora",
        "instance": "172.29.0.11:3001",
        "job": "uptime",
        "monitor_hostname": "186.216.207.189",
        "monitor_name": "14-BV DE NOVAES-WIPI",
        "monitor_port": "null",
        "monitor_type": "ping",
        "monitor_url": "https://"
      },
      "annotations": {
        "summary": "Monitor 14-BV DE NOVAES-WIPI está offline"
      },
      "startsAt": "2026-02-09T17:05:00Z",
      "endsAt": "0001-01-01T00:00:00Z",
      "generatorURL": "http://172.29.0.11:3000/alerting/1/view",
      "fingerprint": "def456",
      "silenceURL": "http://172.29.0.11:3000/alerting/silence/new",
      "dashboardURL": "",
      "panelURL": "",
      "values": {"A": 0, "C": 1}
    }
  ]
}'

echo ""
echo ""

# Test 2: Resolved alert
echo "--- Test 2: RESOLVED alert (1 monitor recovered) ---"
curl -s -w "\nHTTP Status: %{http_code}\n" \
  "http://${HOST}/api/v1/webhook/grafana?to=${PHONE}" \
  -H "Authorization: Bearer ${API_KEY}" \
  -H "Content-Type: application/json" \
  -d '{
  "receiver": "wacli-whatsapp",
  "status": "resolved",
  "state": "ok",
  "title": "[RESOLVED] Link Fora",
  "message": "",
  "externalURL": "http://172.29.0.11:3000/",
  "version": "1",
  "orgId": 1,
  "groupLabels": {"alertname": "Link Fora"},
  "commonLabels": {
    "alertname": "Link Fora",
    "job": "uptime"
  },
  "commonAnnotations": {},
  "truncatedAlerts": 0,
  "alerts": [
    {
      "status": "resolved",
      "labels": {
        "alertname": "Link Fora",
        "instance": "172.29.0.11:3001",
        "monitor_hostname": "172.19.2.51",
        "monitor_name": "01-INHOAIBA-TEF",
        "monitor_type": "push"
      },
      "annotations": {
        "summary": "Monitor 01-INHOAIBA-TEF voltou ao normal"
      },
      "startsAt": "2026-02-09T17:00:00Z",
      "endsAt": "2026-02-09T18:00:00Z",
      "generatorURL": "http://172.29.0.11:3000/alerting/1/view",
      "fingerprint": "abc123",
      "values": {}
    }
  ]
}'

echo ""
echo ""
echo "=== Done ==="
