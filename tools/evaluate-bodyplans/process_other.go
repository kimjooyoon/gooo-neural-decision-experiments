//go:build !darwin && !linux

package main

import (
	"errors"
	"os/exec"
)

func containCommand(*exec.Cmd) error {
	return errors.New("bounded study process groups require macOS or Linux")
}
