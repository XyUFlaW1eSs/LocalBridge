//go:build !windows

package clipboard

import (
	"context"
	"time"
)

type platformClipboard struct{}

func (platformClipboard) ReadText() (string, error) { return "", ErrUnsupportedPlatform }
func (platformClipboard) WriteText(string) error    { return ErrUnsupportedPlatform }
func (platformClipboard) Watch(ctx context.Context, _ time.Duration, _ func(string)) error {
	<-ctx.Done()
	return nil
}
