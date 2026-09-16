//go:build !windows

package credentials

import "os"

func replaceFile(from, to string) error { return os.Rename(from, to) }
