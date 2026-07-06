package audit

import "time"

type LogEntry struct {
	RequestID    string
	UserID       string
	UserEmail    string
	Role         string
	Method       string
	Path         string
	QueryParams  string
	StatusCode   int
	RequestBody  string
	ResponseBody string
	IPAddress    string
	UserAgent    string
	DurationMs   int64
	CreatedAt    time.Time
}
