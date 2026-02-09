package api

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/steipete/wacli/internal/app"
	"github.com/steipete/wacli/internal/wa"
)

// GrafanaAlert represents the incoming Grafana webhook payload
type GrafanaAlert struct {
	Receiver          string            `json:"receiver"`
	Status            string            `json:"status"`
	State             string            `json:"state"`
	Title             string            `json:"title"`
	Message           string            `json:"message"`
	ExternalURL       string            `json:"externalURL"`
	Version           string            `json:"version"`
	GroupKey          string            `json:"groupKey"`
	OrgID             int64             `json:"orgId"`
	GroupLabels       map[string]string `json:"groupLabels"`
	CommonLabels      map[string]string `json:"commonLabels"`
	CommonAnnotations map[string]string `json:"commonAnnotations"`
	TruncatedAlerts   int               `json:"truncatedAlerts"`
	Alerts            []struct {
		Status       string                 `json:"status"`
		Labels       map[string]string      `json:"labels"`
		Annotations  map[string]string      `json:"annotations"`
		StartsAt     string                 `json:"startsAt"`
		EndsAt       string                 `json:"endsAt"`
		GeneratorURL string                 `json:"generatorURL"`
		Fingerprint  string                 `json:"fingerprint"`
		SilenceURL   string                 `json:"silenceURL"`
		DashboardURL string                 `json:"dashboardURL"`
		PanelURL     string                 `json:"panelURL"`
		Values       map[string]interface{} `json:"values"`
	} `json:"alerts"`

	// Legacy fields for older Grafana versions
	RuleURL     string            `json:"ruleUrl"`
	EvalMatches []GrafanaMatch    `json:"evalMatches"`
	Tags        map[string]string `json:"tags"`
}

type GrafanaMatch struct {
	Value  interface{} `json:"value"`
	Metric string      `json:"metric"`
	Tags   interface{} `json:"tags"`
}

// webhookGrafanaHandler handles Grafana webhook alerts
func webhookGrafanaHandler(app *app.App, cfg *Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Read raw body for debugging
		bodyBytes, _ := c.GetRawData()
		rawPayload := string(bodyBytes)
		fmt.Printf("DEBUG: Received webhook payload (%d bytes):\n%s\n", len(bodyBytes), rawPayload)

		// Try to parse as Grafana JSON; if it fails, continue with raw body as message
		var alert GrafanaAlert
		var parseErr error
		if len(bodyBytes) > 0 {
			c.Request.Body = io.NopCloser(strings.NewReader(rawPayload))
			parseErr = c.ShouldBindJSON(&alert)
			if parseErr != nil {
				fmt.Printf("WARN: Failed to parse as Grafana JSON (will try fallback): %v\n", parseErr)
			} else {
				fmt.Printf("DEBUG: Parsed alert - Title: %s, Status: %s, State: %s, Alerts count: %d\n",
					alert.Title, alert.Status, alert.State, len(alert.Alerts))
			}
		}

		// Get recipient from multiple sources (priority order):
		// 1. Query parameter ?to=
		// 2. HTTP header X-WhatsApp-To
		// 3. Grafana commonAnnotations.whatsapp_to
		// 4. Grafana alert annotations.whatsapp_to
		// 5. Grafana tags.whatsapp_to
		recipient := c.Query("to")
		if recipient == "" {
			recipient = c.GetHeader("X-WhatsApp-To")
		}
		if recipient == "" && alert.CommonAnnotations != nil {
			recipient = alert.CommonAnnotations["whatsapp_to"]
		}
		if recipient == "" && len(alert.Alerts) > 0 && alert.Alerts[0].Annotations != nil {
			recipient = alert.Alerts[0].Annotations["whatsapp_to"]
		}
		if recipient == "" && alert.Tags != nil {
			recipient = alert.Tags["whatsapp_to"]
		}
		if recipient == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "recipient required: add ?to=PHONE to URL, set X-WhatsApp-To header, or add whatsapp_to annotation in Grafana alert rule",
				"payload": rawPayload,
				"help":    "Example URL: /api/v1/webhook/grafana?to=5511999999999",
			})
			return
		}

		// If JSON parsing failed, use the raw body as the message (fallback for custom templates)
		if parseErr != nil {
			trimmed := strings.TrimSpace(rawPayload)
			if trimmed == "" {
				// Empty body — Grafana likely has a broken/empty Message template.
				// Send a default alert message instead of failing.
				fmt.Printf("WARN: Empty body received from Grafana. The webhook Message field in Grafana may need to be cleared. Sending default message.\n")
				trimmed = "⚠️ Grafana alert received (empty payload — clear the Message field in Grafana Webhook Contact Point to get full alert details)"
			}
			fmt.Printf("DEBUG: Using raw body as message (JSON parse failed), sending to %s\n", recipient)

			ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Minute)
			defer cancel()

			if err := app.EnsureAuthed(); err != nil {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "not authenticated: " + err.Error()})
				return
			}
			if err := app.Connect(ctx, false, nil); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "connection failed: " + err.Error()})
				return
			}

			toJID, err := wa.ParseUserOrJID(recipient)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid recipient: " + err.Error()})
				return
			}

			msgID, err := app.WA().SendText(ctx, toJID, trimmed)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "send failed: " + err.Error()})
				return
			}

			c.JSON(http.StatusOK, gin.H{
				"sent":     true,
				"to":       toJID.String(),
				"id":       msgID,
				"fallback": true,
			})
			return
		}

		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Minute)
		defer cancel()

		if err := app.EnsureAuthed(); err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "not authenticated: " + err.Error()})
			return
		}

		if err := app.Connect(ctx, false, nil); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "connection failed: " + err.Error()})
			return
		}

		toJID, err := wa.ParseUserOrJID(recipient)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid recipient: " + err.Error()})
			return
		}

		// Format the message
		message := formatGrafanaMessage(alert)

		msgID, err := app.WA().SendText(ctx, toJID, message)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "send failed: " + err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"sent":  true,
			"to":    toJID.String(),
			"id":    msgID,
			"alert": alert.Title,
		})
	}
}

