package upload

// DuplicatePolicy controls behavior for already-sent media.
type DuplicatePolicy string

const (
	DuplicateSkip   DuplicatePolicy = "skip"
	DuplicateAsk    DuplicatePolicy = "ask"
	DuplicateUpload DuplicatePolicy = "upload"
)
