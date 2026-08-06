package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

// Config holds all runtime configuration loaded from environment (and .env).
type Config struct {
	HTTPAddr string

	CMSDBDSN string

	SupabaseURL   string
	SupabaseKey   string
	StorageBucket string
	// StorageBaseURL derives from SupabaseURL, e.g.
	// https://<ref>.supabase.co/storage/v1/object/public/<bucket>
	storageBase string
}

// Load reads .env (if present) then environment variables.
func Load() (*Config, error) {
	// .env is optional; missing file is fine.
	_ = godotenv.Load()

	cfg := &Config{
		HTTPAddr:      get("HTTP_ADDR", "127.0.0.1:8080"),
		CMSDBDSN:      os.Getenv("CMS_DB_DSN"),
		SupabaseURL:   os.Getenv("SUPABASE_URL"),
		SupabaseKey:   os.Getenv("SUPABASE_SERVICE_KEY"),
		StorageBucket: get("SUPABASE_STORAGE_BUCKET", "checkut"),
	}

	var missing []string
	if cfg.CMSDBDSN == "" {
		missing = append(missing, "CMS_DB_DSN")
	}
	if cfg.SupabaseURL == "" {
		missing = append(missing, "SUPABASE_URL")
	}
	if cfg.SupabaseKey == "" {
		missing = append(missing, "SUPABASE_SERVICE_KEY")
	}
	if len(missing) > 0 {
		return nil, fmt.Errorf("missing required env config: %v", missing)
	}

	// Base URL for building storage endpoints.
	base := cfg.SupabaseURL
	if len(base) > 0 && base[len(base)-1] == '/' {
		base = base[:len(base)-1]
	}
	// rest/v1 -> storage/v1
	idx := indexOfSuffix(base, "/rest/v1")
	if idx >= 0 {
		base = base[:idx] + "/storage/v1"
	} else {
		base = base + "/storage/v1"
	}
	cfg.storageBase = base

	return cfg, nil
}

// StorageBase returns the storage API base URL.
func (c *Config) StorageBase() string { return c.storageBase }

func indexOfSuffix(s, suffix string) int {
	n := len(s)
	m := len(suffix)
	if m == 0 || m > n {
		return -1
	}
	if s[n-m:] == suffix {
		return n - m
	}
	return -1
}

func get(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
