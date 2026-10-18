//go:build windows

package main

import "os/exec"

func detach(*exec.Cmd) {}
