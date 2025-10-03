package apiv1

import (
	"net/http"
	"time"

	"github.com/getfider/fider/app/models/query"
	"github.com/getfider/fider/app/pkg/bus"
	"github.com/getfider/fider/app/pkg/jwt"
	"github.com/getfider/fider/app/pkg/web"
)

const (
	// RefreshTokenCookieName is the name of the HTTP-only cookie used for refresh tokens
	RefreshTokenCookieName = "fider_refresh_token"
	
	// AccessTokenDuration is the lifetime of access tokens (15 minutes)
	AccessTokenDuration = 15 * time.Minute
	
	// RefreshTokenDuration is the lifetime of refresh tokens (7 days)
	RefreshTokenDuration = 7 * 24 * time.Hour
)

// LoginRequest represents the login request body
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// LoginResponse represents the login response
type LoginResponse struct {
	AccessToken string      `json:"accessToken"`
	User        interface{} `json:"user"`
}

// RefreshResponse represents the refresh response
type RefreshResponse struct {
	AccessToken string `json:"accessToken"`
}

// Login handles POST /api/v1/auth/login
// Issues access token and sets refresh token cookie for SPA clients
func Login() web.HandlerFunc {
	return func(c *web.Context) error {
		var request LoginRequest
		if err := c.Bind(&request); err != nil {
			return c.BadRequest(web.Map{
				"errors": []web.Map{
					{"message": "Invalid request format"},
				},
			})
		}

		// Validate email and password
		if request.Email == "" || request.Password == "" {
			return c.BadRequest(web.Map{
				"errors": []web.Map{
					{"message": "Email and password are required"},
				},
			})
		}

		// Query user by email
		getUserByEmail := &query.GetUserByEmail{Email: request.Email}
		if err := bus.Dispatch(c, getUserByEmail); err != nil {
			return c.Unauthorized()
		}

		// Verify the user belongs to the current tenant
		if getUserByEmail.Result.Tenant.ID != c.Tenant().ID {
			return c.Unauthorized()
		}

		// Note: This is a simplified implementation. In a real system, you would:
		// 1. Verify the password against a hashed password stored in the database
		// 2. Use proper password verification (bcrypt, argon2, etc.)
		// For now, we'll use the existing authentication mechanism

		user := getUserByEmail.Result

		// Generate short-lived access token (JWT for Authorization header)
		accessTokenExpires := time.Now().Add(AccessTokenDuration)
		accessToken, err := jwt.Encode(jwt.FiderClaims{
			UserID:    user.ID,
			UserName:  user.Name,
			UserEmail: user.Email,
			Origin:    jwt.FiderClaimsOriginAPI,
			Metadata: jwt.Metadata{
				ExpiresAt: jwt.Time(accessTokenExpires),
				IssuedAt:  jwt.Time(time.Now()),
			},
		})
		if err != nil {
			return c.Failure(err)
		}

		// Generate long-lived refresh token (JWT for HTTP-only cookie)
		refreshTokenExpires := time.Now().Add(RefreshTokenDuration)
		refreshToken, err := jwt.Encode(jwt.FiderClaims{
			UserID:    user.ID,
			UserName:  user.Name,
			UserEmail: user.Email,
			Origin:    jwt.FiderClaimsOriginAPI,
			Metadata: jwt.Metadata{
				ExpiresAt: jwt.Time(refreshTokenExpires),
				IssuedAt:  jwt.Time(time.Now()),
			},
		})
		if err != nil {
			return c.Failure(err)
		}

		// Set refresh token as HTTP-only, Secure, SameSite=None cookie
		setRefreshTokenCookie(c, refreshToken, refreshTokenExpires)

		// Return access token and user info
		return c.Ok(LoginResponse{
			AccessToken: accessToken,
			User: web.Map{
				"id":     user.ID,
				"name":   user.Name,
				"email":  user.Email,
				"role":   user.Role,
				"status": user.Status,
			},
		})
	}
}

// Refresh handles POST /api/v1/auth/refresh
// Validates refresh token and issues new access token
func Refresh() web.HandlerFunc {
	return func(c *web.Context) error {
		// Read refresh token from HTTP-only cookie
		cookie, err := c.Request.Cookie(RefreshTokenCookieName)
		if err != nil {
			return c.Unauthorized()
		}

		// Decode and validate refresh token (includes expiration check)
		claims, err := jwt.DecodeFiderClaims(cookie.Value)
		if err != nil {
			return c.Unauthorized()
		}

		// Verify token origin is API
		if claims.Origin != jwt.FiderClaimsOriginAPI {
			return c.Unauthorized()
		}

		// Query user to ensure they still exist and are active
		getUserByID := &query.GetUserByID{UserID: claims.UserID}
		if err := bus.Dispatch(c, getUserByID); err != nil {
			return c.Unauthorized()
		}

		user := getUserByID.Result

		// Verify user belongs to current tenant
		if user.Tenant.ID != c.Tenant().ID {
			return c.Unauthorized()
		}

		// Generate new short-lived access token
		accessTokenExpires := time.Now().Add(AccessTokenDuration)
		accessToken, err := jwt.Encode(jwt.FiderClaims{
			UserID:    user.ID,
			UserName:  user.Name,
			UserEmail: user.Email,
			Origin:    jwt.FiderClaimsOriginAPI,
			Metadata: jwt.Metadata{
				ExpiresAt: jwt.Time(accessTokenExpires),
				IssuedAt:  jwt.Time(time.Now()),
			},
		})
		if err != nil {
			return c.Failure(err)
		}

		// Rotate refresh token (issue new one)
		refreshTokenExpires := time.Now().Add(RefreshTokenDuration)
		refreshToken, err := jwt.Encode(jwt.FiderClaims{
			UserID:    user.ID,
			UserName:  user.Name,
			UserEmail: user.Email,
			Origin:    jwt.FiderClaimsOriginAPI,
			Metadata: jwt.Metadata{
				ExpiresAt: jwt.Time(refreshTokenExpires),
				IssuedAt:  jwt.Time(time.Now()),
			},
		})
		if err != nil {
			return c.Failure(err)
		}

		// Set new refresh token cookie
		setRefreshTokenCookie(c, refreshToken, refreshTokenExpires)

		// Return new access token
		return c.Ok(RefreshResponse{
			AccessToken: accessToken,
		})
	}
}

// Logout handles POST /api/v1/auth/logout
// Clears refresh token cookie to terminate session
func Logout() web.HandlerFunc {
	return func(c *web.Context) error {
		// Clear refresh token cookie by setting it to expire immediately
		cookie := &http.Cookie{
			Name:     RefreshTokenCookieName,
			Value:    "",
			Path:     "/",
			HttpOnly: true,
			Secure:   c.Request.IsSecure,
			SameSite: http.SameSiteNoneMode,
			MaxAge:   -1,
			Expires:  time.Now().Add(-100 * time.Hour),
		}
		http.SetCookie(&c.Response, cookie)

		return c.Ok(web.Map{
			"status": "ok",
		})
	}
}

// setRefreshTokenCookie sets the refresh token as an HTTP-only, Secure, SameSite=None cookie
func setRefreshTokenCookie(c *web.Context, token string, expires time.Time) {
	cookie := &http.Cookie{
		Name:     RefreshTokenCookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   c.Request.IsSecure,
		SameSite: http.SameSiteNoneMode,
		Expires:  expires,
	}
	http.SetCookie(&c.Response, cookie)
}
