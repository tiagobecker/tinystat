package models

// ActionSummary contains a summary of actions over several passed intervals
type ActionSummary struct {
	Hour  int64 `json:"hour"`
	Day   int64 `json:"day"`
	Week  int64 `json:"week"`
	Month int64 `json:"month"`
	Year  int64 `json:"year"`
}
