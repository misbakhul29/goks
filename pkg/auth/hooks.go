package auth

import (
	"context"
	"net/http"
	"sync"
)

// HookFunc is a callback invoked when an authentication lifecycle event occurs.
type HookFunc func(ctx context.Context, user *User, r *http.Request)

var (
	hooksMu     sync.RWMutex
	loginHooks  []HookFunc
	logoutHooks []HookFunc
)

// OnLogin registers a callback to be called when a user logs in.
func OnLogin(fn HookFunc) {
	if fn == nil {
		return
	}
	hooksMu.Lock()
	defer hooksMu.Unlock()
	loginHooks = append(loginHooks, fn)
}

// OnLogout registers a callback to be called when a user logs out.
func OnLogout(fn HookFunc) {
	if fn == nil {
		return
	}
	hooksMu.Lock()
	defer hooksMu.Unlock()
	logoutHooks = append(logoutHooks, fn)
}

func triggerLoginHooks(ctx context.Context, user *User, r *http.Request) {
	hooksMu.RLock()
	defer hooksMu.RUnlock()
	for _, fn := range loginHooks {
		fn(ctx, user, r)
	}
}

func triggerLogoutHooks(ctx context.Context, user *User, r *http.Request) {
	hooksMu.RLock()
	defer hooksMu.RUnlock()
	for _, fn := range logoutHooks {
		fn(ctx, user, r)
	}
}
