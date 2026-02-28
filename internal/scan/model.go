package scan

// Kind represents the detected media class for an input file.
type Kind string

const (
	KindImage Kind = "image"
	KindVideo Kind = "video"
)

// Item is a normalized upload candidate.
type Item struct {
	Path      string
	SrcRoot   string
	ParentDir string
	Size      int64
	MTimeNS   int64
	Kind      Kind
}
