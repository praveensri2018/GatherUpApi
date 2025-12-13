package main

import (
	"log"
	"net/http"
	"time"

	"gatherup/api"
	"gatherup/auth"
	"gatherup/config"
	"gatherup/db"
	"gatherup/repository"
	"gatherup/service"
	"gatherup/storage"

	_ "github.com/denisenkom/go-mssqldb"
)

func main() {
	/* ---------------- CONFIG ---------------- */

	cfg := config.Load()

	/* ---------------- DATABASE ---------------- */

	dbConn, err := db.Connect(cfg.DSN)
	if err != nil {
		log.Fatalf("db connect failed: %v", err)
	}
	defer dbConn.Close()

	/* ---------------- REPOSITORIES ---------------- */

	// Auth + User only
	userRepo := repository.NewUserRepo(dbConn, nil, nil)

	// Phase-2 (posts, feed, comments, social)
	socialRepo := repository.NewSocialRepo(dbConn)

	/* ---------------- AUTH ---------------- */

	jwtMgr := auth.NewJWTManager(cfg.JWTSecret, cfg.AccessTokenTTL)

	authCfg := &service.AuthConfig{
		BcryptCost:        cfg.BcryptCost,
		RefreshTokenBytes: cfg.RefreshTokenBytes,
		RefreshTTL:        cfg.RefreshTokenTTL,
		OTPDigits:         6,
		OTPTTL:            10 * time.Minute,
	}

	smsClient := service.NewFast2SMSClient(cfg.Fast2SMSKey)
	authSvc := service.NewAuthService(userRepo, jwtMgr, authCfg, smsClient)

	/* ---------------- STORAGE (Cloudflare R2) ---------------- */

	r2Client, err := storage.NewR2Client(
		cfg.R2AccountID,
		cfg.R2AccessKey,
		cfg.R2SecretKey,
		cfg.R2Bucket,
	)
	if err != nil {
		log.Fatalf("R2 init failed: %v", err)
	}

	/* ---------------- ROUTER ---------------- */

	handler := api.WireRouter(
		userRepo,
		socialRepo,
		r2Client,
		jwtMgr,
		authSvc,
	)

	/* ---------------- HTTP SERVER ---------------- */

	srv := &http.Server{
		Addr:         cfg.ServerAddr,
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	log.Printf("🚀 GatherUp API started on %s", cfg.ServerAddr)

	if err := srv.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}
