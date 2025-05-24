# Puente 🌉

[![Go Reference](https://pkg.go.dev/badge/github.com/javiertlopez/puente.svg)](https://pkg.go.dev/github.com/javiertlopez/puente)
[![Go Report Card](https://goreportcard.com/badge/github.com/javiertlopez/puente)](https://goreportcard.com/report/github.com/javiertlopez/puente)
[![MIT License](https://img.shields.io/github/license/javiertlopez/puente)](https://github.com/javiertlopez/puente/blob/main/LICENSE)
[![Go Version](https://img.shields.io/github/go-mod/go-version/javiertlopez/puente)](https://github.com/javiertlopez/puente/blob/main/go.mod)

> *"Puente"* means "bridge" in Spanish - connecting your HTTP handlers with security and logging capabilities

Puente is a lightweight and flexible HTTP middleware package for Go that provides:

- 🔑 **JWT Authentication**: Extract and validate JWT claims from requests
- 📋 **Logging**: Detailed request logging with user context
- 🔍 **Request Tracking**: Unique request IDs for improved traceability
- 🛠️ **Extensible**: Easy to integrate with any Go HTTP service

## Installation

```bash
go get github.com/javiertlopez/puente
```

## Usage

### Basic Setup

```go
package main

import (
    "log"
    "net/http"

    "github.com/javiertlopez/puente"
    "github.com/sirupsen/logrus"
)

func main() {
    // Create a logger
    logger := logrus.New()
    logger.SetFormatter(&logrus.JSONFormatter{})

    // Create a JWT extractor (implement the JWTExtractor interface)
    extractor := NewMyJWTExtractor()

    // Create middleware instance
    middleware := puente.New("my-service", logger, extractor)

    // Create your router/mux
    mux := http.NewServeMux()
    mux.HandleFunc("/", helloHandler)

    // Apply middleware
    handler := middleware.Logging(middleware.JWT(mux))

    // Start server
    log.Fatal(http.ListenAndServe(":8080", handler))
}

func helloHandler(w http.ResponseWriter, r *http.Request) {
    // Get user ID from context (added by JWT middleware)
    userID, ok := puente.GetUserID(r.Context())
    
    // Get request ID from context
    requestID, hasRequestID := puente.GetRequestID(r.Context())
    if hasRequestID {
        w.Header().Set("X-Request-ID", requestID)
    }
    
    if ok {
        w.Write([]byte("Hello, " + userID))
        return
    }
    w.Write([]byte("Hello, world!"))
}
```

### Implementing JWT Extraction

To use the JWT middleware, implement the `JWTExtractor` interface:

```go
type MyJWTExtractor struct {
    // Your implementation details
}

func (e *MyJWTExtractor) ExtractJWT(r *http.Request) (puente.JWTClaims, error) {
    // Extract token from Authorization header
    authHeader := r.Header.Get("Authorization")
    if authHeader == "" {
        return puente.JWTClaims{}, errors.New("no authorization header")
    }

    // Parse JWT token and validate (implementation dependent on your JWT library)
    // ...

    // Return claims
    return puente.JWTClaims{
        Sub:    "user123",
        Issuer: "my-service",
        // ... other claims
    }, nil
}
```

## Middleware Features

### JWT Middleware

The JWT middleware:
- Extracts JWT claims from requests using the provided extractor
- Adds the user ID (`sub` claim) to the request context
- Generates or propagates a unique request ID for each request
- Logs extraction success or failures with request ID for traceability
- Passes through the request even if extraction fails (non-blocking)

### Logging Middleware

The logging middleware:
- Records request information including:
  - Application name
  - HTTP method and path
  - Response status code
  - Request duration
  - User ID (if available)
  - Request ID for cross-component tracing
- Generates new request IDs or uses existing ones from context
- Logs warnings when user ID is not found in context
- Uses structured logging via logrus with consistent field formatting

### Request ID Tracking

The request ID functionality:
- Automatically generates a unique UUID for each request if not present
- Propagates request IDs through all middleware components
- Makes request IDs available in the context for your handlers
- Includes request IDs in all log entries for easy request tracing
- Provides helper function (`GetRequestID`) to retrieve request IDs from context

## Examples

### Chaining Multiple Middleware

```go
// Create a chain of middleware
handler := middleware.Logging(   // First executed (outer)
             middleware.JWT(     // Second executed
               yourHandler       // Finally, your handler
             )
           )
```

### Using Request IDs for Distributed Tracing

```go
func apiHandler(w http.ResponseWriter, r *http.Request) {
    // Get the request ID from context
    requestID, ok := puente.GetRequestID(r.Context())
    if !ok {
        // Should not happen if middleware is properly set up
        requestID = uuid.New().String()
    }
    
    // Include the request ID in response headers
    w.Header().Set("X-Request-ID", requestID)
    
    // Use the request ID in your application logic
    resp, err := callDownstreamService(r.Context(), requestID)
    if err != nil {
        // The error will be logged with request ID automatically
        http.Error(w, "Internal Server Error", http.StatusInternalServerError)
        return
    }
    
    w.Write(resp)
}

// Example downstream service call with request ID propagation
func callDownstreamService(ctx context.Context, requestID string) ([]byte, error) {
    req, err := http.NewRequest("GET", "https://api.example.com/data", nil)
    if err != nil {
        return nil, err
    }
    
    // Propagate the request ID to downstream services
    req.Header.Set("X-Request-ID", requestID)
    
    // Use context for request
    req = req.WithContext(ctx)
    
    // Make the request...
    // ...
    
    return responseData, nil
}
```

### Custom Logger Configuration

```go
logger := logrus.New()
logger.SetLevel(logrus.InfoLevel)
logger.SetFormatter(&logrus.JSONFormatter{
    TimestampFormat: time.RFC3339,
})

middleware := puente.New("api-service", logger, extractor)
```

## Testing

Puente includes comprehensive test suites for all components. To run the tests:

```bash
go test -v ./...
```

### Testing with Your Application

When testing your application that uses Puente middleware, you can use the `logrus/hooks/test` package to verify logging behavior:

```go
import (
    "context"
    "net/http"
    "net/http/httptest"
    "testing"
    
    "github.com/sirupsen/logrus"
    "github.com/sirupsen/logrus/hooks/test"
    "github.com/javiertlopez/puente"
)

func TestMyHandler(t *testing.T) {
    // Setup test logger with hook
    logger, hook := test.NewNullLogger()
    
    // Create middleware with test logger
    middleware := puente.New("test-app", logger, myExtractor)
    
    // Create a test handler
    testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Get request ID from context
        requestID, ok := puente.GetRequestID(r.Context())
        if !ok {
            t.Error("Expected request ID in context, but none found")
        }
        
        // Get user ID from context
        userID, ok := puente.GetUserID(r.Context())
        if !ok {
            t.Error("Expected user ID in context, but none found")
        }
        
        // Write user ID in response
        w.Write([]byte(userID))
    })
    
    // Apply middleware to your handler
    handler := middleware.Logging(middleware.JWT(testHandler))
    
    // Create test request
    req := httptest.NewRequest("GET", "/api/resource", nil)
    // Optionally provide a request ID to simulate incoming request with ID
    ctx := context.WithValue(req.Context(), puente.RequestIDKey, "test-request-id")
    req = req.WithContext(ctx)
    
    // Record the response
    recorder := httptest.NewRecorder()
    
    // Execute the handler
    handler.ServeHTTP(recorder, req)
    
    // Verify status code
    if recorder.Code != http.StatusOK {
        t.Errorf("Expected status code %d, got %d", http.StatusOK, recorder.Code)
    }
    
    // Verify logging behavior
    for _, entry := range hook.Entries {
        if entry.Level == logrus.InfoLevel {
            // Check log fields
            userID, exists := entry.Data["user_id"]
            if !exists {
                t.Error("Expected user_id field in log entry but it was not present")
            }
            
            // Check request ID in logs
            requestID, exists := entry.Data["request_id"]
            if !exists {
                t.Error("Expected request_id field in log entry but it was not present")
            }
            
            if requestID != "test-request-id" {
                t.Errorf("Expected request_id to be %s, got %v", "test-request-id", requestID)
            }
        }
    }
}
```

## Best Practices

### Request ID Management

1. **Propagate Request IDs**: When making downstream service calls, include the request ID in headers (commonly as `X-Request-ID`)
2. **Return Request IDs**: Include the request ID in API responses to help clients correlate their requests
3. **Accept Incoming Request IDs**: If a client provides a request ID, use it instead of generating a new one
4. **Log with Request IDs**: Always include the request ID in log entries for distributed tracing

### Standardized Logging

Puente ensures consistent logging across all middleware components by:

1. **Common Base Fields**: All log entries include:
   - `app`: The application name provided during middleware creation
   - `timestamp`: UTC timestamp in RFC3339 format
   - `request_id`: Unique identifier for request tracing
   
2. **Request-Specific Fields**: The logging middleware adds:
   - `method`: HTTP method (GET, POST, etc.)
   - `path`: Request path
   - `status`: HTTP status code
   - `duration`: Request processing duration
   - `user_id`: User identifier (when available)

This standardization makes logs easier to search, filter, and analyze across your distributed system.

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

This project is licensed under the MIT License - see the LICENSE file for details.
