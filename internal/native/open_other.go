//go:build !windows

package native

import (
	"os/exec"
	"runtime"
)

func OpenURL(url string) error {
	if runtime.GOOS == "darwin" {
		return exec.Command("open", url).Start()
	}
	return exec.Command("xdg-open", url).Start()
}
