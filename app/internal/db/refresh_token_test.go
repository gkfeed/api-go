package db

import (
	"errors"
	"sync"
	"testing"
	"time"

	"gkfeed/api/internal/models"
)

func TestConsumeRefreshTokenIsAtomic(t *testing.T) {
	useTestDatabase(t)

	const tokenID = "refresh-token"
	if err := StoreRefreshToken(models.RefreshToken{
		ID:        tokenID,
		UserID:    1,
		ExpiresAt: time.Now().Add(time.Hour),
	}); err != nil {
		t.Fatalf("StoreRefreshToken() returned an error: %v", err)
	}

	start := make(chan struct{})
	results := make(chan error, 2)
	var waitGroup sync.WaitGroup
	waitGroup.Add(2)
	for range 2 {
		go func() {
			defer waitGroup.Done()
			<-start
			_, err := ConsumeRefreshToken(tokenID)
			results <- err
		}()
	}
	close(start)
	waitGroup.Wait()
	close(results)

	var successes, notFound int
	for err := range results {
		switch {
		case err == nil:
			successes++
		case errors.Is(err, ErrRefreshTokenNotFound):
			notFound++
		default:
			t.Fatalf("ConsumeRefreshToken() returned unexpected error: %v", err)
		}
	}
	if successes != 1 || notFound != 1 {
		t.Fatalf("ConsumeRefreshToken() results = %d successes, %d not found; want 1 each", successes, notFound)
	}
}
