package main

import (
	"flag"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
)

func main() {
	listen := flag.String("listen", "127.0.0.1:7790", "listen address")
	proxyTarget := flag.String("proxy", "http://127.0.0.1:6674", "extension proxy target")
	cipherDir := flag.String("cipher-dir", "/tmp/workproof-ciphertexts", "ciphertext directory")
	flag.Parse()

	target, err := url.Parse(*proxyTarget)
	if err != nil {
		log.Fatalf("parse proxy target: %v", err)
	}
	rp := httputil.NewSingleHostReverseProxy(target)
	fs := http.FileServer(http.Dir(*cipherDir))

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/info",
			r.URL.Path == "/instruction",
			strings.HasPrefix(r.URL.Path, "/action/"),
			strings.HasPrefix(r.URL.Path, "/wallet/"),
			strings.HasPrefix(r.URL.Path, "/state"):
			rp.ServeHTTP(w, r)
		default:
			fs.ServeHTTP(w, r)
		}
	})

	log.Printf("workproof gateway listening on %s, proxy=%s, cipher-dir=%s", *listen, *proxyTarget, *cipherDir)
	log.Fatal(http.ListenAndServe(*listen, nil))
}
