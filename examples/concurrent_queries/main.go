package main

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	claude "github.com/clsx524/claude-agent-sdk-go"
)

// Example 1: Multiple parallel queries using goroutines
func parallelQueries() {
	fmt.Println("\n=== Example 1: Parallel Queries ===")

	ctx := context.Background()
	queries := []string{
		"What is 2+2?",
		"What is the capital of France?",
		"What is the meaning of life?",
	}

	var wg sync.WaitGroup
	results := make(chan string, len(queries))

	for i, query := range queries {
		wg.Add(1)
		go func(id int, q string) {
			defer wg.Done()

			msgCh, errCh, err := claude.Query(ctx, q, nil, nil)
			if err != nil {
				log.Printf("Query %d failed: %v", id, err)
				return
			}

			var response string
			for msg := range msgCh {
				if assistantMsg, ok := msg.(*claude.AssistantMessage); ok {
					for _, block := range assistantMsg.Content {
						if textBlock, ok := block.(claude.TextBlock); ok {
							response += textBlock.Text
						}
					}
				}
			}

			if err := <-errCh; err != nil {
				log.Printf("Query %d error: %v", id, err)
				return
			}

			results <- fmt.Sprintf("Query %d (%s): %s", id, q, response)
		}(i, query)
	}

	wg.Wait()
	close(results)

	for result := range results {
		fmt.Println(result)
	}
}

// Example 2: Worker pool pattern for processing multiple queries
func workerPoolPattern() {
	fmt.Println("\n=== Example 2: Worker Pool Pattern ===")

	ctx := context.Background()
	numWorkers := 3
	queries := []string{
		"Count to 3",
		"List 3 colors",
		"Name 3 animals",
		"List 3 countries",
		"Name 3 fruits",
		"List 3 programming languages",
	}

	// Create job queue
	jobs := make(chan string, len(queries))
	results := make(chan string, len(queries))

	// Start worker pool
	var wg sync.WaitGroup
	for w := 1; w <= numWorkers; w++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for query := range jobs {
				fmt.Printf("Worker %d processing: %s\n", workerID, query)

				msgCh, errCh, err := claude.Query(ctx, query, nil, nil)
				if err != nil {
					log.Printf("Worker %d error: %v", workerID, err)
					continue
				}

				var response string
				for msg := range msgCh {
					if assistantMsg, ok := msg.(*claude.AssistantMessage); ok {
						for _, block := range assistantMsg.Content {
							if textBlock, ok := block.(claude.TextBlock); ok {
								response += textBlock.Text
							}
						}
					}
				}

				if err := <-errCh; err != nil {
					log.Printf("Worker %d error: %v", workerID, err)
					continue
				}

				results <- fmt.Sprintf("Worker %d result: %s", workerID, response)
			}
		}(w)
	}

	// Send jobs
	for _, query := range queries {
		jobs <- query
	}
	close(jobs)

	// Wait for workers to finish
	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect results
	for result := range results {
		fmt.Println(result)
	}
}

// Example 3: Fan-out/Fan-in pattern
func fanOutFanIn() {
	fmt.Println("\n=== Example 3: Fan-Out/Fan-In Pattern ===")

	ctx := context.Background()

	// Fan-out: split work across multiple queries
	queries := []string{
		"What is 10 + 5?",
		"What is 20 + 3?",
		"What is 7 + 8?",
	}

	channels := make([]<-chan string, len(queries))

	for i, query := range queries {
		ch := make(chan string, 1)
		channels[i] = ch

		go func(q string, resultCh chan<- string) {
			defer close(resultCh)

			msgCh, errCh, err := claude.Query(ctx, q, nil, nil)
			if err != nil {
				resultCh <- fmt.Sprintf("Error: %v", err)
				return
			}

			var response string
			for msg := range msgCh {
				if assistantMsg, ok := msg.(*claude.AssistantMessage); ok {
					for _, block := range assistantMsg.Content {
						if textBlock, ok := block.(claude.TextBlock); ok {
							response += textBlock.Text
						}
					}
				}
			}

			if err := <-errCh; err != nil {
				resultCh <- fmt.Sprintf("Error: %v", err)
				return
			}

			resultCh <- response
		}(query, ch)
	}

	// Fan-in: merge results from multiple channels
	merged := merge(channels...)

	fmt.Println("Results:")
	for result := range merged {
		fmt.Printf("- %s\n", result)
	}
}

// merge combines multiple channels into a single channel
func merge(channels ...<-chan string) <-chan string {
	var wg sync.WaitGroup
	out := make(chan string)

	// Start an output goroutine for each input channel
	output := func(c <-chan string) {
		defer wg.Done()
		for n := range c {
			out <- n
		}
	}

	wg.Add(len(channels))
	for _, c := range channels {
		go output(c)
	}

	// Close the output channel when all outputs are done
	go func() {
		wg.Wait()
		close(out)
	}()

	return out
}

