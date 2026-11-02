package notify

import (
	"context"
	"errors"
	"fmt"
	"html"
	"log/slog"
	"strings"
	"sync"
	"time"
	"unicode/utf16"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"

	"locatorbpla/internal/domain"
)

const (
	maxAttempts       = 5
	maxFloodWait      = 15 * time.Minute
	drainGrace        = 20 * time.Second
	telegramTextLimit = 4096
)

type Blocker interface {
	Block(ctx context.Context, tgID int64) error
}

type Poster interface {
	Push(url, body string)
}

type Queue struct {
	Bot    *bot.Bot
	Users  Blocker
	Log    *slog.Logger
	Alert  func(error)
	Out    Poster
	Denied func(context.Context, int64)
	Gap    time.Duration

	ch   chan post
	mu   sync.Mutex
	last domain.PostStats
}

type post struct {
	html    string
	matches []domain.Match
}

func NewQueue(users Blocker, log *slog.Logger) *Queue {
	return &Queue{
		Users: users,
		Log:   log,
		ch:    make(chan post, 1),
	}
}

func (q *Queue) Enqueue(ctx context.Context, html string, matches []domain.Match) error {
	job := post{html: html, matches: matches}
	select {
	case <-ctx.Done():
		return ctx.Err()
	case q.ch <- job:
		return nil
	}
}

func (q *Queue) Latest() domain.PostStats {
	q.mu.Lock()
	defer q.mu.Unlock()
	return q.last
}

func (q *Queue) Run(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			q.drainRest(ctx)
			return ctx.Err()
		case job := <-q.ch:
			sendCtx, cancel := graceContext(ctx, drainGrace)
			q.deliver(sendCtx, job)
			cancel()
		}
	}
}

func (q *Queue) drainRest(parent context.Context) {
	sendCtx, cancel := graceContext(parent, drainGrace)
	defer cancel()
	for {
		select {
		case <-sendCtx.Done():
			return
		case job := <-q.ch:
			q.deliver(sendCtx, job)
		default:
			return
		}
	}
}

func graceContext(parent context.Context, grace time.Duration) (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		select {
		case <-parent.Done():
			timer := time.NewTimer(grace)
			select {
			case <-timer.C:
				cancel()
			case <-ctx.Done():
				timer.Stop()
			}
		case <-ctx.Done():
		}
	}()
	return ctx, cancel
}

type recip struct {
	match    domain.Match
	floods   int
	attempts int
}

func (q *Queue) deliver(ctx context.Context, job post) {
	stats := domain.PostStats{Present: true}
	defer func() {
		if rec := recover(); rec != nil {
			q.Log.Error("delivery panic", "err", rec)
		}
		q.publish(stats)
	}()
	queue := make([]recip, len(job.matches))
	for i, match := range job.matches {
		queue[i] = recip{match: match}
	}
	var deferred []recip
	var maxWait time.Duration
	globalPaused := false
	consecFlood := false
	alerted := false
	abandon := func(left int) {
		stats.Undelivered += left
		q.publish(stats)
		if alerted || q.Alert == nil {
			return
		}
		alerted = true
		q.Alert(fmt.Errorf("delivery flood wait longer than %s, skipped %d", maxFloodWait, left))
	}
	for len(queue) > 0 {
		if ctx.Err() != nil {
			return
		}
		cur := queue[0]
		queue = queue[1:]
		body, err := q.sendOne(ctx, job.html, cur.match)
		if ctx.Err() != nil {
			return
		}
		if err == nil {
			stats.Sent++
			consecFlood = false
			q.publish(stats)
			q.pushHook(cur.match, body)
			if sleep(ctx, q.gap()) != nil {
				return
			}
			continue
		}
		var flood *RetryAfterError
		if errors.As(err, &flood) && flood.Wait > 0 {
			stats.FloodWaits++
			q.publish(stats)
			cur.floods++
			if cur.floods >= 2 {
				stats.Undelivered++
				consecFlood = false
				q.publish(stats)
				continue
			}
			if flood.Wait > maxFloodWait {
				abandon(1 + len(deferred) + len(queue))
				return
			}
			if flood.Wait > maxWait {
				maxWait = flood.Wait
			}
			deferred = append(deferred, cur)
			if consecFlood && !globalPaused {
				globalPaused = true
				q.Log.Warn("delivery global flood", "wait", maxWait.String())
				if sleep(ctx, maxWait) != nil {
					return
				}
				maxWait = 0
				consecFlood = false
				queue = append(deferred, queue...)
				deferred = nil
				continue
			}
			consecFlood = true
			q.Log.Warn("delivery flood wait", "telegram_id", cur.match.User.TgID, "wait", flood.Wait.String())
			continue
		}
		consecFlood = false
		if blocked(err) {
			if berr := q.Users.Block(ctx, cur.match.User.TgID); berr != nil {
				q.Log.Error("block user", "telegram_id", cur.match.User.TgID, "err", berr)
			}
			if q.Denied != nil {
				q.Denied(ctx, cur.match.User.TgID)
			}
			stats.Blocked++
			q.publish(stats)
			continue
		}
		cur.attempts++
		q.Log.Error("send alert", "telegram_id", cur.match.User.TgID, "attempts", cur.attempts, "err", err)
		if cur.attempts >= maxAttempts {
			stats.Undelivered++
			q.publish(stats)
			continue
		}
		queue = append([]recip{cur}, queue...)
	}
	if len(deferred) == 0 {
		return
	}
	if maxWait > 0 && sleep(ctx, maxWait) != nil {
		return
	}
	for _, cur := range deferred {
		if ctx.Err() != nil {
			return
		}
		body, err := q.sendOne(ctx, job.html, cur.match)
		if ctx.Err() != nil {
			return
		}
		if err == nil {
			stats.Sent++
			q.publish(stats)
			q.pushHook(cur.match, body)
			if sleep(ctx, q.gap()) != nil {
				return
			}
			continue
		}
		var flood *RetryAfterError
		if errors.As(err, &flood) && flood.Wait > 0 {
			stats.FloodWaits++
		}
		if blocked(err) {
			if berr := q.Users.Block(ctx, cur.match.User.TgID); berr != nil {
				q.Log.Error("block user", "telegram_id", cur.match.User.TgID, "err", berr)
			}
			if q.Denied != nil {
				q.Denied(ctx, cur.match.User.TgID)
			}
			stats.Blocked++
			q.publish(stats)
			continue
		}
		stats.Undelivered++
		q.publish(stats)
	}
}

