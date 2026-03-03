package security

import (
	"encoding/json"
	"net"
	"net/http"
	"strings"
)

// AuthConfig holds configuration for API key authentication middleware
type AuthConfig struct {
	APIKeys   []string // valid API keys
	AllowFrom []string // allowed IP CIDRs (empty = allow all)
	Enabled   bool     // if false, skip auth
}

// AuthMiddleware returns HTTP middleware that validates API keys
func AuthMiddleware(cfg AuthConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// If disabled, pass through
			if !cfg.Enabled {
				next.ServeHTTP(w, r)
				return
			}

			// Extract API key from X-API-Key header or Authorization Bearer token
			apiKey := r.Header.Get("X-API-Key")
			if apiKey == "" {
				authHeader := r.Header.Get("Authorization")
				if strings.HasPrefix(authHeader, "Bearer ") {
					apiKey = strings.TrimPrefix(authHeader, "Bearer ")
				}
			}

			// Check if API key is valid
			if apiKey == "" || !isValidAPIKey(apiKey, cfg.APIKeys) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
				return
			}

			// If AllowFrom is set, validate source IP
			if len(cfg.AllowFrom) > 0 {
				clientIP := getClientIP(r)
				if !isIPAllowed(clientIP, cfg.AllowFrom) {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusForbidden)
					json.NewEncoder(w).Encode(map[string]string{"error": "forbidden"})
					return
				}
			}

			next.ServeHTTP(w, r)
		})
	}
}

// isValidAPIKey checks if the provided key is in the list of valid keys
func isValidAPIKey(key string, validKeys []string) bool {
	for _, validKey := range validKeys {
		if key == validKey {
			return true
		}
	}
	return false
}

// getClientIP extracts the client IP from the request
func getClientIP(r *http.Request) string {
	// Try X-Forwarded-For first
	xff := r.Header.Get("X-Forwarded-For")
	if xff != "" {
		ips := strings.Split(xff, ",")
		if len(ips) > 0 {
			return strings.TrimSpace(ips[0])
		}
	}

	// Try X-Real-IP
	xri := r.Header.Get("X-Real-IP")
	if xri != "" {
		return xri
	}

	// Fall back to RemoteAddr
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return ip
}

// isIPAllowed checks if the client IP is in the allowed CIDR ranges
func isIPAllowed(clientIP string, allowedCIDRs []string) bool {
	ip := net.ParseIP(clientIP)
	if ip == nil {
		return false
	}

	for _, cidr := range allowedCIDRs {
		_, network, err := net.ParseCIDR(cidr)
		if err != nil {
			continue
		}
		if network.Contains(ip) {
			return true
		}
	}

	return false
}
