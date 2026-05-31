package kernel

import "context"

type ctxKey int

const (
	keyTraceID ctxKey = iota + 1
	keyTenantID
	keyMemberID
	keyPlatformUserID
)

// WithTraceID attaches a request-scoped trace id (set by middleware/request_id).
func WithTraceID(ctx context.Context, traceID string) context.Context {
	return context.WithValue(ctx, keyTraceID, traceID)
}

// TraceIDFromContext returns the trace id or "" if not set.
func TraceIDFromContext(ctx context.Context) string {
	if v, ok := ctx.Value(keyTraceID).(string); ok {
		return v
	}
	return ""
}

// WithTenantID attaches the active tenant id (set by tenant-loader middleware in M2).
func WithTenantID(ctx context.Context, tenantID ID) context.Context {
	return context.WithValue(ctx, keyTenantID, tenantID)
}

// TenantIDFromContext returns (tenantID, true) if set, else ("", false).
func TenantIDFromContext(ctx context.Context) (ID, bool) {
	v, ok := ctx.Value(keyTenantID).(ID)
	return v, ok
}

// WithMemberID attaches the calling member id (set by IAM middleware in M2).
func WithMemberID(ctx context.Context, memberID ID) context.Context {
	return context.WithValue(ctx, keyMemberID, memberID)
}

// MemberIDFromContext returns (memberID, true) if set, else ("", false).
func MemberIDFromContext(ctx context.Context) (ID, bool) {
	v, ok := ctx.Value(keyMemberID).(ID)
	return v, ok
}

// WithPlatformUserID attaches the global platform user id from the verified token.
func WithPlatformUserID(ctx context.Context, platformUserID ID) context.Context {
	return context.WithValue(ctx, keyPlatformUserID, platformUserID)
}

// PlatformUserIDFromContext returns (platformUserID, true) if set, else ("", false).
func PlatformUserIDFromContext(ctx context.Context) (ID, bool) {
	v, ok := ctx.Value(keyPlatformUserID).(ID)
	return v, ok
}
