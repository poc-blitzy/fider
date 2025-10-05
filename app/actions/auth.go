package actions

import (
	"context"

	"github.com/getfider/fider/app/models/entity"
	"github.com/getfider/fider/app/models/enum"
	"github.com/getfider/fider/app/pkg/validate"
)

// LoginByEmail handles API-based login authentication with email and verification key
// Used by the cross-origin SPA authentication flow via POST /api/v1/auth/login
type LoginByEmail struct {
	Email           string `json:"email" format:"lower"`
	VerificationKey string `json:"verificationKey"`
}

// IsAuthorized returns true if current user is authorized to perform this action
// Login attempts are allowed for all users (unauthenticated)
func (action *LoginByEmail) IsAuthorized(ctx context.Context, user *entity.User) bool {
	return true
}

// Validate checks if the login request contains valid email and verification key
func (action *LoginByEmail) Validate(ctx context.Context, user *entity.User) *validate.Result {
	result := validate.Success()

	if action.Email == "" {
		result.AddFieldFailure("email", propertyIsRequired(ctx, "email"))
		return result
	}

	messages := validate.Email(ctx, action.Email)
	result.AddFieldFailure("email", messages...)

	if action.VerificationKey == "" {
		result.AddFieldFailure("verificationKey", propertyIsRequired(ctx, "verificationKey"))
	}

	return result
}

// GetEmail returns the email being verified
func (action *LoginByEmail) GetEmail() string {
	return action.Email
}

// GetName returns empty for this kind of process
func (action *LoginByEmail) GetName() string {
	return ""
}

// GetUser returns nil as this is an unauthenticated action
func (action *LoginByEmail) GetUser() *entity.User {
	return nil
}

// GetKind returns EmailVerificationKindSignIn for login flow
func (action *LoginByEmail) GetKind() enum.EmailVerificationKind {
	return enum.EmailVerificationKindSignIn
}
