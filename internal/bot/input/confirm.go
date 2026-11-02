package input

import (
	"context"
	"fmt"

	"locatorbpla/internal/bot/view"
	"locatorbpla/internal/domain"
)

type confirmKind struct {
	title    string
	question string
	save     string
}

func kindOf(waiting string) (confirmKind, bool) {
	switch waiting {
	case domain.WaitLocations:
		return confirmKind{
			title:    "Проверьте локации",
			question: "Сохранить этот список или ввести заново?",
			save:     "✅ Сохранить",
		}, true
	case domain.WaitError:
		return confirmKind{
			title:    "Проверьте текст",
			question: "Отправить это сообщение или ввести заново?",
			save:     "✅ Отправить",
		}, true
	default:
		return confirmKind{}, false
	}
}

func (r *Router) ask(ctx context.Context, in Incoming, phrase string) error {
	kind, ok := kindOf(in.Waiting)
	if !ok {
		return fmt.Errorf("no confirm step for %q", in.Waiting)
	}
	text, kb := view.Confirm(kind.title, phrase, kind.question, kind.save)
	return r.preview(ctx, in, phrase, text, kb)
}
