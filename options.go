package saga

import (
	"time"
)

const (
	defaultMaxRetries    = 5
	defaultRetryWaitTime = 300 * time.Millisecond
)

type SagaOptions struct {
	MaxRetries    int
	RetryWaitTime time.Duration
}
