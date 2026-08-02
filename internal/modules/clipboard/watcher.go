package clipboard

import (
	"context"
	"time"
)

// Watcher adapts the platform clipboard change stream to the module callback.
// Keeping it separate makes the Windows implementation replaceable by native
// adapters on macOS, Linux and future clients.
type Watcher struct {
	platform Platform
	interval time.Duration
	onChange func(string)
	onError  func(error)
}

func (w Watcher) Run(ctx context.Context) error {
	return w.platform.Watch(ctx, w.interval, w.onChange, w.onError)
}
