package apiv1_test

import (
	"context"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/getfider/fider/app"
	"github.com/getfider/fider/app/handlers/apiv1"
	"github.com/getfider/fider/app/middlewares"
	"github.com/getfider/fider/app/models/cmd"
	"github.com/getfider/fider/app/models/entity"
	"github.com/getfider/fider/app/models/enum"
	"github.com/getfider/fider/app/models/query"
	"github.com/getfider/fider/app/pkg/bus"
	"github.com/getfider/fider/app/pkg/jwt"
	"github.com/getfider/fider/app/pkg/mock"
	. "github.com/getfider/fider/app/pkg/assert"
)

// TestMain sets up the test environment
func TestMain(m *testing.M) {
	// Set GO_ENV to "test" to skip SSR initialization and prevent ssr.js file requirement
	os.Setenv("GO_ENV", "test")
	
	// Set JWT_SECRET for token generation/validation in tests
	os.Setenv("JWT_SECRET", "test-secret-key-for-jwt-token-generation")
	
	// Set ALLOWED_ORIGINS for CORS tests to allow cross-origin requests from the test origin
	os.Setenv("ALLOWED_ORIGINS", "https://app.example.com,http://localhost:3000")
	
	// Run tests
	exitCode := m.Run()
	
	// Exit with the test result code
	os.Exit(exitCode)
}

// TestLogin_Success tests successful login with valid credentials
func TestLogin_Success(t *testing.T) {
	RegisterT(t)

	server := mock.NewServer()

	// Mock verification key lookup for login
	bus.AddHandler(func(ctx context.Context, q *query.GetVerificationByKey) error {
		if q.Key == "valid-verification-key" && q.Kind == enum.EmailVerificationKindSignIn {
			q.Result = &entity.EmailVerification{
				Email:     mock.JonSnow.Email,
				Name:      mock.JonSnow.Name,
				Key:       q.Key,
				Kind:      enum.EmailVerificationKindSignIn,
				CreatedAt: time.Now(),
				ExpiresAt: time.Now().Add(15 * time.Minute),
			}
			return nil
		}
		return app.ErrNotFound
	})

	// Mock setting the key as verified
	bus.AddHandler(func(ctx context.Context, c *cmd.SetKeyAsVerified) error {
		return nil
	})

	// Mock user lookup by email
	bus.AddHandler(func(ctx context.Context, q *query.GetUserByEmail) error {
		if q.Email == mock.JonSnow.Email {
			q.Result = mock.JonSnow
			return nil
		}
		return app.ErrNotFound
	})

	status, response := server.
		OnTenant(mock.DemoTenant).
		ExecutePostAsJSON(apiv1.Login(),
			`{
				"email": "jon.snow@got.com",
				"verificationKey": "valid-verification-key"
			}`)

	Expect(status).Equals(http.StatusOK)
	Expect(response.String("accessToken")).IsNotEmpty()
	Expect(response.Int32("user.id")).Equals(mock.JonSnow.ID)
	Expect(response.String("user.name")).Equals(mock.JonSnow.Name)
	Expect(response.String("user.email")).Equals(mock.JonSnow.Email)
}

// TestLogin_InvalidEmail tests login with non-existent email
func TestLogin_InvalidEmail(t *testing.T) {
	RegisterT(t)

	server := mock.NewServer()

	// Mock verification key lookup
	bus.AddHandler(func(ctx context.Context, q *query.GetVerificationByKey) error {
		if q.Key == "valid-verification-key" {
			q.Result = &entity.EmailVerification{
				Email:      "nonexistent@got.com",
				Name:       "Test User",
				Key:        "valid-verification-key",
				UserID:     0,
				Kind:       enum.EmailVerificationKindSignIn,
				ExpiresAt:  time.Now().Add(1 * time.Hour),
				VerifiedAt: nil,
			}
			return nil
		}
		return app.ErrNotFound
	})

	// Mock setting key as verified
	bus.AddHandler(func(ctx context.Context, c *cmd.SetKeyAsVerified) error {
		return nil
	})

	// Mock user lookup by email - always return not found
	bus.AddHandler(func(ctx context.Context, q *query.GetUserByEmail) error {
		return app.ErrNotFound
	})

	status, _ := server.
		OnTenant(mock.DemoTenant).
		ExecutePostAsJSON(apiv1.Login(),
			`{
				"email": "nonexistent@got.com",
				"verificationKey": "valid-verification-key"
			}`)

	Expect(status).Equals(http.StatusUnauthorized)
}

