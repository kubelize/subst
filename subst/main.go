package main

import (
	"runtime/debug"

	"github.com/kubelize/subst/subst/cmd"
)

func main() {
	debug.SetGCPercent(100)
	cmd.Execute()
}