// Example 4: Context cancellation and timeout
func contextCancellation() {
	fmt.Println("\n=== Example 4: Context Cancellation ===")

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Start a long-running query
	done := make(chan bool)

	go func() {
		msgCh, errCh, err := claude.Query(ctx, "Write a very long essay about the history of computing", nil, nil)
		if err != nil {
			log.Printf("Query failed: %v", err)
			done <- true
			return
		}

		for msg := range msgCh {
			if assistantMsg, ok := msg.(*claude.AssistantMessage); ok {
				for _, block := range assistantMsg.Content {
					if textBlock, ok := block.(claude.TextBlock); ok {
						fmt.Printf("Received: %s...\n", textBlock.Text[:min(50, len(textBlock.Text))])
					}
				}
			}
		}

		if err := <-errCh; err != nil {
			fmt.Printf("Query ended with: %v\n", err)
		}

		done <- true
	}()

	select {
	case <-done:
		fmt.Println("Query completed")
	case <-ctx.Done():
		fmt.Println("Query cancelled due to timeout")
	}
}

// Example 5: Buffered channels for backpressure
func bufferedChannels() {
	fmt.Println("\n=== Example 5: Buffered Channels with Custom Buffer Size ===")

	ctx := context.Background()

	// Use custom buffer size for better performance with high message volume
	bufferSize := 500
	options := &claude.ClaudeAgentOptions{
		MessageChannelBufferSize: &bufferSize,
	}

	query := "List 100 random numbers"
	msgCh, errCh, err := claude.Query(ctx, query, options, nil)
	if err != nil {
		log.Fatal(err)
	}

	messageCount := 0
	for msg := range msgCh {
		messageCount++
		if assistantMsg, ok := msg.(*claude.AssistantMessage); ok {
			for _, block := range assistantMsg.Content {
				if textBlock, ok := block.(claude.TextBlock); ok {
					fmt.Printf("Message %d: %s\n", messageCount, textBlock.Text[:min(100, len(textBlock.Text))])
				}
			}
		}
	}

	if err := <-errCh; err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Total messages received: %d\n", messageCount)
}

// Example 6: Streaming with concurrent message processing
func concurrentMessageProcessing() {
	fmt.Println("\n=== Example 6: Concurrent Message Processing ===")

	ctx := context.Background()
	client := claude.NewClaudeSDKClient(nil)

	if err := client.Connect(ctx); err != nil {
		log.Fatal(err)
	}
	defer client.Disconnect()

	// Send query
	msgCh, errCh := client.Query(ctx, "Explain Go concurrency patterns")

	// Process messages concurrently
	var wg sync.WaitGroup
	processingCh := make(chan claude.Message, 10)

	// Start message processors
	numProcessors := 3
	for i := 0; i < numProcessors; i++ {
		wg.Add(1)
		go func(processorID int) {
			defer wg.Done()
			for msg := range processingCh {
				// Simulate processing
				if assistantMsg, ok := msg.(*claude.AssistantMessage); ok {
					for _, block := range assistantMsg.Content {
						if textBlock, ok := block.(claude.TextBlock); ok {
							fmt.Printf("Processor %d: %s...\n", processorID, textBlock.Text[:min(50, len(textBlock.Text))])
						}
					}
				}
			}
		}(i)
	}

	// Send messages to processors
	go func() {
		for msg := range msgCh {
			processingCh <- msg
		}
		close(processingCh)
	}()

	wg.Wait()

	if err := <-errCh; err != nil {
		log.Fatal(err)
	}
}

// Example 7: Rate limiting with semaphore pattern
func rateLimiting() {
	fmt.Println("\n=== Example 7: Rate Limiting ===")

	ctx := context.Background()

	// Limit to 2 concurrent queries
	maxConcurrent := 2
	semaphore := make(chan struct{}, maxConcurrent)

	queries := []string{
		"What is Go?",
		"What is Python?",
		"What is JavaScript?",
		"What is Rust?",
		"What is Java?",
	}

	var wg sync.WaitGroup

	for i, query := range queries {
		wg.Add(1)

		go func(id int, q string) {
			defer wg.Done()

			// Acquire semaphore
			semaphore <- struct{}{}
			defer func() { <-semaphore }() // Release semaphore

			fmt.Printf("Starting query %d: %s\n", id, q)

			msgCh, errCh, err := claude.Query(ctx, q, nil, nil)
			if err != nil {
				log.Printf("Query %d failed: %v", id, err)
				return
			}

			for msg := range msgCh {
				if assistantMsg, ok := msg.(*claude.AssistantMessage); ok {
					for _, block := range assistantMsg.Content {
						if textBlock, ok := block.(claude.TextBlock); ok {
							fmt.Printf("Query %d response: %s\n", id, textBlock.Text[:min(80, len(textBlock.Text))])
						}
					}
				}
			}

			if err := <-errCh; err != nil {
				log.Printf("Query %d error: %v", id, err)
			}

			fmt.Printf("Completed query %d\n", id)
		}(i, query)
	}

	wg.Wait()
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func main() {
	fmt.Println("Claude Agent SDK - Concurrency Patterns Showcase")
	fmt.Println("================================================")

	// Run examples
	parallelQueries()
	workerPoolPattern()
	fanOutFanIn()
	contextCancellation()
	bufferedChannels()
	concurrentMessageProcessing()
	rateLimiting()

	fmt.Println("\n=== All examples completed ===")
}
