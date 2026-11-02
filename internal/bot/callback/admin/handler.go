package admin

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"locatorbpla/internal/bot/cb"
	"locatorbpla/internal/bot/screen"
	"locatorbpla/internal/bot/view"
	"locatorbpla/internal/domain"
	"locatorbpla/internal/repository"
)

type PostStats interface {
	Latest() domain.PostStats
}

type Handler struct {
	Users  *repository.Users
	Posts  PostStats
	Screen *screen.Screen
}

func (h *Handler) Open(ctx context.Context, q cb.Query) error {
	if _, err := h.require(ctx, q); err != nil {
		return err
	}
	if err := h.home(ctx, q); err != nil {
		return err
	}
	h.Screen.Toast(ctx, q.ID, "Меню администратора")
	return nil
}

func (h *Handler) UsersPage(ctx context.Context, q cb.Query, _ string) error {
	if _, err := h.require(ctx, q); err != nil {
		return err
	}
	h.Screen.Toast(ctx, q.ID, "Списка больше нет")
	return nil
}

func (h *Handler) Toggle(ctx context.Context, q cb.Query, rest string) error {
	if _, err := h.require(ctx, q); err != nil {
		return err
	}
	target, page, err := parseTarget(rest)
	if err != nil {
		h.Screen.Toast(ctx, q.ID, "Некорректный пользователь")
		return nil
	}
	user, err := h.Users.ToggleAdmin(ctx, target)
	if err != nil {
		return fmt.Errorf("toggle admin: %w", err)
	}
	users, err := h.Users.List(ctx)
	if err != nil {
		return fmt.Errorf("list users: %w", err)
	}
	text, kb := view.AdminUsers(users, page)
	if err := h.Screen.EditFunctional(ctx, q.ChatID, q.MessageID, text, kb); err != nil {
		return fmt.Errorf("admin refresh: %w", err)
	}
	verb := "убираете"
	if user.Admin {
		verb = "делаете"
	}
	h.Screen.Toast(ctx, q.ID, "Вы "+verb+" пользователя админом")
	return nil
}

func (h *Handler) home(ctx context.Context, q cb.Query) error {
	audience, err := h.Users.Audience(ctx)
	if err != nil {
		return fmt.Errorf("audience: %w", err)
	}
	post := h.Posts.Latest()
	text, kb := view.AdminHome(audience, post)
	if err := h.Screen.EditFunctional(ctx, q.ChatID, q.MessageID, text, kb); err != nil {
		return fmt.Errorf("admin home: %w", err)
	}
	return nil
}

func (h *Handler) require(ctx context.Context, q cb.Query) (domain.User, error) {
	if q.Group {
		h.Screen.Toast(ctx, q.ID, "Недоступно")
		return domain.User{}, errDenied
	}
	user, err := h.Users.Find(ctx, q.ChatID)
	if err != nil {
		return domain.User{}, fmt.Errorf("admin user: %w", err)
	}
	if !user.Admin {
		h.Screen.Toast(ctx, q.ID, "Недоступно")
		return domain.User{}, errDenied
	}
	return user, nil
}

var errDenied = fmt.Errorf("not admin")

func IsDenied(err error) bool {
	return err == errDenied
}

func parseTarget(rest string) (int64, int, error) {
	idPart, pagePart, _ := strings.Cut(rest, "-")
	id, err := strconv.ParseInt(idPart, 10, 64)
	if err != nil || id == 0 {
		return 0, 0, fmt.Errorf("id")
	}
	page := 0
	if pagePart != "" {
		page, err = strconv.Atoi(pagePart)
		if err != nil || page < 0 {
			page = 0
		}
	}
	return id, page, nil
}
