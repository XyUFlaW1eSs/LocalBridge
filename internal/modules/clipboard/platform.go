package clipboard

import (
	"context"
	"errors"
	"time"
)

type Platform interface {
	ReadText() (string, error)
	WriteText(string) error
	Watch(context.Context, time.Duration, func(string), func(error)) error
}

var ErrUnsupportedPlatform = errors.New("clipboard integration is not supported on this platform")

func newPlatform() Platform { return platformClipboard{} }
