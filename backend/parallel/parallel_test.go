package parallel

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestForEachHonorsWorkerLimit(t *testing.T) {
	indexes := make([]int, 12)
	for index := range indexes {
		indexes[index] = index
	}

	var mu sync.Mutex
	active := 0
	maximum := 0
	ForEach(context.Background(), indexes, DefaultWorkers, func(int) bool {
		mu.Lock()
		active++
		if active > maximum {
			maximum = active
		}
		mu.Unlock()

		time.Sleep(5 * time.Millisecond)

		mu.Lock()
		active--
		mu.Unlock()
		return true
	})

	if maximum != DefaultWorkers {
		t.Fatalf("maximum concurrent work = %d, want %d", maximum, DefaultWorkers)
	}
}
