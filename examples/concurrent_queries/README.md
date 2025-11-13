# Concurrent Queries Example

This example demonstrates various Go concurrency patterns using the Claude Agent SDK.

## Overview

The example showcases 7 different concurrency patterns that are commonly used with the SDK:

1. **Parallel Queries** - Execute multiple independent queries concurrently
2. **Worker Pool Pattern** - Process queries using a fixed number of worker goroutines
3. **Fan-Out/Fan-In Pattern** - Distribute work and aggregate results
4. **Context Cancellation** - Handle timeouts and cancellation
5. **Buffered Channels** - Use custom buffer sizes for better performance
6. **Concurrent Message Processing** - Process streaming messages in parallel
7. **Rate Limiting** - Control concurrent query execution with semaphores

## Running the Example

```bash
cd examples/concurrent_queries
go run main.go
```

## Pattern Details

### 1. Parallel Queries

Executes multiple queries simultaneously using goroutines and sync.WaitGroup:

```go
var wg sync.WaitGroup
for _, query := range queries {
    wg.Add(1)
    go func(q string) {
        defer wg.Done()
        msgCh, errCh, err := claude.Query(ctx, q, nil, nil)
        // Process response...
    }(query)
}
wg.Wait()
```

**Use case**: When you have multiple independent queries and want to minimize total execution time.

### 2. Worker Pool Pattern

Creates a fixed number of workers that process queries from a job queue:

```go
jobs := make(chan string, len(queries))
for w := 1; w <= numWorkers; w++ {
    go worker(w, jobs, results)
}
```

**Use case**: Limiting concurrent API calls while processing a large number of queries.

### 3. Fan-Out/Fan-In Pattern

Distributes work across multiple goroutines (fan-out) and merges results (fan-in):

```go
channels := make([]<-chan string, len(queries))
// Start goroutines...
merged := merge(channels...)
```

**Use case**: Breaking down a large problem into parallel subtasks and combining results.

### 4. Context Cancellation

Demonstrates proper context usage for timeouts and cancellation:

```go
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()
```

**Use case**: Enforcing execution time limits or allowing user cancellation.

### 5. Buffered Channels

Uses custom buffer sizes for high-throughput scenarios:

```go
bufferSize := 500
options := &claude.ClaudeAgentOptions{
    MessageChannelBufferSize: &bufferSize,
}
```

**Use case**: Handling queries that produce many messages without blocking.

### 6. Concurrent Message Processing

Processes streaming messages using multiple processor goroutines:

```go
processingCh := make(chan claude.Message, 10)
for i := 0; i < numProcessors; i++ {
    go processor(i, processingCh)
}
```

**Use case**: Parallelizing expensive message processing operations.

### 7. Rate Limiting

Controls concurrency using a semaphore pattern:

```go
semaphore := make(chan struct{}, maxConcurrent)
semaphore <- struct{}{}        // Acquire
defer func() { <-semaphore }() // Release
```

**Use case**: Limiting concurrent API calls to avoid rate limits or resource exhaustion.

## Key Takeaways

- **Always use context.Context** for cancellation and timeouts
- **Close channels properly** to avoid goroutine leaks
- **Use sync.WaitGroup** to coordinate goroutine completion
- **Handle errors** from both the initial query and error channels
- **Choose appropriate patterns** based on your use case
- **Consider resource limits** when running many concurrent queries

## Performance Tips

1. **Reuse clients** for multiple queries instead of creating new ones
2. **Set appropriate buffer sizes** based on expected message volume
3. **Limit concurrency** to avoid overwhelming the system
4. **Use worker pools** for predictable resource usage
5. **Profile your application** to identify bottlenecks

## Related Examples

- `../streaming/` - Streaming patterns and message handling
- `../max_budget_usd/` - Budget control across multiple queries
- `../tool_permission_callback/` - Permission handling in concurrent scenarios
