package puente

import (
	"context"
	"net/http"
)

// contextKey is a custom type for context keys
type contextKey string

const userIDKey contextKey = "user_id"

type JWTClaims struct {
	Sub      string `json:"sub"`
	Issuer   string `json:"iss"`
	Audience string `json:"aud"`
	Username string `json:"username,omitempty"`  // Cognito specific
	TokenUse string `json:"token_use,omitempty"` // Cognito specific
}

type APIGatewayV2AuthorizerContext struct {
	Authorizer struct {
		JWT struct {
			Claims JWTClaims `json:"claims"`
		} `json:"jwt"`
	} `json:"authorizer"`
}

func JWTAPIGatewayMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if claims, ok := r.Context().Value("requestContext").(APIGatewayV2AuthorizerContext); ok {
			// Add claims to a new context
			ctx := context.WithValue(r.Context(), userIDKey, claims.Authorizer.JWT.Claims.Sub)
			next.ServeHTTP(w, r.WithContext(ctx))
			return
		}
		next.ServeHTTP(w, r)
	})
}
