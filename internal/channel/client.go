package channel

import (
	"fmt"

	"github.com/gotd/td/session"
	"github.com/gotd/td/telegram"
	"go.uber.org/zap"
)

func New(apiID int, apiHash string, storage session.Storage) (*telegram.Client, error) {
	if apiID == 0 || apiHash == "" {
		return nil, fmt.Errorf("API_ID and API_HASH are required")
	}
	if storage == nil {
		return nil, fmt.Errorf("session storage is required")
	}
	return telegram.NewClient(apiID, apiHash, telegram.Options{
		SessionStorage: storage,
		Logger:         zap.NewNop(),
	}), nil
}