// TestLogin_MissingCredentials tests login with missing email or password
func TestLogin_MissingCredentials(t *testing.T) {
	RegisterT(t)

	server := mock.NewServer()

	// Test missing email - action's Validate() returns field-specific error
	status, response := server.
		OnTenant(mock.DemoTenant).
		ExecutePostAsJSON(apiv1.Login(),
			`{
				"email": "",
				"verificationKey": "test-key-123"
			}`)

	Expect(status).Equals(http.StatusBadRequest)
	Expect(response.String("errors[0].field")).Equals("email")
	Expect(response.String("errors[0].message")).Equals("Email is required.")

	// Test missing verificationKey - action's Validate() returns field-specific error
	status, response = server.
		OnTenant(mock.DemoTenant).
		ExecutePostAsJSON(apiv1.Login(),
			`{
				"email": "jon.snow@got.com",
				"verificationKey": ""
			}`)

	Expect(status).Equals(http.StatusBadRequest)
	Expect(response.String("errors[0].field")).Equals("verificationKey")
	Expect(response.String("errors[0].message")).Equals("Verification Key is required.")
}

// TestLogin_InvalidJSON tests login with malformed JSON
func TestLogin_InvalidJSON(t *testing.T) {
	RegisterT(t)

	server := mock.NewServer()

	status, response := server.
		OnTenant(mock.DemoTenant).
		ExecutePostAsJSON(apiv1.Login(), `{invalid json}`)

	Expect(status).Equals(http.StatusBadRequest)
	Expect(response.String("errors[0].message")).Equals("Invalid request format")
}

// TestLogin_WrongTenant tests login when user belongs to different tenant
func TestLogin_WrongTenant(t *testing.T) {
	RegisterT(t)

	server := mock.NewServer()

	// Mock verification key lookup
	bus.AddHandler(func(ctx context.Context, q *query.GetVerificationByKey) error {
		if q.Key == "test123" {
			q.Result = &entity.EmailVerification{
				Email:      mock.JonSnow.Email,
				Name:       mock.JonSnow.Name,
				Key:        "test123",
				UserID:     mock.JonSnow.ID,
				Kind:       enum.EmailVerificationKindSignIn,
				ExpiresAt:  time.Now().Add(1 * time.Hour),
				VerifiedAt: nil,
			}
			return nil
		}
		return app.ErrNotFound
	})

	// Mock setting key as verified
	bus.AddHandler(func(ctx context.Context, c *cmd.SetKeyAsVerified) error {
		return nil
	})

	// Mock user lookup - return user with different tenant ID
	bus.AddHandler(func(ctx context.Context, q *query.GetUserByEmail) error {
		if q.Email == mock.JonSnow.Email {
			q.Result = mock.JonSnow
			return nil
		}
		return app.ErrNotFound
	})

	// Use a different tenant than JonSnow's tenant
	status, _ := server.
		OnTenant(mock.AvengersTenant).
		ExecutePostAsJSON(apiv1.Login(),
			`{
				"email": "jon.snow@got.com",
				"verificationKey": "test123"
			}`)

	Expect(status).Equals(http.StatusUnauthorized)
}

