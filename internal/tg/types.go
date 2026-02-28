package tg

// ResolvedTarget is an app-owned target projection.
type ResolvedTarget struct {
	Kind string
	ID   int64
	Raw  string
}

// SentMessage is an app-owned projection of Telegram message results.
type SentMessage struct {
	ID        int
	MediaKind string
	FileName  string
	Size      int64
}
