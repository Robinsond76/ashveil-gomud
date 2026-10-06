package mudlog

import (
	"sync"
	"testing"
)

// Logging from one goroutine while another sets up the logger must not race
// (the server's workers log while tests and tools call SetupLogger).
func TestLoggingNeverRacesSetupLogger(t *testing.T) {
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 50; i++ {
			Debug("racing", "i", i)
		}
	}()
	for i := 0; i < 50; i++ {
		SetupLogger(nil, "LOW", "", false)
	}
	wg.Wait()
}
