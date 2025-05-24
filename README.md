# Puente 🌉

[![Go Reference](https://pkg.go.dev/badge/github.com/javiertlopez/puente.svg)](https://pkg.go.dev/github.com/javiertlopez/puente)
[![Go Report Card](https://goreportcard.com/badge/github.com/javiertlopez/puente)](https://goreportcard.com/report/github.com/javiertlopez/puente)
[![MIT License](https://img.shields.io/github/license/javiertlopez/puente)](https://github.com/javiertlopez/puente/blob/main/LICENSE)
[![Go Version](https://img.shields.io/github/go-mod/go-version/javiertlopez/puente)](https://github.com/javiertlopez/puente/blob/main/go.mod)

> *"Puente"* means "bridge" in Spanish - connecting your HTTP handlers with security and logging capabilities

Puente is a lightweight and flexible HTTP middleware package for Go that provides:

- 🔑 **JWT Authentication**: Extract and validate JWT claims from requests
- 📋 **Logging**: Detailed request logging with user context
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
- Logs extraction success or failures
- Passes through the request even if extraction fails (non-blocking)

### Logging Middleware

The logging middleware:
- Records request information including:
  - Application name
  - HTTP method and path
  - Response status code
  - Request duration
  - User ID (if available)
- Logs warnings when user ID is not found in context
- Uses structured logging via logrus

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
    
    // Test your handler with middleware
    // ...
    
    // Verify logging behavior
    for _, entry := range hook.Entries {
        if entry.Level == logrus.InfoLevel {
            // Check log fields
            userID := entry.Data["user_id"]
            // ...assertions...
        }
    }
}
```

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

This project is licensed under the MIT License - see the LICENSE file for details.
