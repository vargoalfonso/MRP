package config

import "strconv"

// RaigineConfig holds the connection settings for the Raigine automation
// platform (crp-backend). It is loaded independently from the main Config so
// the integration can be dropped in without touching the core config struct.
//
// Environment variables:
//
//	RAIGINE_API_BASE_URL      base URL of crp-backend, e.g. http://localhost:8080
//	RAIGINE_API_TOKEN         optional long-lived bearer/machine token
//	RAIGINE_API_EMAIL         service-account email (used when token is empty)
//	RAIGINE_API_PASSWORD      service-account password (used when token is empty)
//	RAIGINE_API_TIMEOUT_SECS  per-request timeout in seconds (default 30)
//	RAIGINE_DEFAULT_FOLDER_ID default folder id applied to schedules/filters
type RaigineConfig struct {
	BaseURL         string
	Token           string
	Email           string
	Password        string
	TimeoutSeconds  int
	DefaultFolderID string
}

// Enabled reports whether enough configuration exists to reach crp-backend.
func (r RaigineConfig) Enabled() bool {
	return r.BaseURL != "" && (r.Token != "" || (r.Email != "" && r.Password != ""))
}

// LoadRaigineConfig reads the RAIGINE_* environment variables.
func LoadRaigineConfig() RaigineConfig {
	timeout := 30
	if v := getEnv("RAIGINE_API_TIMEOUT_SECS", ""); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			timeout = n
		}
	}
	return RaigineConfig{
		BaseURL:         getEnv("RAIGINE_API_BASE_URL", ""),
		Token:           getEnv("RAIGINE_API_TOKEN", ""),
		Email:           getEnv("RAIGINE_API_EMAIL", ""),
		Password:        getEnv("RAIGINE_API_PASSWORD", ""),
		TimeoutSeconds:  timeout,
		DefaultFolderID: getEnv("RAIGINE_DEFAULT_FOLDER_ID", ""),
	}
}
