package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/steipete/wacli/internal/api"
	"github.com/steipete/wacli/internal/app"
	"github.com/steipete/wacli/internal/config"
)

var version = "dev"

func main() {
	// Load .env file if it exists (ignore error if file doesn't exist)
	_ = godotenv.Load()

	cfg := loadConfig()

	storeDir := cfg.StoreDir
	if storeDir == "" {
		storeDir = config.DefaultStoreDir()
	}

	// Initialize the app
	appInstance, err := app.New(app.Options{
		StoreDir: storeDir,
		Version:  version,
		JSON:     true,
	})
	if err != nil {
		log.Fatalf("Failed to initialize app: %v", err)
	}

	// Setup Gin router
	if cfg.ReleaseMode {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.Default()

	// Setup routes (API key middleware applied selectively)
	api.SetupRoutes(router, appInstance, cfg)

	// Start server
	srv := &api.Server{
		Router: router,
		App:    appInstance,
		Config: cfg,
	}

	go func() {
		addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
		log.Printf("Starting wacli API server on %s", addr)
		if err := router.Run(addr); err != nil {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	srv.Shutdown(ctx)
	log.Println("Server stopped")
}

func loadConfig() *api.Config {
	apiKeys := os.Getenv("WACLI_API_KEYS")
	if apiKeys == "" {
		log.Fatal("WACLI_API_KEYS environment variable is required (comma-separated list of valid API keys)")
	}

	cfg := &api.Config{
		Host:        getEnvOrDefault("WACLI_API_HOST", "0.0.0.0"),
		Port:        getEnvIntOrDefault("WACLI_API_PORT", 8080),
		StoreDir:    os.Getenv("WACLI_STORE_DIR"),
		APIKeys:     parseAPIKeys(apiKeys),
		ReleaseMode: getEnvOrDefault("GIN_MODE", "debug") == "release",
		AI: api.AIConfig{
			Enabled:    getEnvBool("WACLI_AI_ENABLED"),
			GroqAPIKey: os.Getenv("GROQ_API_KEY"),
		},
	}

	return cfg
}

func getEnvOrDefault(key, defaultValue string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultValue
}

func getEnvIntOrDefault(key string, defaultValue int) int {
	if val := os.Getenv(key); val != "" {
		var intVal int
		if _, err := fmt.Sscanf(val, "%d", &intVal); err == nil {
			return intVal
		}
	}
	return defaultValue
}

func getEnvBool(key string) bool {
	val := os.Getenv(key)
	return val == "true" || val == "1" || val == "yes"
}

func parseAPIKeys(raw string) []string {
	keys := []string{}
	for _, key := range splitAndTrim(raw, ",") {
		if key != "" {
			keys = append(keys, key)
		}
	}
	return keys
}

func splitAndTrim(s, sep string) []string {
	parts := []string{}
	for _, p := range split(s, sep) {
		if trimmed := trim(p); trimmed != "" {
			parts = append(parts, trimmed)
		}
	}
	return parts
}

func split(s, sep string) []string {
	result := []string{}
	current := ""
	for _, c := range s {
		if string(c) == sep {
			result = append(result, current)
			current = ""
		} else {
			current += string(c)
		}
	}
	if current != "" || len(result) > 0 {
		result = append(result, current)
	}
	return result
}

func trim(s string) string {
	start := 0
	end := len(s)
	for start < end && (s[start] == ' ' || s[start] == '\t' || s[start] == '\n' || s[start] == '\r') {
		start++
	}
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t' || s[end-1] == '\n' || s[end-1] == '\r') {
		end--
	}
	return s[start:end]
}
