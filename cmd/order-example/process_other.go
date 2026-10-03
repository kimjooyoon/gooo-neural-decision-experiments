//go:build !linux && !darwin

package main

import "os/exec"

// Other platforms retain exec.CommandContext's direct-child cancellation.
func bindCommandGroup(c *exec.Cmd) {}
