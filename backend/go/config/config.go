/* Place: backend/go/config/config.go */
package config

import (
	"log"
	"strconv"
	"time"
)

// AppConfig collects runtime config values.
type AppConfig struct {
	DSN               string
	JWTSecret         string
	AccessTokenTTL    time.Duration
	RefreshTokenTTL   time.Duration
	BcryptCost        int
	ServerAddr        string
	RefreshTokenBytes int
}

func Load() *AppConfig {
	c := &AppConfig{
		DSN:               MsSQLDSN(),
		JWTSecret:         JwtSecret(),
		AccessTokenTTL:    getenvDuration("ACCESS_TOKEN_TTL", 30*24*time.Hour),
		RefreshTokenTTL:   getenvDuration("REFRESH_TOKEN_TTL", 30*24*time.Hour),
		BcryptCost:        getenvInt("BCRYPT_COST", 12),
		ServerAddr:        ":" + GetEnv("PORT", "8080"),
		RefreshTokenBytes: getenvInt("REFRESH_BYTES", 32),
	}
	if c.BcryptCost < 4 {
		log.Println("Bcrypt cost too low; bumping to 12")
		c.BcryptCost = 12
	}
	return c
}

func getenvInt(key string, fallback int) int {
	if v := GetEnv(key, ""); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return fallback
}

func getenvDuration(key string, fallback time.Duration) time.Duration {
	if v := GetEnv(key, ""); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return fallback
}
