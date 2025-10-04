package apiv1

import (
	"net/http"
	"time"

	"github.com/getfider/fider/app"
	"github.com/getfider/fider/app/models/cmd"
	"github.com/getfider/fider/app/models/entity"
	"github.com/getfider/fider/app/models/enum"
	"github.com/getfider/fider/app/models/query"
	"github.com/getfider/fider/app/pkg/bus"
	"github.com/getfider/fider/app/pkg/env"
	"github.com/getfider/fider/app/pkg/errors"
	"github.com/getfider/fider/app/pkg/jwt"
	"github.com/getfider/fider/app/pkg/web"
)

const (
	// RefreshCookieName is the name of the refresh token cookie
	RefreshCookieName = "fider_refresh_token"
	
	// AccessTokenExpiration is the duration for access token validity (15 minutes)
	AccessTokenExpiration = 15 * time.Minute
	
	// RefreshTokenExpiration is the duration for refresh token validity (7 days)
	RefreshTokenExpiration = 7 * 24 * time.Hour
)

// Login authenticates a user with email and password, issuing JWT access token and refresh cookie
// POST /api/v1/auth/login
func Login() web.HandlerFunc {
	return func(c *web.Context) error {
		// Parse request body for email and verification key
		var input struct {
			Email           string `json:"email"`
			VerificationKey string `json:"verificationKey"`
		}
		
		if err := c.BindTo(&input); err != nil {
			return c.BadRequest(web.Map{
				"errors": []web.Map{
					{"message": "Invalid request body"},
				},
			})
		}

		// Validate verification key
		verifyKey := &query.GetVerificationByKey{
			Key:  input.VerificationKey,
			Kind: enum.EmailVerificationKindSignIn,
		}
		if err := bus.Dispatch(c, verifyKey); err != nil {
			if errors.Cause(err) == app.ErrNotFound {
				return c.Unauthorized(web.Map{
					"errors": []web.Map{
						{"message": "Invalid or expired verification key"},
					},
				})
			}
			return c.Failure(err)
		}

		// Verify email matches
		if verifyKey.Result.Email != input.Email {
			return c.Unauthorized(web.Map{
				"errors": []web.Map{
					{"message": "Email does not match verification key"},
				},
			})
		}

		// Get user by email
		userByEmail := &query.GetUserByEmail{Email: input.Email}
		if err := bus.Dispatch(c, userByEmail); err != nil {
			if errors.Cause(err) == app.ErrNotFound {
				return c.Unauthorized(web.Map{
					"errors": []web.Map{
						{"message": "User not found"},
					},
				})
			}
			return c.Failure(err)
		}

		user := userByEmail.Result

		// Mark verification key as used
		if err := bus.Dispatch(c, &cmd.SetKeyAsVerified{Key: input.VerificationKey}); err != nil {
			return c.Failure(err)
		}

		// Generate short-lived access token
		accessToken, err := generateAccessToken(user)
		if err != nil {
			return c.Failure(errors.Wrap(err, "failed to generate access token"))
		}

		// Generate and set refresh token cookie
		refreshToken, err := generateRefreshToken(user)
		if err != nil {
			return c.Failure(errors.Wrap(err, "failed to generate refresh token"))
		}

		setRefreshCookie(c, refreshToken)

		// Return access token and user info
		return c.Ok(web.Map{
			"accessToken": accessToken,
			"user": web.Map{
				"id":    user.ID,
				"name":  user.Name,
				"email": user.Email,
				"role":  user.Role,
			},
		})
	}
}

// Refresh exchanges a valid refresh token cookie for a new access token
// POST /api/v1/auth/refresh
func Refresh() web.HandlerFunc {
	return func(c *web.Context) error {
		// Get refresh token from cookie
		cookie, err := c.Request.Cookie(RefreshCookieName)
		if err != nil {
			return c.Unauthorized(web.Map{
				"errors": []web.Map{
					{"message": "Refresh token not found"},
				},
			})
		}

		// Decode refresh token
		claims, err := jwt.DecodeFiderClaims(cookie.Value)
		if err != nil {
			return c.Unauthorized(web.Map{
				"errors": []web.Map{
					{"message": "Invalid refresh token"},
				},
			})
		}

		// Verify token is refresh token (not access token)
		if claims.Origin != jwt.FiderClaimsOriginRefresh {
			return c.Unauthorized(web.Map{
				"errors": []web.Map{
					{"message": "Invalid token type"},
				},
			})
		}

		// Get user from database to ensure still active
		userByID := &query.GetUserByID{UserID: claims.UserID}
		if err := bus.Dispatch(c, userByID); err != nil {
			if errors.Cause(err) == app.ErrNotFound {
				return c.Unauthorized(web.Map{
					"errors": []web.Map{
						{"message": "User not found"},
					},
				})
			}
			return c.Failure(err)
		}

		user := userByID.Result

		// Generate new access token
		accessToken, err := generateAccessToken(user)
		if err != nil {
			return c.Failure(errors.Wrap(err, "failed to generate access token"))
		}

		// Rotate refresh token for security
		newRefreshToken, err := generateRefreshToken(user)
		if err != nil {
			return c.Failure(errors.Wrap(err, "failed to generate refresh token"))
		}

		setRefreshCookie(c, newRefreshToken)

		// Return new access token
		return c.Ok(web.Map{
			"accessToken": accessToken,
		})
	}
}

// Logout clears the refresh token cookie, invalidating the user's session
// POST /api/v1/auth/logout
func Logout() web.HandlerFunc {
	return func(c *web.Context) error {
		// Clear refresh token cookie by setting MaxAge to -1
		http.SetCookie(&c.Response, &http.Cookie{
			Name:     RefreshCookieName,
			Value:    "",
			Path:     "/",
			HttpOnly: true,
			Secure:   c.Request.IsSecure,
			SameSite: http.SameSiteNoneMode,
			MaxAge:   -1,
			Expires:  time.Now().Add(-100 * time.Hour),
		})

		return c.Ok(web.Map{
			"status": "ok",
		})
	}
}

// generateAccessToken creates a short-lived JWT access token for API authentication
func generateAccessToken(user *entity.User) (string, error) {
	token, err := jwt.Encode(jwt.FiderClaims{
		UserID:    user.ID,
		UserName:  user.Name,
		UserEmail: user.Email,
		Origin:    jwt.FiderClaimsOriginUI,
		Metadata: jwt.Metadata{
			ExpiresAt: jwt.Time(time.Now().Add(AccessTokenExpiration)),
		},
	})
	if err != nil {
		return "", err
	}
	return token, nil
}

// generateRefreshToken creates a long-lived JWT refresh token for token renewal
func generateRefreshToken(user *entity.User) (string, error) {
	token, err := jwt.Encode(jwt.FiderClaims{
		UserID:    user.ID,
		UserName:  user.Name,
		UserEmail: user.Email,
		Origin:    jwt.FiderClaimsOriginRefresh,
		Metadata: jwt.Metadata{
			ExpiresAt: jwt.Time(time.Now().Add(RefreshTokenExpiration)),
		},
	})
	if err != nil {
		return "", err
	}
	return token, nil
}

// setRefreshCookie sets the refresh token as an HttpOnly, Secure, SameSite=None cookie
func setRefreshCookie(c *web.Context, token string) {
	http.SetCookie(&c.Response, &http.Cookie{
		Name:     RefreshCookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   c.Request.IsSecure,
		SameSite: http.SameSiteNoneMode,
		Expires:  time.Now().Add(RefreshTokenExpiration),
	})
}
