package main

import (
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
)

func main() {
	// Parse the target backend URL
	target, err := url.Parse("http://gitea:3000")
	if err != nil {
		log.Fatal(err)
	}

	// Create a reverse proxy instance pointing to the target
	proxy := httputil.NewSingleHostReverseProxy(target)

	// Start the proxy server on port 3000
	log.Println("Starting proxy server on :7002")
	log.Fatal(http.ListenAndServe(":7002", proxy))
}
