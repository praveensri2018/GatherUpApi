package main

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
		"Server=94.249.213.96,1433;Database=GatherUpDB;User Id=sa;Password=Sivanya@2025;TrustServerCertificate=true;Encrypt=false;",
	)
}

func JwtSecret() string {
	return GetEnv("JWT_SECRET", "devsecret")
}

func ServerPort() string {
	return GetEnv("PORT", "8080")
}
