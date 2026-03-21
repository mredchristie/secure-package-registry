package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strconv"
	"strings"

	"git.duti.dev/secure-package-registry/internal/gen/coredb"
	"git.duti.dev/secure-package-registry/pkg/config"
	"git.duti.dev/secure-package-registry/pkg/logger"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
)

var log zerolog.Logger

func hashToken(token string) string {
	// Placeholder for actual hashing logic
	hash := sha256.Sum256([]byte(token))
	tokenHash := hex.EncodeToString(hash[:])
	return tokenHash
}

func main() {
	log = logger.WithComponent("reg-proxy")
	ctx := context.Background()
	cfg := config.NewCoreConfig(config.WithEnv())

	// Connect to database
	pool, err := pgxpool.New(ctx, cfg.ReverseProxy.DatabaseURL)
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

	// Some metadata uses gitea:3000 instead of localhost:7002, so pulling breaks
	// This rewrites anything like that
	proxy.ModifyResponse = func(resp *http.Response) error {
		ct := resp.Header.Get("Content-Type")

		// Only modify npm metadata JSON
		if !strings.Contains(ct, "application/json") {
			return nil
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		err = resp.Body.Close()
		if err != nil {
			return err
		}

		// Replace internal hostname with proxy host
		rewritten := bytes.ReplaceAll(
			body,
			[]byte(cfg.ReverseProxy.InternalURL),
			[]byte(cfg.ReverseProxy.ExternalURL),
		)

		resp.Body = io.NopCloser(bytes.NewBuffer(rewritten))
		resp.ContentLength = int64(len(rewritten))
		resp.Header.Set("Content-Length", strconv.Itoa(len(rewritten)))

		return nil
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
