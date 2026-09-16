//go:build windows

package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"

	"golang.org/x/sys/windows"
)

func readSecret(input *os.File, prompt io.Writer) ([]byte, error) {
	var mode uint32
	handle := windows.Handle(input.Fd())
	isConsole := windows.GetConsoleMode(handle, &mode) == nil
	if isConsole {
		if _, err := fmt.Fprint(prompt, "Secret: "); err != nil {
			return nil, err
		}
		if err := windows.SetConsoleMode(handle, mode&^windows.ENABLE_ECHO_INPUT); err != nil {
			return nil, fmt.Errorf("disable terminal echo: %w", err)
		}
		defer windows.SetConsoleMode(handle, mode) //nolint:errcheck
		defer fmt.Fprintln(prompt)                 //nolint:errcheck
	}
	return readSecretBytes(input)
}

func readSecretBytes(input io.Reader) ([]byte, error) {
	reader := bufio.NewReader(io.LimitReader(input, 64*1024+3))
	data, err := reader.ReadBytes('\n')
	if err != nil && err != io.EOF {
		return nil, err
	}
	data = bytes.TrimSuffix(data, []byte("\n"))
	data = bytes.TrimSuffix(data, []byte("\r"))
	if len(data) == 0 {
		return nil, fmt.Errorf("secret must not be empty")
	}
	if len(data) > 64*1024 {
		return nil, fmt.Errorf("secret exceeds 65536 bytes")
	}
	return data, nil
}
