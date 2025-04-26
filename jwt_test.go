package puente

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sirupsen/logrus/hooks/test"
)

type mockJWTExtractor struct {
	claims JWTClaims
	err    error
}

func (m *mockJWTExtractor) ExtractJWT(r *http.Request) (JWTClaims, error) {
	return m.claims, m.err
}

func TestGetUserID(t *testing.T) {
	tests := []struct {
		name        string
		ctx         context.Context
		wantUserID  string
		wantPresent bool
	}{
		{
			name:        "user id exists",
			ctx:         context.WithValue(context.Background(), UserIDKey, "test-user"),
			wantUserID:  "test-user",
			wantPresent: true,
		},
		{
			name:        "user id does not exist",
			ctx:         context.Background(),
			wantUserID:  "",
			wantPresent: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotUserID, gotPresent := GetUserID(tt.ctx)
			if gotUserID != tt.wantUserID {
				t.Errorf("GetUserID() gotUserID = %v, want %v", gotUserID, tt.wantUserID)
			}
			if gotPresent != tt.wantPresent {
				t.Errorf("GetUserID() gotPresent = %v, want %v", gotPresent, tt.wantPresent)
			}
		})
	}
}

func TestJWTMiddleware(t *testing.T) {
	tests := []struct {
		name       string
		claims     JWTClaims
		extractErr error
	}{
		{
			name: "successful extraction",
			claims: JWTClaims{
				Sub: "test-user",
			},
			extractErr: nil,
		},
		{
			name:       "extraction failure",
			claims:     JWTClaims{},
			extractErr: errors.New("extraction failed"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger, _ := test.NewNullLogger()
			extractor := &mockJWTExtractor{
				claims: tt.claims,
				err:    tt.extractErr,
			}

			m := &Middleware{
				extractor: extractor,
				logger:    logger,
			}

			handler := m.JWTMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if tt.extractErr == nil {
					userID, ok := GetUserID(r.Context())
					if !ok || userID != tt.claims.Sub {
						t.Errorf("Expected user ID %s in context, got %s", tt.claims.Sub, userID)
					}
				}
			}))

			req := httptest.NewRequest("GET", "/", nil)
			rr := httptest.NewRecorder()

			handler.ServeHTTP(rr, req)
		})
	}
}