// TestRefresh_Success tests successful token refresh with valid refresh token
func TestRefresh_Success(t *testing.T) {
	RegisterT(t)

	server := mock.NewServer()

	// Generate a valid refresh token
	refreshTokenExpires := time.Now().Add(7 * 24 * time.Hour)
	refreshToken, err := jwt.Encode(jwt.FiderClaims{
		UserID:    mock.JonSnow.ID,
		UserName:  mock.JonSnow.Name,
		UserEmail: mock.JonSnow.Email,
		Origin:    jwt.FiderClaimsOriginRefresh,
		Metadata: jwt.Metadata{
			ExpiresAt: jwt.Time(refreshTokenExpires),
			IssuedAt:  jwt.Time(time.Now()),
		},
	})
	Expect(err).IsNil()

	// Mock user lookup by ID
	bus.AddHandler(func(ctx context.Context, q *query.GetUserByID) error {
		if q.UserID == mock.JonSnow.ID {
			q.Result = mock.JonSnow
			return nil
		}
		return app.ErrNotFound
	})

	// Set refresh token cookie
	refreshTokenName := apiv1.RefreshCookieName
	refreshTokenValue := refreshToken

	status, response := server.
		OnTenant(mock.DemoTenant).
		AddCookie(refreshTokenName, refreshTokenValue).
		ExecutePostAsJSON(apiv1.Refresh(), `{}`)

	Expect(status).Equals(http.StatusOK)
	Expect(response.String("accessToken")).IsNotEmpty()
}

// TestRefresh_MissingCookie tests refresh without refresh token cookie
func TestRefresh_MissingCookie(t *testing.T) {
	RegisterT(t)

	server := mock.NewServer()

	status, _ := server.
		OnTenant(mock.DemoTenant).
		ExecutePostAsJSON(apiv1.Refresh(), `{}`)

	Expect(status).Equals(http.StatusUnauthorized)
}

// TestRefresh_InvalidToken tests refresh with invalid refresh token
func TestRefresh_InvalidToken(t *testing.T) {
	RegisterT(t)

	server := mock.NewServer()

	// Set invalid refresh token cookie
	refreshTokenName := apiv1.RefreshCookieName
	refreshTokenValue := "invalid.token.value"

	status, _ := server.
		OnTenant(mock.DemoTenant).
		AddCookie(refreshTokenName, refreshTokenValue).
		ExecutePostAsJSON(apiv1.Refresh(), `{}`)

	Expect(status).Equals(http.StatusUnauthorized)
}

// TestRefresh_ExpiredToken tests refresh with expired refresh token
func TestRefresh_ExpiredToken(t *testing.T) {
	RegisterT(t)

	server := mock.NewServer()

	// Generate an expired refresh token (expired 1 hour ago)
	expiredTime := time.Now().Add(-1 * time.Hour)
	expiredToken, err := jwt.Encode(jwt.FiderClaims{
		UserID:    mock.JonSnow.ID,
		UserName:  mock.JonSnow.Name,
		UserEmail: mock.JonSnow.Email,
		Origin:    jwt.FiderClaimsOriginRefresh,
		Metadata: jwt.Metadata{
			ExpiresAt: jwt.Time(expiredTime),
			IssuedAt:  jwt.Time(time.Now().Add(-8 * time.Hour)),
		},
	})
	Expect(err).IsNil()

	// Set expired refresh token cookie
	refreshTokenName := apiv1.RefreshCookieName
	refreshTokenValue := expiredToken

	status, _ := server.
		OnTenant(mock.DemoTenant).
		AddCookie(refreshTokenName, refreshTokenValue).
		ExecutePostAsJSON(apiv1.Refresh(), `{}`)

	// Should be unauthorized because jwt.DecodeFiderClaims checks expiration
	Expect(status).Equals(http.StatusUnauthorized)
}

// TestRefresh_WrongOrigin tests refresh with token from wrong origin
func TestRefresh_WrongOrigin(t *testing.T) {
	RegisterT(t)

	server := mock.NewServer()

	// Generate a token with wrong origin
	refreshTokenExpires := time.Now().Add(7 * 24 * time.Hour)
	wrongOriginToken, err := jwt.Encode(jwt.FiderClaims{
		UserID:    mock.JonSnow.ID,
		UserName:  mock.JonSnow.Name,
		UserEmail: mock.JonSnow.Email,
		Origin:    jwt.FiderClaimsOriginUI, // Wrong origin
		Metadata: jwt.Metadata{
			ExpiresAt: jwt.Time(refreshTokenExpires),
			IssuedAt:  jwt.Time(time.Now()),
		},
	})
	Expect(err).IsNil()

	// Set refresh token cookie with wrong origin
	refreshTokenName := apiv1.RefreshCookieName
	refreshTokenValue := wrongOriginToken

	status, _ := server.
		OnTenant(mock.DemoTenant).
		AddCookie(refreshTokenName, refreshTokenValue).
		ExecutePostAsJSON(apiv1.Refresh(), `{}`)

	Expect(status).Equals(http.StatusUnauthorized)
}

