//go:build linux

package main

import (
	"fmt"
	"os"
)

func setConsoleTitle(title string) {
	if os.Getenv("TERM") == "" || os.Getenv("TERM") == "dumb" {
		return
	}
	_, _ = fmt.Fprintf(os.Stderr, "\033]0;%s\007", title)
}
