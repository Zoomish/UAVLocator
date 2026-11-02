package channel

import (
	"context"
	"fmt"
	"html"
	"log/slog"
	"sort"
	"strings"

	"github.com/gotd/td/tg"
)

const footer = "📡Локатор России - @locatorru"

const historyPage = 100

type Item struct {
	ID    int
	Plain string
	HTML  string
}

type Batch struct {
	Items   []Item
	Channel tg.InputChannelClass
}

type Reader struct {
	API     *tg.Client
	Channel string
	Log     *slog.Logger
}

func (r *Reader) Unread(ctx context.Context) (Batch, error) {
	ch, err := r.resolve(ctx)
	if err != nil {
		return Batch{}, err
	}
	input := &tg.InputChannel{ChannelID: ch.ID, AccessHash: ch.AccessHash}
	dlg, err := r.findDialog(ctx, ch.ID)
	if err != nil {
		return Batch{}, err
	}
	if dlg == nil {
		return Batch{}, fmt.Errorf("channel %s is not in dialogs", r.Channel)
	}
	if dlg.UnreadCount == 0 {
		r.Log.Info("channel has no unread")
		return Batch{Channel: input}, nil
	}
	full, err := r.API.ChannelsGetFullChannel(ctx, input)
	if err != nil {
		return Batch{}, fmt.Errorf("full channel: %w", err)
	}
	cf, ok := full.FullChat.(*tg.ChannelFull)
	if !ok {
		return Batch{}, fmt.Errorf("unexpected full chat")
	}
	limit := historyPage
	req := &tg.MessagesGetHistoryRequest{
		Peer:  &tg.InputPeerChannel{ChannelID: ch.ID, AccessHash: ch.AccessHash},
		Limit: limit,
	}
	if cf.ReadInboxMaxID > 0 {
		req.OffsetID = cf.ReadInboxMaxID
		req.AddOffset = -limit
	}
	hist, err := r.API.MessagesGetHistory(ctx, req)
	if err != nil {
		return Batch{}, fmt.Errorf("history: %w", err)
	}
	var items []Item
	for _, raw := range messagesOf(hist) {
		id, ok := messageID(raw)
		if !ok || id <= cf.ReadInboxMaxID {
			continue
		}
		item := Item{ID: id}
		if msg, ok := raw.(*tg.Message); ok {
			plain, rendered, err := r.render(ctx, input, msg)
			if err != nil {
				return Batch{}, err
			}
			item.Plain = plain
			item.HTML = rendered
		}
		items = append(items, item)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].ID < items[j].ID })
	r.Log.Info("unread channel posts", "count", len(items))
	return Batch{Items: items, Channel: input}, nil
}

func (r *Reader) MarkRead(ctx context.Context, channel tg.InputChannelClass, maxID int) error {
	if maxID == 0 || channel == nil {
		return nil
	}
	if _, err := r.API.ChannelsReadHistory(ctx, &tg.ChannelsReadHistoryRequest{
		Channel: channel,
		MaxID:   maxID,
	}); err != nil {
		return fmt.Errorf("read history: %w", err)
	}
	r.Log.Info("channel marked read", "max_id", maxID)
	return nil
}

func (r *Reader) resolve(ctx context.Context) (*tg.Channel, error) {
	resolved, err := r.API.ContactsResolveUsername(ctx, &tg.ContactsResolveUsernameRequest{Username: r.Channel})
	if err != nil {
		return nil, fmt.Errorf("resolve channel: %w", err)
	}
	peer, ok := resolved.Peer.(*tg.PeerChannel)
	if !ok {
		return nil, fmt.Errorf("channel %s was not resolved", r.Channel)
	}
	for _, chat := range resolved.Chats {
		ch, ok := chat.(*tg.Channel)
		if ok && ch.ID == peer.ChannelID {
			return ch, nil
		}
	}
	return nil, fmt.Errorf("channel %s was not resolved", r.Channel)
}

func (r *Reader) render(ctx context.Context, channel tg.InputChannelClass, msg *tg.Message) (string, string, error) {
	body := strip(msg.Message)
	reply := ""
	hdr, ok := msg.GetReplyTo()
	if ok {
		if replyHdr, ok := hdr.(*tg.MessageReplyHeader); ok {
			if replyID, ok := replyHdr.GetReplyToMsgID(); ok && replyID != 0 {
				text, err := r.replyText(ctx, channel, replyID)
				if err != nil {
					return "", "", fmt.Errorf("reply %d: %w", replyID, err)
				}
				reply = text
			}
		}
	}
	plain := strings.TrimSpace(strings.TrimSpace(body) + " " + strings.TrimSpace(reply))
	if plain == "" {
		return "", "", nil
	}
	rendered := html.EscapeString(body)
	if reply != "" {
		rendered = "<blockquote expandable>" + html.EscapeString(reply) + "</blockquote>" + rendered
	}
	return plain, rendered, nil
}

func (r *Reader) replyText(ctx context.Context, channel tg.InputChannelClass, id int) (string, error) {
	res, err := r.API.ChannelsGetMessages(ctx, &tg.ChannelsGetMessagesRequest{
		Channel: channel,
		ID:      []tg.InputMessageClass{&tg.InputMessageID{ID: id}},
	})
	if err != nil {
		return "", err
	}
	for _, raw := range messagesOf(res) {
		msg, ok := raw.(*tg.Message)
		if ok && msg.ID == id {
			return strip(msg.Message), nil
		}
	}
	return "", nil
}

func strip(text string) string {
	return strings.ReplaceAll(text, footer, "")
}

func messageID(raw tg.MessageClass) (int, bool) {
	switch m := raw.(type) {
	case *tg.Message:
		return m.ID, true
	case *tg.MessageService:
		return m.ID, true
	case *tg.MessageEmpty:
		return m.ID, true
	default:
		return 0, false
	}
}

func messagesOf(res tg.MessagesMessagesClass) []tg.MessageClass {
	switch m := res.(type) {
	case *tg.MessagesMessages:
		return m.Messages
	case *tg.MessagesMessagesSlice:
		return m.Messages
	case *tg.MessagesChannelMessages:
		return m.Messages
	default:
		return nil
	}
}
