package service

import (
	"context"
	"crypto/subtle"

	"github.com/alexedwards/scs/v2"
)

const (
	// sessionKeyAuthenticated marks a session as an authenticated admin session.
	sessionKeyAuthenticated = "authenticated"
)

// AuthOptions holds the settings of the AuthService.
type AuthOptions struct {
	// Password is the admin password a login has to match.
	Password string
}

// AuthService checks the admin password and keeps the login in the session.
type AuthService struct {
	options AuthOptions
	sm      *scs.SessionManager
}

// NewAuthService creates an AuthService.
func NewAuthService(options AuthOptions, sm *scs.SessionManager) *AuthService {
	return &AuthService{
		options: options,
		sm:      sm,
	}
}

// Login reports whether a candidate matches the configured password.
func (s *AuthService) Login(ctx context.Context, candidate string) (bool, error) {
	// A constant-time comparison does not reveal through its duration how much of the password matched
	if subtle.ConstantTimeCompare([]byte(candidate), []byte(s.options.Password)) != 1 {
		return false, nil
	}

	if err := s.sm.RenewToken(ctx); err != nil {
		return false, err
	}
	s.sm.Put(ctx, sessionKeyAuthenticated, true)

	return true, nil
}

// Logout destroys the current session.
func (s *AuthService) Logout(ctx context.Context) error {
	return s.sm.Destroy(ctx)
}

// Authenticated reports whether the current session is an authenticated admin session.
func (s *AuthService) Authenticated(ctx context.Context) bool {
	return s.sm.GetBool(ctx, sessionKeyAuthenticated)
}
