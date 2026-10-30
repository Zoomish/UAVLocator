package channel

import (
	"context"
	"fmt"

	"github.com/gotd/td/tg"
)

const (
	dialogPage = 100
	maxPages   = 30
)

func (r *Reader) findDialog(ctx context.Context, channelID int64) (*tg.Dialog, error) {
	for _, archived := range []bool{false, true} {
		dlg, err := r.scanFolder(ctx, channelID, archived)
		if err != nil {
			return nil, err
		}
		if dlg != nil {
			return dlg, nil
		}
	}
	return nil, nil
}

func (r *Reader) scanFolder(ctx context.Context, channelID int64, archived bool) (*tg.Dialog, error) {
	var offset tg.InputPeerClass = &tg.InputPeerEmpty{}
	offsetDate, offsetID := 0, 0
	for page := 0; page < maxPages; page++ {
		req := &tg.MessagesGetDialogsRequest{
			OffsetDate: offsetDate,
			OffsetID:   offsetID,
			OffsetPeer: offset,
			Limit:      dialogPage,
		}
		if archived {
			req.SetFolderID(1)
		}
		res, err := r.API.MessagesGetDialogs(ctx, req)
		if err != nil {
			return nil, fmt.Errorf("dialogs: %w", err)
		}
		dialogs, messages, chats, users := dialogsOf(res)
		for _, raw := range dialogs {
			dlg, ok := raw.(*tg.Dialog)
			if !ok {
				continue
			}
			peer, ok := dlg.Peer.(*tg.PeerChannel)
			if ok && peer.ChannelID == channelID {
				return dlg, nil
			}
		}
		if len(dialogs) < dialogPage {
			return nil, nil
		}
		last := lastDialog(dialogs)
		if last == nil {
			return nil, nil
		}
		next := inputPeer(last.Peer, chats, users)
		nextDate := messageDate(messages, last.TopMessage)
		if next == nil || (nextDate == offsetDate && last.TopMessage == offsetID) {
			return nil, nil
		}
		offset = next
		offsetDate = nextDate
		offsetID = last.TopMessage
	}
	return nil, nil
}

func dialogsOf(res tg.MessagesDialogsClass) ([]tg.DialogClass, []tg.MessageClass, []tg.ChatClass, []tg.UserClass) {
	switch m := res.(type) {
	case *tg.MessagesDialogs:
		return m.Dialogs, m.Messages, m.Chats, m.Users
	case *tg.MessagesDialogsSlice:
		return m.Dialogs, m.Messages, m.Chats, m.Users
	default:
		return nil, nil, nil, nil
	}
}

func inputPeer(peer tg.PeerClass, chats []tg.ChatClass, users []tg.UserClass) tg.InputPeerClass {
	switch p := peer.(type) {
	case *tg.PeerChannel:
		for _, chat := range chats {
			ch, ok := chat.(*tg.Channel)
			if ok && ch.ID == p.ChannelID {
				return &tg.InputPeerChannel{ChannelID: ch.ID, AccessHash: ch.AccessHash}
			}
		}
	case *tg.PeerChat:
		return &tg.InputPeerChat{ChatID: p.ChatID}
	case *tg.PeerUser:
		for _, raw := range users {
			user, ok := raw.(*tg.User)
			if ok && user.ID == p.UserID {
				return &tg.InputPeerUser{UserID: user.ID, AccessHash: user.AccessHash}
			}
		}
	}
	return nil
}

func lastDialog(dialogs []tg.DialogClass) *tg.Dialog {
	for i := len(dialogs) - 1; i >= 0; i-- {
		if dlg, ok := dialogs[i].(*tg.Dialog); ok {
			return dlg
		}
	}
	return nil
}

func messageDate(messages []tg.MessageClass, id int) int {
	for _, raw := range messages {
		switch m := raw.(type) {
		case *tg.Message:
			if m.ID == id {
				return m.Date
			}
		case *tg.MessageService:
			if m.ID == id {
				return m.Date
			}
		}
	}
	return 0
}
