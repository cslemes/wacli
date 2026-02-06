package config

import (
	"os"
	"path/filepath"
	"strconv"
)

type Config struct {
	StoreDir string
	AI       AIConfig
}

type AIConfig struct {
	Enabled    bool
	GroqAPIKey string
}

func Load() *Config {
	return &Config{
		StoreDir: DefaultStoreDir(),
		AI: AIConfig{
			Enabled:    getEnvBool("WACLI_AI_ENABLED", false),
			GroqAPIKey: os.Getenv("GROQ_API_KEY"),
		},
	}
}

func DefaultStoreDir() string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return ".wacli"
	}
	return filepath.Join(home, ".wacli")
}

func getEnvBool(key string, defaultValue bool) bool {
	if v := os.Getenv(key); v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			return b
		}
	}
	return defaultValue
}
