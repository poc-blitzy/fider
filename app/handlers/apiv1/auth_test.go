package apiv1_test

import (
	"context"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/getfider/fider/app"
	"github.com/getfider/fider/app/handlers/apiv1"
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
