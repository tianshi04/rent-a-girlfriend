package vo

// ContextKey represents a custom context key type.
type ContextKey string

// CorrelationIDKey is the context key for the trace correlation ID.
const CorrelationIDKey ContextKey = "correlation_id"
