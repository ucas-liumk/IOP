package errors

// Kind categorizes errors for HTTP status mapping and observability.
type Kind int

const (
	KindUnknown   Kind = iota
	KindBusiness       // 400 — business rule violation, safe to show user
	KindParam          // 400 — invalid request parameter
	KindAuth           // 401 — unauthenticated
	KindForbidden      // 403 — authenticated but not authorized
	KindNotFound       // 404 — resource missing or hidden by tenant scope
	KindConflict       // 409 — idempotency / version conflict
	KindRateLimit      // 429 — rate limited
	KindDatabase       // 500 — DB / persistence failure
	KindExternal       // 502 — external service failure
	KindInternal       // 500 — programming error / panic recovered
)

func (k Kind) String() string {
	switch k {
	case KindBusiness:
		return "business"
	case KindParam:
		return "param"
	case KindAuth:
		return "auth"
	case KindForbidden:
		return "forbidden"
	case KindNotFound:
		return "not_found"
	case KindConflict:
		return "conflict"
	case KindRateLimit:
		return "rate_limit"
	case KindDatabase:
		return "database"
	case KindExternal:
		return "external"
	case KindInternal:
		return "internal"
	default:
		return "unknown"
	}
}

// HTTPStatus maps Kind to an HTTP status code (used by interface/apiresp).
func (k Kind) HTTPStatus() int {
	switch k {
	case KindBusiness, KindParam:
		return 400
	case KindAuth:
		return 401
	case KindForbidden:
		return 403
	case KindNotFound:
		return 404
	case KindConflict:
		return 409
	case KindRateLimit:
		return 429
	case KindExternal:
		return 502
	case KindDatabase, KindInternal:
		return 500
	default:
		return 500
	}
}