// TestRefresh_UserNotFound tests refresh when user no longer exists
func TestRefresh_UserNotFound(t *testing.T) {
	RegisterT(t)

	server := mock.NewServer()

	// Generate a valid refresh token
	refreshTokenExpires := time.Now().Add(7 * 24 * time.Hour)
	refreshToken, err := jwt.Encode(jwt.FiderClaims{
		UserID:    mock.JonSnow.ID,
		UserName:  mock.JonSnow.Name,
		UserEmail: mock.JonSnow.Email,
		Origin:    jwt.FiderClaimsOriginRefresh,
		Metadata: jwt.Metadata{
			ExpiresAt: jwt.Time(refreshTokenExpires),
			IssuedAt:  jwt.Time(time.Now()),
		},
	})
	Expect(err).IsNil()

	// Mock user lookup - always return not found
	bus.AddHandler(func(ctx context.Context, q *query.GetUserByID) error {
		return app.ErrNotFound
	})

	// Set refresh token cookie
	refreshTokenName := apiv1.RefreshCookieName
	refreshTokenValue := refreshToken

	status, _ := server.
		OnTenant(mock.DemoTenant).
		AddCookie(refreshTokenName, refreshTokenValue).
		ExecutePostAsJSON(apiv1.Refresh(), `{}`)

	Expect(status).Equals(http.StatusUnauthorized)
}

// TestRefresh_WrongTenant tests refresh when user belongs to different tenant
func TestRefresh_WrongTenant(t *testing.T) {
	RegisterT(t)

	server := mock.NewServer()

	// Generate a valid refresh token
	refreshTokenExpires := time.Now().Add(7 * 24 * time.Hour)
	refreshToken, err := jwt.Encode(jwt.FiderClaims{
		UserID:    mock.JonSnow.ID,
		UserName:  mock.JonSnow.Name,
		UserEmail: mock.JonSnow.Email,
		Origin:    jwt.FiderClaimsOriginRefresh,
		Metadata: jwt.Metadata{
			ExpiresAt: jwt.Time(refreshTokenExpires),
			IssuedAt:  jwt.Time(time.Now()),
		},
	})
	Expect(err).IsNil()

	// Mock user lookup
	bus.AddHandler(func(ctx context.Context, q *query.GetUserByID) error {
		if q.UserID == mock.JonSnow.ID {
			q.Result = mock.JonSnow
			return nil
		}
		return app.ErrNotFound
	})

	// Set refresh token cookie
	refreshTokenName := apiv1.RefreshCookieName
	refreshTokenValue := refreshToken

	// Use a different tenant than JonSnow's tenant
	status, _ := server.
		OnTenant(mock.AvengersTenant).
		AddCookie(refreshTokenName, refreshTokenValue).
		ExecutePostAsJSON(apiv1.Refresh(), `{}`)

	Expect(status).Equals(http.StatusUnauthorized)
}

// TestLogout_Success tests successful logout
func TestLogout_Success(t *testing.T) {
	RegisterT(t)

	server := mock.NewServer()

	status, response := server.
		OnTenant(mock.DemoTenant).
		ExecutePostAsJSON(apiv1.Logout(), `{}`)

	Expect(status).Equals(http.StatusOK)
	Expect(response.String("status")).Equals("ok")
}

