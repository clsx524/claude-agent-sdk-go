package claude

import "context"

// Transport is the abstract interface for Claude communication.
//
// WARNING: This internal API is exposed for custom transport implementations
// (e.g., remote Claude Code connections). The Claude Code team may change or
// remove this interface in any future release. Custom implementations must be
// updated to match interface changes.
//
// This is a low-level transport interface that handles raw I/O with the Claude
// process or service. The Query class builds on top of this to implement the
// control protocol and message routing.
type Transport interface {
	// Connect initializes the transport and prepares for communication.
	// For subprocess transports, this starts the process.
	// For network transports, this establishes the connection.
	Connect(ctx context.Context) error

	// Write sends raw data to the transport.
	// Data is typically JSON + newline.
	Write(ctx context.Context, data string) error

	// ReadMessages returns a channel that yields parsed JSON messages from the transport.
	// The channel will be closed when the transport is closed or encounters an error.
	ReadMessages(ctx context.Context) (<-chan map[string]interface{}, <-chan error)

	// Close terminates the transport connection and cleans up resources.
	Close() error

	// IsReady checks if the transport is ready for communication.
	//
	// Returns true after successful Connect() and before Close().
	// Primarily used for testing transport state transitions.
	//
	// Note: Most applications don't need to call this directly - the SDK
	// handles connection management automatically.
	IsReady() bool

	// EndInput signals the end of the input stream (close stdin for process transports).
	EndInput() error
}
