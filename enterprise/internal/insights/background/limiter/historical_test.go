package limiter_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/sourcegraph/sourcegraph/enterprise/internal/insights/background/limiter"
)

func TestGetHistoricLimit(t *testing.T) {
	t.Run("Is singleton", func(t *testing.T) {
		limiter1 := limiter.HistoricalWorkRate()
		limiter2 := limiter.HistoricalWorkRate()

		if limiter1 != limiter2 {
			t.Error("both limiters should be the same instance")
		}
	})

	t.Run("all instances see changes", func(t *testing.T) {
		limiter1 := limiter.HistoricalWorkRate()
		limiter2 := limiter.HistoricalWorkRate()
		limiter1.SetLimit(99)
		assert.Equal(t, limiter1.Limit(), limiter2.Limit(), 99)
	})

}
