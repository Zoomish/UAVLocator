package callback

import (
	"context"
	"fmt"

	"locatorbpla/internal/bot/callback/admin"
	"locatorbpla/internal/bot/callback/cancel"
	"locatorbpla/internal/bot/callback/confirm"
	"locatorbpla/internal/bot/callback/get"
	"locatorbpla/internal/bot/callback/set/hook"
	"locatorbpla/internal/bot/callback/set/locations"
	"locatorbpla/internal/bot/callback/set/notifications"
	"locatorbpla/internal/bot/cb"
	"locatorbpla/internal/bot/screen"
)

type Router struct {
	Screen        *screen.Screen
	Settings      *get.Settings
	Locations     *locations.Handler
	Webhook       *hook.Handler
	Notifications *notifications.Handler
	Cancel        *cancel.Handler
	Confirm       *confirm.Handler
	Admin         *admin.Handler
}

func (r *Router) Handle(ctx context.Context, q cb.Query) error {
	head, rest := cb.Split(q.Data)
	switch head {
	case "get":
		return r.onGet(ctx, q, rest)
	case "set":
		return r.onSet(ctx, q, rest)
	case "cancel":
		return r.Cancel.Handle(ctx, q, rest)
	case "confirm":
		return r.Confirm.Handle(ctx, q, rest)
	default:
		r.Screen.Toast(ctx, q.ID, "Неизвестная кнопка")
		return nil
	}
}

func (r *Router) onGet(ctx context.Context, q cb.Query, rest string) error {
	head, tail := cb.Split(rest)
	switch head {
	case "settings":
		return r.Settings.Handle(ctx, q)
	case "api":
		return r.Webhook.Open(ctx, q)
	case "admin":
		return r.onAdminGet(ctx, q, tail)
	default:
		r.Screen.Toast(ctx, q.ID, "Неизвестная кнопка")
		return nil
	}
}

func (r *Router) onAdminGet(ctx context.Context, q cb.Query, rest string) error {
	if rest == "" {
		return r.Admin.Open(ctx, q)
	}
	head, tail := cb.Split(rest)
	switch head {
	case "users":
		return r.Admin.UsersPage(ctx, q, tail)
	default:
		r.Screen.Toast(ctx, q.ID, "Неизвестная кнопка")
		return nil
	}
}

func (r *Router) onSet(ctx context.Context, q cb.Query, rest string) error {
	head, tail := cb.Split(rest)
	switch head {
	case "locations":
		return r.Locations.Handle(ctx, q)
	case "api":
		if tail == "off" {
			return r.Webhook.Disable(ctx, q)
		}
		return r.Webhook.Change(ctx, q)
	case "webhook":
		return r.Webhook.Change(ctx, q)
	case "notifications":
		switch tail {
		case "true":
			return r.Notifications.Handle(ctx, q, true)
		case "false":
			return r.Notifications.Handle(ctx, q, false)
		default:
			r.Screen.Toast(ctx, q.ID, "Неизвестная кнопка")
			return nil
		}
	case "admin":
		return r.Admin.Toggle(ctx, q, tail)
	default:
		r.Screen.Toast(ctx, q.ID, "Неизвестная кнопка")
		return nil
	}
}

func Denied(err error) bool {
	return admin.IsDenied(err)
}

func Wrap(err error) error {
	if err == nil || admin.IsDenied(err) {
		return nil
	}
	return fmt.Errorf("callback: %w", err)
}
