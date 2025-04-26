package puente

import (
	"context"
	"net/http"
)

type contextKey string

const UserIDKey contextKey = "user_id"

// JWTExtractor is an interface for extracting JWT claims from a request
type JWTExtractor interface {
	ExtractJWT(r *http.Request) (JWTClaims, error)
}

// JWTClaims represents the claims in a JWT token
type JWTClaims struct {
	Sub      string
	Issuer   string
	AuthTime string
	Exp      string
	Iat      string
	Jti      string
	Scope    string
}

// GetUserID retrieves the user ID from the context
func GetUserID(ctx context.Context) (string, bool) {
	v := ctx.Value(UserIDKey)
	if v == nil {
		return "", false
	}

	userID, ok := v.(string)
	return userID, ok
}

// JWTMiddleware is a middleware that extracts JWT claims from the request and adds the user ID to the context
func (m *Middleware) JWTMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims, err := m.extractor.ExtractJWT(r)
		if err != nil {
			m.logger.Warn("Failed to extract JWT claims")
			next.ServeHTTP(w, r)
			return
		}

		m.logger.WithField("user_id", claims.Sub).Info("User ID found in JWT")
		reqCtx := context.WithValue(r.Context(), UserIDKey, claims.Sub)
		next.ServeHTTP(w, r.WithContext(reqCtx))
	})
}
