//go:build !windows

package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
)

func readSecret(input *os.File, _ io.Writer) ([]byte, error) {
	if info, err := input.Stat(); err == nil && info.Mode()&os.ModeCharDevice != 0 {
		return nil, fmt.Errorf("hidden terminal input is unsupported on this platform; pipe the secret through stdin")
	}
	reader := bufio.NewReader(io.LimitReader(input, 64*1024+3))
	data, err := reader.ReadBytes('\n')
	if err != nil && err != io.EOF {
		return nil, err
	}
	data = bytes.TrimSuffix(data, []byte("\n"))
	data = bytes.TrimSuffix(data, []byte("\r"))
	if len(data) == 0 || len(data) > 64*1024 {
		return nil, fmt.Errorf("secret must contain between 1 and 65536 bytes")
	}
	return data, nil
}
