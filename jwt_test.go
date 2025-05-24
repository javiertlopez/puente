package puente

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sirupsen/logrus"
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
		{
			name:        "value exists but is not a string",
			ctx:         context.WithValue(context.Background(), UserIDKey, 123),
			wantUserID:  "",
			wantPresent: false,
		},
		{
			name:        "empty string user id",
			ctx:         context.WithValue(context.Background(), UserIDKey, ""),
			wantUserID:  "",
			wantPresent: true,
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
		name              string
		claims            JWTClaims
		extractErr        error
		setupContext      func() context.Context
		expectWarnLog     bool
		expectInfoLog     bool
		expectUserID      string
		expectUserInLog   bool
		expectRequestID   bool
		checkRequestIDLog bool
	}{
		{
			name: "successful extraction",
			claims: JWTClaims{
				Sub: "test-user",
			},
			setupContext: func() context.Context {
				return context.Background() // No request ID in context
			},
			extractErr:        nil,
			expectWarnLog:     false,
			expectInfoLog:     true,
			expectUserID:      "test-user",
			expectUserInLog:   true,
			expectRequestID:   true,
			checkRequestIDLog: true,
		},
		{
			name: "existing request ID in context",
			claims: JWTClaims{
				Sub: "context-user",
			},
			setupContext: func() context.Context {
				return context.WithValue(context.Background(), RequestIDKey, "existing-request-id")
			},
			extractErr:        nil,
			expectWarnLog:     false,
			expectInfoLog:     true,
			expectUserID:      "context-user",
			expectUserInLog:   true,
			expectRequestID:   true,
			checkRequestIDLog: true,
		},
		{
			name:       "extraction failure",
			claims:     JWTClaims{},
			extractErr: errors.New("extraction failed"),
			setupContext: func() context.Context {
				return context.Background()
			},
			expectWarnLog:     true,
			expectInfoLog:     false,
			expectUserID:      "",
			expectUserInLog:   false,
			expectRequestID:   false,
			checkRequestIDLog: true,
		},
		{
			name: "empty subject in claims",
			claims: JWTClaims{
				Sub: "",
			},
			setupContext: func() context.Context {
				return context.Background()
			},
			extractErr:        nil,
			expectWarnLog:     false,
			expectInfoLog:     true,
			expectUserID:      "",
			expectUserInLog:   true,
			expectRequestID:   true,
			checkRequestIDLog: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup logger with test hook
			logger, hook := test.NewNullLogger()
			extractor := &mockJWTExtractor{
				claims: tt.claims,
				err:    tt.extractErr,
			}

			m := &Middleware{
				app:       "test-app",
				extractor: extractor,
				logger:    logger,
			}

			handler := m.JWT(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if tt.extractErr == nil {
					userID, ok := GetUserID(r.Context())
					if !ok || userID != tt.claims.Sub {
						t.Errorf("Expected user ID %s in context, got %s", tt.claims.Sub, userID)
					}

					if tt.expectRequestID {
						requestID, ok := GetRequestID(r.Context())
						if !ok {
							t.Error("Expected request ID in context, but it was not present")
						} else if tt.setupContext().Value(RequestIDKey) != nil && requestID != tt.setupContext().Value(RequestIDKey).(string) {
							t.Errorf("Expected request ID %s in context, got %s", tt.setupContext().Value(RequestIDKey).(string), requestID)
						}
					}
				}
			}))

			req := httptest.NewRequest("GET", "/", nil)
			// Apply the context from the test case
			req = req.WithContext(tt.setupContext())
			rr := httptest.NewRecorder()

			handler.ServeHTTP(rr, req)

			// Verify logging behavior using the hook
			if tt.expectWarnLog {
				foundWarning := false
				for _, entry := range hook.Entries {
					if entry.Level == logrus.WarnLevel && entry.Message == "Failed to extract JWT claims" {
						foundWarning = true
						break
					}
				}
				if !foundWarning {
					t.Error("Expected warning log but none was recorded")
				}
			}

			if tt.expectInfoLog {
				foundInfo := false
				for _, entry := range hook.Entries {
					if entry.Level == logrus.InfoLevel && entry.Message == "User ID found in JWT" {
						foundInfo = true
						if tt.expectUserInLog {
							if userID, exists := entry.Data["user_id"]; exists {
								if userID != tt.claims.Sub {
									t.Errorf("Expected user_id=%s in log field, got %v", tt.claims.Sub, userID)
								}
							} else {
								t.Error("Expected user_id field in log entry but it was not present")
							}
						}

						// Verify request ID in log fields
						if tt.checkRequestIDLog {
							if _, exists := entry.Data["request_id"]; !exists {
								t.Error("Expected request_id field in log entry but it was not present")
							}
						}
						break
					}
				}
				if !foundInfo {
					t.Error("Expected info log but none was recorded")
				}
			}

			// Reset the hook for the next test
			hook.Reset()
		})
	}
}
