package kernel

const (
	defaultPage     = 1
	defaultPageSize = 20
	maxPageSize     = 200
)

// Pagination is a value object passed to Query application services.
type Pagination struct {
	Page     int `form:"page"      json:"page"`
	PageSize int `form:"page_size" json:"page_size"`
}

// Normalize clamps invalid input to safe defaults; returns a new Pagination.
// Always call Normalize() before using Pagination in a repository.
func (p Pagination) Normalize() Pagination {
	if p.Page < 1 {
		p.Page = defaultPage
	}
	if p.PageSize < 1 {
		p.PageSize = defaultPageSize
	}
	if p.PageSize > maxPageSize {
		p.PageSize = maxPageSize
	}
	return p
}

// Offset returns the SQL OFFSET equivalent.
func (p Pagination) Offset() int {
	return (p.Page - 1) * p.PageSize
}
