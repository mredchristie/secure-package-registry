package main

import (
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	"git.duti.dev/secure-package-registry/pkg/logger"
)

func main() {
	log := logger.WithComponent("reg-proxy")
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

			if !isFound || token != "securetoken" {
				log.Info().Msgf("Blocking Request: %s %s, Invalid Token: %s", req.Method, req.URL.Path, token)
				http.Error(wri, "Forbidden", http.StatusForbidden)
				return
			}

			log.Info().Msgf("Accepting Request: %s %s, token: %s", req.Method, req.URL.Path, token)
			req.Header.Del("Authorization")
			proxy.ServeHTTP(wri, req)
		})

	log.Info().Msg("Starting proxy server on :7002")
	err = http.ListenAndServe(":7002", handler)
	log.Fatal().Err(err).Msg("proxy server failed")
}
