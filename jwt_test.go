package puente

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

const requestContextKey contextKey = "requestContext"

type mockHandler struct {
	called   bool
	captured *http.Request
	response string
}

func (h *mockHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.called = true
	h.captured = r
	w.Write([]byte(h.response))
}

func TestJWTAPIGatewayMiddleware(t *testing.T) {
	tests := []struct {
		name           string
		contextSetup   func(context.Context) context.Context
		expectedUserID string
		expectClaims   bool
	}{
		{
			name: "Invalid claims type in context",
			contextSetup: func(ctx context.Context) context.Context {
				// Put wrong type in context
				return context.WithValue(ctx, requestContextKey, "invalid-claims")
			},
			expectedUserID: "",
			expectClaims:   false,
		},
		{
			name: "Nested empty JWT claims",
			contextSetup: func(ctx context.Context) context.Context {
				claims := APIGatewayV2AuthorizerContext{}
				claims.Authorizer.JWT.Claims = JWTClaims{} // Explicitly empty claims
				return context.WithValue(ctx, requestContextKey, claims)
			},
			expectedUserID: "",
			expectClaims:   true,
		},
		{
			name: "No claims in context",
			contextSetup: func(ctx context.Context) context.Context {
				return ctx
			},
			expectedUserID: "",
			expectClaims:   false,
		},
		{
			name: "Empty claims in context",
			contextSetup: func(ctx context.Context) context.Context {
				claims := APIGatewayV2AuthorizerContext{}
				return context.WithValue(ctx, requestContextKey, claims)
			},
			expectedUserID: "",
			expectClaims:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			mock := &mockHandler{response: "ok"}
			middleware := JWTAPIGatewayMiddleware(mock)

			// Create request with test context
			req := httptest.NewRequest("GET", "/test", nil)
			req = req.WithContext(tt.contextSetup(req.Context()))

			// Create response recorder
			rec := httptest.NewRecorder()

			// Execute
			middleware.ServeHTTP(rec, req)

			// Verify handler was called
			if !mock.called {
				t.Error("Handler was not called")
			}

			// Verify context values
			if tt.expectClaims {
				userID, ok := mock.captured.Context().Value(userIDKey).(string)
				if !ok && tt.expectedUserID != "" {
					t.Error("Expected user ID in context but found none")
				}
				if userID != tt.expectedUserID {
					t.Errorf("Expected user ID %s but got %s", tt.expectedUserID, userID)
				}
			}

			// Verify response
			if rec.Code != http.StatusOK {
				t.Errorf("Expected status code %d but got %d", http.StatusOK, rec.Code)
			}
		})
	}
}
