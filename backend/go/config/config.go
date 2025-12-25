/* Place: backend/go/config/config.go */
package config

import (
	"log"
	"strconv"
	"time"
)

type AppConfig struct {
	DSN               string
	JWTSecret         string
	AccessTokenTTL    time.Duration
	RefreshTokenTTL   time.Duration
	BcryptCost        int
	ServerAddr        string
	RefreshTokenBytes int
	Fast2SMSKey       string
	R2AccountID       string
	R2AccessKey       string
	R2SecretKey       string
	R2Bucket          string
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

		Fast2SMSKey: GetEnv(
			"FAST2SMS_API_KEY",
			"hA1W8oqiueCAU3DYBOExvzpRLTCiVR1wgv5gL3Gf8u3SKCCmhovuRS7Iy1kc",
		),

		R2AccountID: GetEnv("R2_ACCOUNT_ID", "8c45e84e66b0ef932a97407d1dc5f022"),
		R2AccessKey: GetEnv("R2_ACCESS_KEY", "17cd993626d63226ab7da82792e3b80e"),
		R2SecretKey: GetEnv("R2_SECRET_KEY", "76a97f8a4d4b2e29b27ff56751dd079814872e234886fa0107f60c04cea7ac14"),
		R2Bucket:    GetEnv("R2_BUCKET", "gatherup"),
	}

	if c.BcryptCost < 4 {
		log.Println("Bcrypt cost too low; bumping to 12")
		c.BcryptCost = 12
	}

	if c.R2AccountID == "" || c.R2AccessKey == "" || c.R2SecretKey == "" {
		log.Println("⚠️ WARNING: Cloudflare R2 config missing (uploads will fail)")
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
