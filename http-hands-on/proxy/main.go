package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"
	"time"
)

func main() {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		log.Fatal("JWT_SECRET is empty")
	}
	upstream := os.Getenv("UPSTREAM")
	if upstream == "" {
		upstream = "http://httpbin:8080"
	}
	target, err := url.Parse(upstream)
	if err != nil {
		log.Fatal(err)
	}

	proxy := httputil.NewSingleHostReverseProxy(target)
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		http.Error(w, "bad gateway", http.StatusBadGateway)
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if !valid(bearer(r.Header.Get("Authorization")), secret) {
			w.Header().Set("WWW-Authenticate", "Bearer")
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		proxy.ServeHTTP(w, r)
	})

	log.Fatal(http.ListenAndServe(":80", nil))
}

func bearer(h string) string {
	const p = "Bearer "
	if len(h) < len(p) || !strings.EqualFold(h[:len(p)], p) {
		return ""
	}
	return strings.TrimSpace(h[len(p):])
}

func valid(token, secret string) bool {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return false
	}
	headerJSON, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return false
	}
	var header struct {
		Alg string `json:"alg"`
	}
	if json.Unmarshal(headerJSON, &header) != nil || header.Alg != "HS256" {
		return false
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(parts[0] + "." + parts[1]))
	sig, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil || !hmac.Equal(sig, mac.Sum(nil)) {
		return false
	}
	payloadJSON, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return false
	}
	var payload struct {
		Exp *int64 `json:"exp"`
	}
	if json.Unmarshal(payloadJSON, &payload) != nil || payload.Exp == nil {
		return false
	}
	return time.Now().Unix() < *payload.Exp
}
