package config

import "os"

// GetEnv returns environment variable or fallback
func GetEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// MsSQLDSN builds the SQL Server DSN connection string
func MsSQLDSN() string {
	// Default fallback connection (change only Database name if needed)
	return GetEnv(
		"DATABASE_URL",
		"Server=94.249.213.61,1433;Database=GatherUpDB;User Id=sa;Password=Iniyal@14;TrustServerCertificate=true;Encrypt=false;",
	)
}

func JwtSecret() string {
	return GetEnv("JWT_SECRET", "devsecret")
}

func ServerPort() string {
	return GetEnv("PORT", "8080")
}

func Fast2SMSKey() string {
	return GetEnv("FAST2SMS_API_KEY", "hA1W8oqiueCAU3DYBOExvzpRLTCiVR1wgv5gL3Gf8u3SKCCmhovuRS7Iy1kc")
}