// TestLogin_JWTTokenFormat tests that login returns properly formatted JWT token
func TestLogin_JWTTokenFormat(t *testing.T) {
	RegisterT(t)

	server := mock.NewServer()

	// Mock verification key lookup
	bus.AddHandler(func(ctx context.Context, q *query.GetVerificationByKey) error {
		if q.Key == "valid-key" {
			q.Result = &entity.EmailVerification{
				Email:     mock.JonSnow.Email,
				Name:      mock.JonSnow.Name,
				Key:       "valid-key",
				Kind:      enum.EmailVerificationKindSignIn,
				CreatedAt: time.Now(),
				ExpiresAt: time.Now().Add(15 * time.Minute),
			}
			return nil
		}
		return app.ErrNotFound
	})

	// Mock setting key as verified
	bus.AddHandler(func(ctx context.Context, c *cmd.SetKeyAsVerified) error {
		return nil
	})

	// Mock user lookup
	bus.AddHandler(func(ctx context.Context, q *query.GetUserByEmail) error {
		if q.Email == mock.JonSnow.Email {
			q.Result = mock.JonSnow
			return nil
		}
		return app.ErrNotFound
	})

	status, response := server.
		OnTenant(mock.DemoTenant).
		ExecutePostAsJSON(apiv1.Login(),
			`{
				"email": "jon.snow@got.com",
				"verificationKey": "valid-key"
			}`)

	Expect(status).Equals(http.StatusOK)

	// Verify access token is returned
	accessToken := response.String("accessToken")
	Expect(accessToken).IsNotEmpty()

	// Decode and verify JWT claims
	claims, err := jwt.DecodeFiderClaims(accessToken)
	Expect(err).IsNil()
	Expect(claims.UserID).Equals(mock.JonSnow.ID)
	Expect(claims.UserName).Equals(mock.JonSnow.Name)
	Expect(claims.UserEmail).Equals(mock.JonSnow.Email)
	Expect(claims.Origin).Equals(jwt.FiderClaimsOriginUI)

	// Verify token expiration is set correctly (should be ~15 minutes from now)
	expiresAt := claims.Metadata.ExpiresAt.Time
	expectedExpiry := time.Now().Add(15 * time.Minute)
	timeDiff := expiresAt.Sub(expectedExpiry)
	Expect(timeDiff < time.Minute).IsTrue() // Allow 1 minute tolerance
}

// TestLogin_RefreshCookieAttributes tests that login sets refresh cookie with correct attributes
func TestLogin_RefreshCookieAttributes(t *testing.T) {
	RegisterT(t)

	server := mock.NewServer()

	// Mock verification key lookup
	bus.AddHandler(func(ctx context.Context, q *query.GetVerificationByKey) error {
		if q.Key == "valid-key" {
			q.Result = &entity.EmailVerification{
				Email:     mock.JonSnow.Email,
				Name:      mock.JonSnow.Name,
				Key:       "valid-key",
				Kind:      enum.EmailVerificationKindSignIn,
				CreatedAt: time.Now(),
				ExpiresAt: time.Now().Add(15 * time.Minute),
			}
			return nil
		}
		return app.ErrNotFound
	})

	// Mock setting key as verified
	bus.AddHandler(func(ctx context.Context, c *cmd.SetKeyAsVerified) error {
		return nil
	})

	// Mock user lookup
	bus.AddHandler(func(ctx context.Context, q *query.GetUserByEmail) error {
		if q.Email == mock.JonSnow.Email {
			q.Result = mock.JonSnow
			return nil
		}
		return app.ErrNotFound
	})

	status, recorder := server.
		OnTenant(mock.DemoTenant).
		ExecutePost(apiv1.Login(),
			`{
				"email": "jon.snow@got.com",
				"verificationKey": "valid-key"
			}`)

	Expect(status).Equals(http.StatusOK)

	// Get the refresh cookie from response
	var refreshCookie *http.Cookie
	for _, cookie := range recorder.Result().Cookies() {
		if cookie.Name == apiv1.RefreshCookieName {
			refreshCookie = cookie
			break
		}
	}

	// Verify refresh cookie exists
	Expect(refreshCookie).IsNotNil()

	// Verify cookie attributes for security
	Expect(refreshCookie.HttpOnly).IsTrue()  // Prevents JavaScript access
	Expect(refreshCookie.Secure).IsTrue()    // Only sent over HTTPS
	Expect(refreshCookie.SameSite).Equals(http.SameSiteNoneMode) // Cross-origin support
	Expect(refreshCookie.Path).Equals("/")   // Available to all paths

	// Verify cookie expiration is set correctly (~7 days from now)
	expectedExpiry := time.Now().Add(7 * 24 * time.Hour)
	timeDiff := refreshCookie.Expires.Sub(expectedExpiry)
	Expect(timeDiff < time.Hour).IsTrue() // Allow 1 hour tolerance

	// Verify refresh token is a valid JWT with correct claims
	claims, err := jwt.DecodeFiderClaims(refreshCookie.Value)
	Expect(err).IsNil()
	Expect(claims.UserID).Equals(mock.JonSnow.ID)
	Expect(claims.Origin).Equals(jwt.FiderClaimsOriginRefresh)
}

