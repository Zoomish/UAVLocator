package cb

type Query struct {
	ID          string
	Data        string
	ChatID      int64
	MessageID   int
	EphemeralID int
	Actor       int64
	Name        string
	Group       bool
}
