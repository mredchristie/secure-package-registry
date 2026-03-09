package main

import (
	"fmt"
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
	// proxy := httputil.NewSingleHostReverseProxy(target)

	proxy := &httputil.ReverseProxy{
		Director: func(req *http.Request) {
			log.Info().Msg(fmt.Sprintf("Received Request: %s %s", req.Method, req.URL.Path))
			req.URL.Scheme = target.Scheme
			req.URL.Host = target.Host
			req.URL.Path = target.Path + req.URL.Path

			if strings.HasPrefix(req.URL.Path, "/api/packages/") {

				req.Header.Set("X-Block-Download", "true")
				log.Info().Msg(fmt.Sprintf("Blocking Request: %s", req.URL.Path))

			}
		},
		ModifyResponse: func(resp *http.Response) error {
			if resp.Request.Header.Get("X-Block-Download") == "true" {
				log.Println("Blocking package pull:", resp.Request.URL.Path)

				resp.StatusCode = http.StatusForbidden
				resp.Status = "403 Forbidden"
				resp.Header.Set("Content-Type", "text/plain")
				resp.Body = http.NoBody
			}

			return nil
		},
	}

	log.Info().Msg("Starting proxy server on :7002")
	err = http.ListenAndServe(":7002", proxy)
	log.Fatal().Err(err).Msg("proxy server failed")
}