// TestRefresh_TokenRotation tests that refresh rotates the refresh token
func TestRefresh_TokenRotation(t *testing.T) {
	RegisterT(t)

	server := mock.NewServer()

	// Generate a valid refresh token
	refreshTokenExpires := time.Now().Add(7 * 24 * time.Hour)
	oldRefreshToken, err := jwt.Encode(jwt.FiderClaims{
		UserID:    mock.JonSnow.ID,
		UserName:  mock.JonSnow.Name,
		UserEmail: mock.JonSnow.Email,
		Origin:    jwt.FiderClaimsOriginRefresh,
		Metadata: jwt.Metadata{
			ExpiresAt: jwt.Time(refreshTokenExpires),
			IssuedAt:  jwt.Time(time.Now()),
		},
	})
	Expect(err).IsNil()

	// Mock user lookup
	bus.AddHandler(func(ctx context.Context, q *query.GetUserByID) error {
		if q.UserID == mock.JonSnow.ID {
			q.Result = mock.JonSnow
			return nil
		}
		return app.ErrNotFound
	})

	status, recorder := server.
		OnTenant(mock.DemoTenant).
		AddCookie(apiv1.RefreshCookieName, oldRefreshToken).
		ExecutePost(apiv1.Refresh(), `{}`)

	Expect(status).Equals(http.StatusOK)

	// Get the new refresh cookie from response
	var newRefreshCookie *http.Cookie
	for _, cookie := range recorder.Result().Cookies() {
		if cookie.Name == apiv1.RefreshCookieName {
			newRefreshCookie = cookie
			break
		}
	}

	// Verify new refresh cookie exists
	Expect(newRefreshCookie).IsNotNil()

	// Verify the new refresh token is different from the old one (rotation)
	Expect(newRefreshCookie.Value).NotEquals(oldRefreshToken)

	// Verify new refresh token is valid and has correct claims
	claims, err := jwt.DecodeFiderClaims(newRefreshCookie.Value)
	Expect(err).IsNil()
	Expect(claims.UserID).Equals(mock.JonSnow.ID)
	Expect(claims.Origin).Equals(jwt.FiderClaimsOriginRefresh)
}

// TestLogout_ClearsCookie tests that logout clears the refresh cookie
func TestLogout_ClearsCookie(t *testing.T) {
	RegisterT(t)

	server := mock.NewServer()

	status, recorder := server.
		OnTenant(mock.DemoTenant).
		ExecutePost(apiv1.Logout(), `{}`)

	Expect(status).Equals(http.StatusOK)

	// Get the refresh cookie from response
	var refreshCookie *http.Cookie
	for _, cookie := range recorder.Result().Cookies() {
		if cookie.Name == apiv1.RefreshCookieName {
			refreshCookie = cookie
			break
		}
	}

	// Verify cookie exists with cleared value
	Expect(refreshCookie).IsNotNil()
	Expect(refreshCookie.Value).Equals("")

	// Verify cookie has MaxAge -1 to delete it
	Expect(refreshCookie.MaxAge).Equals(-1)
}

