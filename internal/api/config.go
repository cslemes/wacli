package api

type Config struct {
	Host        string
	Port        int
	StoreDir    string
	APIKeys     []string
	ReleaseMode bool
	AI          AIConfig
}

type AIConfig struct {
	Enabled    bool
	GroqAPIKey string
}
