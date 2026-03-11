package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"

	"git.duti.dev/secure-package-registry/internal/gen/coredb"
	"git.duti.dev/secure-package-registry/pkg/logger"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
)

var log zerolog.Logger

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func hashToken(token string) string {
	// Placeholder for actual hashing logic
	hash := sha256.Sum256([]byte(token))
	tokenHash := hex.EncodeToString(hash[:])
	return tokenHash
}

func main() {
	log = logger.WithComponent("reg-proxy")
	ctx := context.Background()

	// Database connection string
	databaseURL := getEnv("DATABASE_URL", "postgres://postgres:postgres@core_db:5432/core?sslmode=disable")

	// Connect to database
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		log.Err(err).Msgf("Failed to connect to database")
		os.Exit(1)
	}
	defer pool.Close()

	// Test connection
	if err := pool.Ping(ctx); err != nil {
		log.Err(err).Msgf("Failed to ping database")
		os.Exit(1)
	}

	log.Info().Msg("Successfully connected to database")

	queries := coredb.New(pool)

	// Parse the target backend URL
	target, err := url.Parse("http://gitea:3000")
	if err != nil {
		log.Fatal().Msg(err.Error())
	}

	// // Create a reverse proxy instance pointing to the target
	proxy := httputil.NewSingleHostReverseProxy(target)

	proxy.Director = func(req *http.Request) {
		req.URL.Scheme = target.Scheme
		req.URL.Host = target.Host
		req.URL.Path = target.Path + req.URL.Path
	}

	handler := http.HandlerFunc(
		func(wri http.ResponseWriter, req *http.Request,
		) {
			log.Info().Msgf("Received Request: %s %s", req.Method, req.URL.Path)

			if !strings.HasPrefix(req.URL.Path, "/api/packages/") {
				proxy.ServeHTTP(wri, req)
				return
			}

			auth := req.Header.Get("Authorization")
			log.Info().Msgf("Authorization Header: %s", auth)
			token, isFound := strings.CutPrefix(auth, "Basic ")

			if !isFound {
				log.Info().Msgf("Blocking Request: %s %s, Token not found", req.Method, req.URL.Path)
				http.Error(wri, "Forbidden", http.StatusForbidden)
				return
			}

			tokenHash := hashToken(token)
			tokenUserID, err := queries.GetUserByToken(ctx, tokenHash)
			if err != nil {
				log.Info().Msgf("Blocking Request: %s %s, Invalid Token (Hash): %s", req.Method, req.URL.Path, tokenHash)
				http.Error(wri, "Forbidden", http.StatusForbidden)
				return
			}

			log.Info().Msgf("Accepting Request: %s %s, tokenHash: %s, UserID: %s", req.Method, req.URL.Path, tokenHash, tokenUserID)
			req.Header.Del("Authorization")
			proxy.ServeHTTP(wri, req)
		})

	log.Info().Msg("Starting proxy server on :7002")
	err = http.ListenAndServe(":7002", handler)
	log.Fatal().Err(err).Msg("proxy server failed")
}