// TestLogin_CORSHeaders tests that login endpoint sets correct CORS headers
func TestLogin_CORSHeaders(t *testing.T) {
	RegisterT(t)

	server := mock.NewServer()
	server.Use(middlewares.CORS())

	// Mock verification key lookup
	bus.AddHandler(func(ctx context.Context, q *query.GetVerificationByKey) error {
		if q.Key == "valid-key" {
			q.Result = &entity.EmailVerification{
				Email:     mock.JonSnow.Email,
				Name:      mock.JonSnow.Name,
				Key:       "valid-key",
				Kind:      enum.EmailVerificationKindSignIn,
				CreatedAt: time.Now(),
				ExpiresAt: time.Now().Add(15 * time.Minute),
			}
			return nil
		}
		return app.ErrNotFound
	})

	// Mock setting key as verified
	bus.AddHandler(func(ctx context.Context, c *cmd.SetKeyAsVerified) error {
		return nil
	})

	// Mock user lookup
	bus.AddHandler(func(ctx context.Context, q *query.GetUserByEmail) error {
		if q.Email == mock.JonSnow.Email {
			q.Result = mock.JonSnow
			return nil
		}
		return app.ErrNotFound
	})

	// Add Origin header to simulate cross-origin request
	status, recorder := server.
		OnTenant(mock.DemoTenant).
		AddHeader("Origin", "https://app.example.com").
		ExecutePost(apiv1.Login(),
			`{
				"email": "jon.snow@got.com",
				"verificationKey": "valid-key"
			}`)

	Expect(status).Equals(http.StatusOK)

	// Verify CORS headers are set in response
	headers := recorder.Result().Header

	// Access-Control-Allow-Origin should be set (specific origin or wildcard depending on config)
	allowOrigin := headers.Get("Access-Control-Allow-Origin")
	Expect(allowOrigin).IsNotEmpty()

	// Access-Control-Allow-Credentials should be true for cookie support
	allowCredentials := headers.Get("Access-Control-Allow-Credentials")
	Expect(allowCredentials).Equals("true")
}

// TestRefresh_CORSHeaders tests that refresh endpoint sets correct CORS headers
func TestRefresh_CORSHeaders(t *testing.T) {
	RegisterT(t)

	server := mock.NewServer()
	server.Use(middlewares.CORS())

	// Generate a valid refresh token
	refreshTokenExpires := time.Now().Add(7 * 24 * time.Hour)
	refreshToken, err := jwt.Encode(jwt.FiderClaims{
		UserID:    mock.JonSnow.ID,
		UserName:  mock.JonSnow.Name,
		UserEmail: mock.JonSnow.Email,
		Origin:    jwt.FiderClaimsOriginRefresh,
		Metadata: jwt.Metadata{
			ExpiresAt: jwt.Time(refreshTokenExpires),
			IssuedAt:  jwt.Time(time.Now()),
		},
	})
	Expect(err).IsNil()

	// Mock user lookup
	bus.AddHandler(func(ctx context.Context, q *query.GetUserByID) error {
		if q.UserID == mock.JonSnow.ID {
			q.Result = mock.JonSnow
			return nil
		}
		return app.ErrNotFound
	})

	// Add Origin header to simulate cross-origin request
	status, recorder := server.
		OnTenant(mock.DemoTenant).
		AddHeader("Origin", "https://app.example.com").
		AddCookie(apiv1.RefreshCookieName, refreshToken).
		ExecutePost(apiv1.Refresh(), `{}`)

	Expect(status).Equals(http.StatusOK)

	// Verify CORS headers are set in response
	headers := recorder.Result().Header

	// Access-Control-Allow-Origin should be set
	allowOrigin := headers.Get("Access-Control-Allow-Origin")
	Expect(allowOrigin).IsNotEmpty()

	// Access-Control-Allow-Credentials should be true for cookie support
	allowCredentials := headers.Get("Access-Control-Allow-Credentials")
	Expect(allowCredentials).Equals("true")
}

// TestLogout_CORSHeaders tests that logout endpoint sets correct CORS headers
func TestLogout_CORSHeaders(t *testing.T) {
	RegisterT(t)

	server := mock.NewServer()
	server.Use(middlewares.CORS())

	// Add Origin header to simulate cross-origin request
	status, recorder := server.
		OnTenant(mock.DemoTenant).
		AddHeader("Origin", "https://app.example.com").
		ExecutePost(apiv1.Logout(), `{}`)

	Expect(status).Equals(http.StatusOK)

	// Verify CORS headers are set in response
	headers := recorder.Result().Header

	// Access-Control-Allow-Origin should be set
	allowOrigin := headers.Get("Access-Control-Allow-Origin")
	Expect(allowOrigin).IsNotEmpty()

	// Access-Control-Allow-Credentials should be true for cookie support
	allowCredentials := headers.Get("Access-Control-Allow-Credentials")
	Expect(allowCredentials).Equals("true")
}
