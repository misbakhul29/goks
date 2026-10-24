package auth

import (
	"errors"
	"net/http"

	"github.com/misbakhul29/goks/pkg/router"
)

func init() {
	router.RegisterAuthUserResolver(func(r *http.Request) any {
		return CurrentUser(r)
	})
}

// FromContext extracts the authenticated User from a router.Context.
func FromContext(ctx *router.Context) *User {
	if ctx == nil || ctx.Request() == nil {
		return nil
	}
	return CurrentUser(ctx.Request())
}

// IsAuthenticated reports whether the current request is authenticated.
func IsAuthenticated(ctx *router.Context) bool {
	return FromContext(ctx) != nil
}

// LoginFromContext creates a session and cookie for the given user using router.Context.
func LoginFromContext(ctx *router.Context, user *User, opts ...LoginOption) (*Session, error) {
	if ctx == nil || ctx.Response() == nil || ctx.Request() == nil {
		return nil, errors.New("auth: router context, response, or request is nil")
	}
	return Default.Login(ctx.Response(), ctx.Request(), user, opts...)
}

// LogoutFromContext clears the user session and cookie using router.Context.
func LogoutFromContext(ctx *router.Context) {
	if ctx != nil && ctx.Response() != nil {
		Default.Logout(ctx.Response(), ctx.Request())
	}
}