func (q *Queue) pushHook(match domain.Match, body string) {
	if q.Out == nil || match.User.WebhookURL == "" || match.User.TgID < 0 {
		return
	}
	q.Out.Push(match.User.WebhookURL, body)
}

func (q *Queue) sendOne(ctx context.Context, postHTML string, match domain.Match) (string, error) {
	body, asHTML := fitMessage(match.Pattern, postHTML)
	return body, q.send(ctx, match.User.TgID, body, asHTML)
}

func (q *Queue) publish(stats domain.PostStats) {
	q.mu.Lock()
	q.last = stats
	q.mu.Unlock()
}

func (q *Queue) gap() time.Duration {
	if q.Gap > 0 {
		return q.Gap
	}
	return 40 * time.Millisecond
}

func (q *Queue) send(ctx context.Context, tgID int64, body string, asHTML bool) error {
	params := &bot.SendMessageParams{ChatID: tgID, Text: body}
	if asHTML {
		params.ParseMode = models.ParseModeHTML
	}
	_, err := q.Bot.SendMessage(ctx, params)
	if err == nil {
		return nil
	}
	if wait, ok := retryAfter(err); ok {
		return &RetryAfterError{Wait: wait}
	}
	return fmt.Errorf("send alert to %d: %w", tgID, err)
}

func Message(pattern, postHTML string) string {
	if pattern == "" {
		return postHTML
	}
	return "совпало: <b>" + html.EscapeString(pattern) + "</b>\n\n" + postHTML
}

func fitMessage(pattern, postHTML string) (string, bool) {
	htmlMsg := Message(pattern, postHTML)
	if utf16Len(htmlMsg) <= telegramTextLimit {
		return htmlMsg, true
	}
	plain := "совпало: " + pattern + "\n\n" + plainFromHTML(postHTML)
	if pattern == "" {
		plain = plainFromHTML(postHTML)
	}
	return clipUTF16(strings.TrimSpace(plain), telegramTextLimit), false
}

func plainFromHTML(s string) string {
	s = strings.ReplaceAll(s, "<blockquote expandable>", "")
	s = strings.ReplaceAll(s, "</blockquote>", "\n")
	s = strings.ReplaceAll(s, "<b>", "")
	s = strings.ReplaceAll(s, "</b>", "")
	return strings.TrimSpace(html.UnescapeString(s))
}

func clipUTF16(s string, limit int) string {
	if utf16Len(s) <= limit {
		return s
	}
	if limit < 1 {
		return ""
	}
	units := utf16.Encode([]rune(s))
	cut := limit - 1
	if cut > len(units) {
		cut = len(units)
	}
	return string(utf16.Decode(units[:cut])) + "…"
}

func utf16Len(s string) int {
	return len(utf16.Encode([]rune(s)))
}

func sleep(ctx context.Context, wait time.Duration) error {
	timer := time.NewTimer(wait)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func blocked(err error) bool {
	if errors.Is(err, bot.ErrorForbidden) {
		return true
	}
	msg := err.Error()
	return strings.Contains(msg, "bot was blocked") || strings.Contains(msg, "user is deactivated") || strings.Contains(msg, "Forbidden")
}

type RetryAfterError struct {
	Wait time.Duration
}

func (e *RetryAfterError) Error() string {
	return "telegram flood wait"
}

func retryAfter(err error) (time.Duration, bool) {
	var tooMany *bot.TooManyRequestsError
	if errors.As(err, &tooMany) && tooMany != nil && tooMany.RetryAfter > 0 {
		return time.Duration(tooMany.RetryAfter) * time.Second, true
	}
	return 0, false
}
