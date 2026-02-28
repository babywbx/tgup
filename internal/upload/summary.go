package upload

// Summary captures normalized run outcome statistics.
type Summary struct {
	Total    int
	Sent     int
	Skipped  int
	Failed   int
	Canceled bool
}
