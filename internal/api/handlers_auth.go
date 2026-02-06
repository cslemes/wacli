package api

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/skip2/go-qrcode"
	"github.com/steipete/wacli/internal/app"
)

// getQRCodeHandler generates a QR code for WhatsApp pairing
// Returns the QR code as a base64-encoded PNG image
func getQRCodeHandler(a *app.App) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Check if already authenticated
		if err := a.OpenWA(); err == nil && a.WA().IsAuthed() {
			c.JSON(http.StatusConflict, gin.H{
				"error":         "already authenticated",
				"authenticated": true,
			})
			return
		}

		// Channel to receive QR code
		qrCodeChan := make(chan string, 1)
		errChan := make(chan error, 1)

		// Start connection in background with longer timeout for QR generation
		ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
		defer cancel()

		go func() {
			// Connect and get QR code, but don't wait for scanning
			err := a.Connect(context.Background(), true, func(code string) {
				select {
				case qrCodeChan <- code:
				default:
				}
			})
			if err != nil && ctx.Err() == nil {
				select {
				case errChan <- err:
				default:
				}
			}
		}()

		// Wait for QR code or error
		select {
		case code := <-qrCodeChan:
			// Generate QR code image
			png, err := qrcode.Encode(code, qrcode.Medium, 256)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "failed to generate QR code image: " + err.Error(),
				})
				return
			}

			// Return as base64-encoded PNG
			encoded := base64.StdEncoding.EncodeToString(png)
			c.JSON(http.StatusOK, gin.H{
				"qr_code":      code,
				"qr_code_png":  "data:image/png;base64," + encoded,
				"expires_in":   60, // QR codes typically expire in 60 seconds
				"instructions": "Scan this QR code with WhatsApp: Settings → Linked Devices → Link a Device",
			})

		case err := <-errChan:
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "connection failed: " + err.Error(),
			})

		case <-ctx.Done():
			c.JSON(http.StatusRequestTimeout, gin.H{
				"error": "timeout waiting for QR code",
			})
		}
	}
}

// pairWithCodeRequest is the request body for pairing with a code
type pairWithCodeRequest struct {
	PhoneNumber string `json:"phone_number" binding:"required"`
}

// pairWithCodeHandler initiates pairing with a phone number (returns a pairing code)
func pairWithCodeHandler(a *app.App) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req pairWithCodeRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
			return
		}

		// Check if already authenticated
		if err := a.OpenWA(); err == nil && a.WA().IsAuthed() {
			c.JSON(http.StatusConflict, gin.H{
				"error":         "already authenticated",
				"authenticated": true,
			})
			return
		}

		ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
		defer cancel()

		if err := a.OpenWA(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "failed to initialize WhatsApp client: " + err.Error(),
			})
			return
		}

		// For pairing, we need to connect first but the Connect method blocks if not authenticated
		// So we access the underlying client and connect directly
		waClient := a.WA()

		// Check if we can get the underlying whatsmeow client
		type clientGetter interface {
			GetClient() interface{}
		}

		if getter, ok := waClient.(clientGetter); ok {
			if client, ok := getter.GetClient().(interface {
				ConnectContext(context.Context) error
				IsConnected() bool
			}); ok && client != nil {
				// Connect if not already connected
				if !client.IsConnected() {
					if err := client.ConnectContext(ctx); err != nil {
						c.JSON(http.StatusInternalServerError, gin.H{
							"error": "failed to connect to WhatsApp: " + err.Error(),
						})
						return
					}
				}
			}
		}

		// Request pairing code
		code, err := a.WA().PairPhone(ctx, req.PhoneNumber)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "failed to request pairing code: " + err.Error(),
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"pairing_code": code,
			"phone_number": req.PhoneNumber,
			"expires_in":   300, // Pairing codes typically expire in 5 minutes
			"instructions": fmt.Sprintf("Enter this code in WhatsApp: Settings → Linked Devices → Link a Device → Link with Phone Number"),
		})
	}
}

// waitForPairingHandler waits for pairing to complete
// This should be called after getting a QR code or pairing code
func waitForPairingHandler(a *app.App) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Check current auth status
		if err := a.OpenWA(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "failed to check auth status: " + err.Error(),
			})
			return
		}

		if a.WA().IsAuthed() {
			c.JSON(http.StatusOK, gin.H{
				"authenticated": true,
				"message":       "already authenticated",
			})
			return
		}

		// Wait for authentication with timeout
		ctx, cancel := context.WithTimeout(c.Request.Context(), 120*time.Second)
		defer cancel()

		// Poll for authentication status
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				c.JSON(http.StatusRequestTimeout, gin.H{
					"authenticated": false,
					"error":         "timeout waiting for pairing",
				})
				return

			case <-ticker.C:
				if a.WA().IsAuthed() {
					c.JSON(http.StatusOK, gin.H{
						"authenticated": true,
						"message":       "pairing successful",
					})
					return
				}
			}
		}
	}
}

// logoutHandler logs out and invalidates the session
func logoutHandler(a *app.App) gin.HandlerFunc {
	return func(c *gin.Context) {
		if err := a.EnsureAuthed(); err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "not authenticated",
			})
			return
		}

		ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
		defer cancel()

		if err := a.Connect(ctx, false, nil); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "failed to connect: " + err.Error(),
			})
			return
		}

		if err := a.WA().Logout(ctx); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "logout failed: " + err.Error(),
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"logged_out": true,
			"message":    "successfully logged out",
		})
	}
}