// formatGrafanaMessage formats a Grafana alert into a WhatsApp message
func formatGrafanaMessage(alert GrafanaAlert) string {
	var sb strings.Builder

	// Status emoji
	emoji := "🔔"
	state := alert.State
	if state == "" {
		state = alert.Status
	}

	switch state {
	case "alerting", "firing":
		emoji = "🚨"
	case "ok", "resolved":
		emoji = "✅"
	case "no_data":
		emoji = "⚠️"
	}

	sb.WriteString(fmt.Sprintf("%s *Grafana Alert*\n\n", emoji))

	// Use title from payload or build from labels
	title := alert.Title
	if title == "" && alert.CommonLabels != nil {
		if name, ok := alert.CommonLabels["alertname"]; ok {
			title = name
		}
	}
	if title != "" {
		sb.WriteString(fmt.Sprintf("*%s*\n", title))
	}

	sb.WriteString(fmt.Sprintf("Status: %s\n\n", strings.ToUpper(state)))

	// Use message from payload
	if alert.Message != "" {
		sb.WriteString(fmt.Sprintf("%s\n\n", alert.Message))
	}

	// Add firing alerts count
	if len(alert.Alerts) > 0 {
		firingCount := 0
		for _, a := range alert.Alerts {
			if a.Status == "firing" {
				firingCount++
			}
		}
		if firingCount > 0 {
			sb.WriteString(fmt.Sprintf("*Firing:* %d alert(s)\n\n", firingCount))
		}

		// Show details of first few alerts
		maxAlerts := 3
		for i, a := range alert.Alerts {
			if i >= maxAlerts {
				sb.WriteString(fmt.Sprintf("... and %d more\n\n", len(alert.Alerts)-maxAlerts))
				break
			}

			if alertName, ok := a.Labels["alertname"]; ok {
				sb.WriteString(fmt.Sprintf("• %s", alertName))
				if instance, ok := a.Labels["instance"]; ok {
					sb.WriteString(fmt.Sprintf(" (%s)", instance))
				}
				sb.WriteString("\n")
			}

			// Show values if present
			if len(a.Values) > 0 {
				for k, v := range a.Values {
					sb.WriteString(fmt.Sprintf("  %s: %v\n", k, v))
				}
			}
		}
		sb.WriteString("\n")
	}

	// Add legacy evalMatches if present
	if len(alert.EvalMatches) > 0 {
		sb.WriteString("*Metrics:*\n")
		for _, match := range alert.EvalMatches {
			sb.WriteString(fmt.Sprintf("• %s: %v\n", match.Metric, match.Value))
		}
		sb.WriteString("\n")
	}

	// Add important labels
	if alert.CommonLabels != nil {
		important := []string{"severity", "priority", "team", "service", "namespace"}
		hasImportant := false
		for _, key := range important {
			if value, ok := alert.CommonLabels[key]; ok {
				if !hasImportant {
					sb.WriteString("*Labels:*\n")
					hasImportant = true
				}
				sb.WriteString(fmt.Sprintf("• %s: %s\n", key, value))
			}
		}
		if hasImportant {
			sb.WriteString("\n")
		}
	}

	// Add link
	if alert.ExternalURL != "" {
		sb.WriteString(fmt.Sprintf("🔗 %s", alert.ExternalURL))
	} else if alert.RuleURL != "" {
		sb.WriteString(fmt.Sprintf("🔗 %s", alert.RuleURL))
	} else if len(alert.Alerts) > 0 && alert.Alerts[0].GeneratorURL != "" {
		sb.WriteString(fmt.Sprintf("🔗 %s", alert.Alerts[0].GeneratorURL))
	}

	return sb.String()
}

// GenericWebhookRequest allows flexible webhook integration
type GenericWebhookRequest struct {
	To      string                 `json:"to" form:"to"`
	Message string                 `json:"message" form:"message"`
	Data    map[string]interface{} `json:"data"`
}

// webhookGenericHandler is a flexible webhook handler
func webhookGenericHandler(app *app.App) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req GenericWebhookRequest
		if err := c.ShouldBind(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Try to get 'to' from query if not in body
		if req.To == "" {
			req.To = c.Query("to")
		}

		if req.To == "" || req.Message == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "'to' and 'message' are required"})
			return
		}

		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Minute)
		defer cancel()

		if err := app.EnsureAuthed(); err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "not authenticated: " + err.Error()})
			return
		}

		if err := app.Connect(ctx, false, nil); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "connection failed: " + err.Error()})
			return
		}

		toJID, err := wa.ParseUserOrJID(req.To)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid recipient: " + err.Error()})
			return
		}

		msgID, err := app.WA().SendText(ctx, toJID, req.Message)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "send failed: " + err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"sent": true,
			"to":   toJID.String(),
			"id":   msgID,
		})
	}
}
