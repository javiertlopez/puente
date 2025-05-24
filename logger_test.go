package puente

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
)

func TestResponseWriter(t *testing.T) {
	t.Run("default status code", func(t *testing.T) {
		rr := httptest.NewRecorder()
		wrapped := newResponseWriter(rr)

		if wrapped.statusCode != http.StatusOK {
			t.Errorf("Expected default status code %d, got %d", http.StatusOK, wrapped.statusCode)
		}
	})

	t.Run("custom status code", func(t *testing.T) {
		rr := httptest.NewRecorder()
		wrapped := newResponseWriter(rr)

		wrapped.WriteHeader(http.StatusCreated)

		if wrapped.statusCode != http.StatusCreated {
			t.Errorf("Expected status code %d, got %d", http.StatusCreated, wrapped.statusCode)
		}

		if rr.Code != http.StatusCreated {
			t.Errorf("Underlying ResponseWriter not updated, got %d", rr.Code)
		}
	})

	t.Run("error status code", func(t *testing.T) {
		rr := httptest.NewRecorder()
		wrapped := newResponseWriter(rr)

		wrapped.WriteHeader(http.StatusInternalServerError)

		if wrapped.statusCode != http.StatusInternalServerError {
			t.Errorf("Expected status code %d, got %d", http.StatusInternalServerError, wrapped.statusCode)
		}

		if rr.Code != http.StatusInternalServerError {
			t.Errorf("Underlying ResponseWriter not updated, got %d", rr.Code)
		}
	})

	t.Run("write operations", func(t *testing.T) {
		rr := httptest.NewRecorder()
		wrapped := newResponseWriter(rr)

		testData := []byte("test response data")
		_, err := wrapped.Write(testData)

		if err != nil {
			t.Errorf("Write operation failed: %v", err)
		}

		if rr.Body.String() != string(testData) {
			t.Errorf("Expected body %q, got %q", string(testData), rr.Body.String())
		}
	})
}

