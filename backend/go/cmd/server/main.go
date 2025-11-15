/* Place: backend/go/cmd/server/main.go */
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

	_ "github.com/denisenkom/go-mssqldb"
)

func main() {
	cfg := config.Load()

	dbConn, err := db.Connect(cfg.DSN)
	if err != nil {
		log.Fatalf("db connect failed: %v", err)
	}
	defer dbConn.Close()

	userRepo := repository.NewUserRepo(dbConn)
	jwtMgr := auth.NewJWTManager(cfg.JWTSecret, cfg.AccessTokenTTL)

	authCfg := &service.AuthConfig{
		BcryptCost:        cfg.BcryptCost,
		RefreshTokenBytes: cfg.RefreshTokenBytes,
		RefreshTTL:        cfg.RefreshTokenTTL,
	}
	authSvc := service.NewAuthService(userRepo, jwtMgr, authCfg)

	handler := api.WireRouter(userRepo, jwtMgr, authSvc)

	srv := &http.Server{
		Addr:         cfg.ServerAddr,
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
	log.Printf("starting server on %s", cfg.ServerAddr)
	if err := srv.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}
