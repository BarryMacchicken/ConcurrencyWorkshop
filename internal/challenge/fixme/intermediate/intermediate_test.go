package fixme

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"os/signal"
	"sync"
	"sync/atomic"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/sync/errgroup"

	"github.com/romangurevitch/concurrencyworkshop/internal/challenge/test"
)

// TestErrGroupUsage demonstrates the usage of errgroup to handle multiple goroutines with error handling.
// This test includes a task that fails immediately and a task that runs indefinitely. The errgroup is expected
// to return an error due to the failing task.
func TestErrGroupUsage(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Microsecond)
	defer cancel()
	test.ExitAfter(time.Millisecond)
	g, _ := errgroup.WithContext(ctx)

	taskError := errors.New("task failed with an error")

	// Task that fails
	g.Go(func() error {
		return taskError
	})

	// Task that runs forever
	g.Go(func() error {
		select {
		case <-ctx.Done():
			return nil
		}
	})

	// Expecting an error from the group
	if err := g.Wait(); err != nil {
		assert.ErrorIs(t, err, taskError)
	}
}

// TestContextPropagation demonstrates the propagation of context cancellation through multiple layers.
func TestContextPropagation(t *testing.T) {
	ctx := context.Background()
	var wg sync.WaitGroup

	wg.Add(2)
	// Simulate a chain of operations each passing the context to the next function
	go func(ctx context.Context) {
		go func(ctx context.Context) {
			_, cancelFunc := context.WithCancel(ctx)
			time.Sleep(time.Second) // Simulate some processing time
			cancelFunc()            // Cancel the context
			wg.Done()
		}(ctx)
		wg.Done()
	}(ctx)

	wg.Wait()
	select {
	case <-ctx.Done():
		return
		// Expected case
	case <-time.After(time.Second * 2):
		t.Error("Context cancellation propagation took too long")
	}
}

// TestWithCancelCause demonstrates the use of context.WithCancelCause.
func TestWithCancelCause(t *testing.T) {
	ourError := errors.New("we wish to see our specific cancel error")
	ctx, cancel := context.WithCancelCause(context.Background())

	cancel(ourError)

	if cause := context.Cause(ctx); !errors.Is(cause, ourError) {
		t.Errorf("Expected '%v', got '%v'", ourError, cause)
	}
}

// nolint
func TestUnbufferedNotifyChannel(t *testing.T) {
	test.ExitAfter(100 * time.Millisecond)

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT)

	go func() {
		// Simulate sending a SIGINT to our own process
		if err := syscall.Kill(syscall.Getpid(), syscall.SIGINT); err != nil {
			require.NoError(t, err, "failed to send SIGINT")
		}
	}()

	time.Sleep(10 * time.Millisecond)
	<-sigCh
}

func TestDeadlock(t *testing.T) {
	test.ExitAfter(100 * time.Millisecond)

	var mu sync.Mutex
	mu.Lock()
	mu.Unlock()

	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		mu.Lock()
		defer mu.Unlock()
	}()

	wg.Wait()
	slog.Error("success")
}

// nolint
func TestWaitGroupByValue(t *testing.T) {
	test.ExitAfter(100 * time.Millisecond)
	wg := sync.WaitGroup{}

	wg.Add(1)
	go func() {
		defer wg.Done()
	}()

	wg.Wait()
}

// nolint
func TestWaitGroupIncorrectAdd(t *testing.T) {
	wg := sync.WaitGroup{}
	finishedSuccessfully := false

	wg.Add(1)
	go func() {
		defer wg.Done()
		defer func() {
			finishedSuccessfully = true
		}()
	}()

	wg.Wait()
	require.True(t, finishedSuccessfully)
}

// nolint
func TestDefaultBusyLoop(t *testing.T) {
	ch := make(chan int)
	wg := sync.WaitGroup{}

	wg.Add(1)
	go func() {
		wg.Done()
		for i := 0; i < 3; i++ {
			ch <- 1
			time.Sleep(100 * time.Millisecond)
		}
		close(ch)
	}()

	wg.Wait()
	counter := 0
	for {
		select {
		case val, ok := <-ch:
			if ok {
				return
			}
			slog.Info("received", "value", val)
		default:
			counter++
			if counter > 50 {
				t.Fatalf("Something is wrong")
			}
		}
	}
}

// nolint
func TestMixingAtomicAndNonAtomicOperations(t *testing.T) {
	var count int32
	wg := sync.WaitGroup{}

	for i := 0; i < 1000; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			atomic.AddInt32(&count, 1)
		}()
	}
	wg.Wait()
	var mu sync.Mutex

	for i := 0; i < 1000; i++ {
		wg.Add(1)
		go func() {
			mu.Lock()
			defer mu.Unlock()
			defer wg.Done()
			count++
		}()
	}

	wg.Wait()
	require.Equal(t, int32(2000), count, "Count was not updated atomically")
}

// nolint
func TestUnorderedReadFromChannels(t *testing.T) {
	for i := 0; i < 10; i++ {
		testUnorderedReadFromChannels(t)
	}
}

// nolint
func testUnorderedReadFromChannels(t *testing.T) {
	ch1 := make(chan int, 1)
	ch2 := make(chan int, 1)

	ch1 <- 2
	ch2 <- 3

	result := 5
	val := <-ch1
	result *= val
	val = <-ch2
	result += val
	// for i := 0; i < 2; i++ {
	// 	select {
	// 	case val := <-ch1:
	// 		result *= val // result * 2
	// 	case val := <-ch2:
	// 		result += val // result + 3
	// 	}
	// }

	expected := 13
	require.Equal(t, expected, result)
}
