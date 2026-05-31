package kernel

// RowError describes a single failed row in a bulk import. Row is the 1-based
// line number within the uploaded data (header excluded), Key is a human-friendly
// identifier for the row (e.g. the department/member name or username), and
// Message is the validation/persistence reason it failed.
type RowError struct {
	Row     int    `json:"row"`
	Key     string `json:"key,omitempty"`
	Message string `json:"message"`
}

// BulkResult is the uniform summary returned by every bulk import endpoint. The
// HTTP layer returns it with 200 even on partial failure — bad rows are reported
// here rather than failing the whole request.
type BulkResult struct {
	Total     int        `json:"total"`
	Succeeded int        `json:"succeeded"`
	Failed    int        `json:"failed"`
	Errors    []RowError `json:"errors"`
}

// NewBulkResult returns a BulkResult with a non-nil Errors slice so it serializes
// as [] rather than null.
func NewBulkResult() *BulkResult {
	return &BulkResult{Errors: []RowError{}}
}

// Fail records a failed row.
func (r *BulkResult) Fail(row int, key, message string) {
	r.Failed++
	r.Errors = append(r.Errors, RowError{Row: row, Key: key, Message: message})
}

// Ok records a succeeded row.
func (r *BulkResult) Ok() { r.Succeeded++ }

// Page is the uniform paginated-list envelope: a typed data slice plus the total
// matching-row count and the (normalized) page/page_size that produced it.
type Page[T any] struct {
	Data     []T `json:"data"`
	Total    int `json:"total"`
	Page     int `json:"page"`
	PageSize int `json:"page_size"`
}
