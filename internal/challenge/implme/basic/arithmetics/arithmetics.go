package arithmetics

import (
	"sync"
	"sync/atomic"
	"time"
)

func SequentialSum(inputSize int) int {
	sum := 0
	for i := 1; i <= inputSize; i++ {
		sum += process(i)
	}
	return sum
}

// ParallelSum implement this method.
func ParallelSum(inputSize int) int {
	var sum atomic.Int64
	var wg = sync.WaitGroup{}
	for i := 1; i <= inputSize; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			sum.Add(int64(i * i))
		}()
	}
	wg.Wait()
	return int(sum.Load())
}

func process(num int) int {
	time.Sleep(time.Millisecond) // simulate processing time
	return num * num
}
