package facts

import (
	"context"
	"time"
)

func boolPtr(value bool) *bool {
	return &value
}

func pollSBUntilConverged(
	ctx context.Context,
	readSB dbCfgReader,
	nbCfg int64,
	pollInterval time.Duration,
	deadline time.Duration,
) (value *int64, converged *bool, err error) {
	var sbCfg int64
	pollCtx, cancel := context.WithTimeout(ctx, deadline)
	defer cancel()

	for {
		var next int64
		next, err = readSB(pollCtx)
		if err != nil {
			if ctx.Err() != nil {
				return value, nil, ctx.Err()
			}
			if pollCtx.Err() == context.DeadlineExceeded {
				if value == nil {
					return nil, nil, pollCtx.Err()
				}
				return value, boolPtr(false), nil
			}
			return value, nil, err
		}
		sbCfg = next
		value = &sbCfg

		if sbCfg >= nbCfg {
			return value, boolPtr(true), nil
		}

		interval := time.NewTimer(pollInterval)
		select {
		case <-ctx.Done():
			interval.Stop()
			return value, nil, ctx.Err()

		case <-pollCtx.Done():
			interval.Stop()
			if ctx.Err() != nil {
				return value, nil, ctx.Err()
			}
			return value, boolPtr(false), nil

		case <-interval.C:
			// Check again
		}
	}
}
