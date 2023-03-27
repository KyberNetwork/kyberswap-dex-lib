package tracing

import "context"

// ISpan represents a chunk of computation time. Spans have names, durations,
// timestamps and other metadata. A ITracer is used to create hierarchies of
// spans in a request, buffer and submit them to the server.
type ISpan interface {
	// SetTag sets a key/value pair as metadata on the span.
	SetTag(key string, value interface{})

	// Finish finishes the current span with the given options. Finish calls should be idempotent.
	Finish()
}

type ITracer interface {
	Trace(ctx context.Context, operationName string) (ISpan, context.Context)
}