func TestLoggingMiddleware(t *testing.T) {
	tests := []struct {
		name              string
		setupContext      func() context.Context
		setupHandler      func(http.ResponseWriter, *http.Request)
		method            string
		path              string
		expectUserID      string
		expectWarnLog     bool
		expectedStatus    int
		expectRequestID   bool
		provideRequestID  bool
		expectedRequestID string
	}{
		{
			name: "successful request with user ID",
			setupContext: func() context.Context {
				return context.WithValue(context.Background(), UserIDKey, "test-user")
			},
			setupHandler: func(w http.ResponseWriter, r *http.Request) {
				// Do nothing, return 200 OK
			},
			method:            "GET",
			path:              "/test",
			expectUserID:      "test-user",
			expectWarnLog:     false,
			expectedStatus:    http.StatusOK,
			expectRequestID:   true,
			provideRequestID:  false,
			expectedRequestID: "", // Generated automatically
		},
		{
			name: "request without user ID",
			setupContext: func() context.Context {
				return context.Background()
			},
			setupHandler: func(w http.ResponseWriter, r *http.Request) {
				// Do nothing, return 200 OK
			},
			method:            "POST",
			path:              "/api/data",
			expectUserID:      "",
			expectWarnLog:     true,
			expectedStatus:    http.StatusOK,
			expectRequestID:   true,
			provideRequestID:  false,
			expectedRequestID: "", // Generated automatically
		},
		{
			name: "request with error status code",
			setupContext: func() context.Context {
				return context.WithValue(context.Background(), UserIDKey, "error-user")
			},
			setupHandler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusBadRequest)
			},
			method:            "PUT",
			path:              "/api/update",
			expectUserID:      "error-user",
			expectWarnLog:     false,
			expectedStatus:    http.StatusBadRequest,
			expectRequestID:   true,
			provideRequestID:  false,
			expectedRequestID: "", // Generated automatically
		},
		{
			name: "request with existing request ID",
			setupContext: func() context.Context {
				ctx := context.WithValue(context.Background(), UserIDKey, "request-id-user")
				return context.WithValue(ctx, RequestIDKey, "existing-request-id")
			},
			setupHandler: func(w http.ResponseWriter, r *http.Request) {
				// Do nothing, return 200 OK
			},
			method:            "GET",
			path:              "/api/with-request-id",
			expectUserID:      "request-id-user",
			expectWarnLog:     false,
			expectedStatus:    http.StatusOK,
			expectRequestID:   true,
			provideRequestID:  true,
			expectedRequestID: "existing-request-id",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup logger with test hook
			logger, hook := test.NewNullLogger()

			// Create middleware
			m := &Middleware{
				app:    "test-app",
				logger: logger,
			}

			// Create test handler with additional verification
			testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Verify request ID propagation
				if tt.expectRequestID {
					requestID, ok := GetRequestID(r.Context())
					if !ok {
						t.Error("Expected request ID in context, but it was not present")
					} else if tt.provideRequestID && requestID != tt.expectedRequestID {
						t.Errorf("Expected request ID %s in context, got %s", tt.expectedRequestID, requestID)
					}
				}

				// Execute the original test handler
				tt.setupHandler(w, r)
			})

			// Apply middleware
			handler := m.Logging(testHandler)

			// Create request with context
			req := httptest.NewRequest(tt.method, tt.path, nil)
			req = req.WithContext(tt.setupContext())

			// Record the response
			rr := httptest.NewRecorder()

			// Execute the handler
			handler.ServeHTTP(rr, req)

			// Check status code
			if rr.Code != tt.expectedStatus {
				t.Errorf("Expected status code %d, got %d", tt.expectedStatus, rr.Code)
			}

			// Check for warning log if expected
			if tt.expectWarnLog {
				foundWarn := false
				for _, entry := range hook.Entries {
					if entry.Level == logrus.WarnLevel && entry.Message == "Failed to get user ID from context" {
						foundWarn = true
						break
					}
				}
				if !foundWarn {
					t.Error("Expected warning log but none was found")
				}
			}

			// Check for info log with expected fields
			foundInfo := false
			for _, entry := range hook.Entries {
				if entry.Level == logrus.InfoLevel {
					foundInfo = true

					// Check app field
					if app, exists := entry.Data["app"]; !exists || app != "test-app" {
						t.Errorf("Expected app field to be 'test-app', got %v", app)
					}

					// Check method field
					if method, exists := entry.Data["method"]; !exists || method != tt.method {
						t.Errorf("Expected method field to be '%s', got %v", tt.method, method)
					}

					// Check path field
					if path, exists := entry.Data["path"]; !exists || path != tt.path {
						t.Errorf("Expected path field to be '%s', got %v", tt.path, path)
					}

					// Check status field
					if status, exists := entry.Data["status"]; !exists || status != tt.expectedStatus {
						t.Errorf("Expected status field to be %d, got %v", tt.expectedStatus, status)
					}

					// Check user_id field
					if userID, exists := entry.Data["user_id"]; !exists || userID != tt.expectUserID {
						t.Errorf("Expected user_id field to be '%s', got %v", tt.expectUserID, userID)
					}

					// Check that duration field exists
					if _, exists := entry.Data["duration"]; !exists {
						t.Error("Expected duration field but it was missing")
					}

					// Check request_id field
					if tt.expectRequestID {
						requestID, exists := entry.Data["request_id"]
						if !exists {
							t.Error("Expected request_id field in log entry but it was not present")
						} else if tt.provideRequestID && requestID != tt.expectedRequestID {
							t.Errorf("Expected request_id field to be '%s', got %v", tt.expectedRequestID, requestID)
						}
					}
				}
			}

			if !foundInfo {
				t.Error("Expected info log but none was found")
			}

			hook.Reset()
		})
	}
}

func TestGetRequestID(t *testing.T) {
	tests := []struct {
		name          string
		ctx           context.Context
		wantRequestID string
		wantPresent   bool
	}{
		{
			name:          "request id exists",
			ctx:           context.WithValue(context.Background(), RequestIDKey, "test-request-id"),
			wantRequestID: "test-request-id",
			wantPresent:   true,
		},
		{
			name:          "request id does not exist",
			ctx:           context.Background(),
			wantRequestID: "",
			wantPresent:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotRequestID, gotPresent := GetRequestID(tt.ctx)
			if gotRequestID != tt.wantRequestID {
				t.Errorf("GetRequestID() gotRequestID = %v, want %v", gotRequestID, tt.wantRequestID)
			}
			if gotPresent != tt.wantPresent {
				t.Errorf("GetRequestID() gotPresent = %v, want %v", gotPresent, tt.wantPresent)
			}
		})
	}
}
