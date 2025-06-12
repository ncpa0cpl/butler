package butler_test

import (
	"time"
)

func waitUntil(predicate func() bool) {
	start := time.Now()

	for !predicate() {
		if time.Since(start).Seconds() > 10 {
			panic("predicate not satisfied within the timeout")
		}

		time.Sleep(200 * time.Millisecond)
	}
}
