package puente

import (
	"context"
	"net/http"
)

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

// JWT is a middleware that extracts JWT claims from the request and adds the user ID to the context
func (m *Middleware) JWT(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get request ID from context or generate a new one
		requestID := r.Context().Value(RequestIDKey)
		if requestID == nil {
			requestID = generateRequestID()
		}

		logFields := m.defaultLogFields()
		logFields["request_id"] = requestID

		claims, err := m.extractor.ExtractJWT(r)
		if err != nil {
			m.logger.WithFields(logFields).Warn("Failed to extract JWT claims")
			next.ServeHTTP(w, r)
			return
		}

		logFields["user_id"] = claims.Sub
		m.logger.WithFields(logFields).Info("User ID found in JWT")

		// Add both user_id and request_id to context
		ctx := r.Context()
		ctx = context.WithValue(ctx, UserIDKey, claims.Sub)
		ctx = context.WithValue(ctx, RequestIDKey, requestID)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
