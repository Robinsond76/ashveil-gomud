//go:build !windows

package main

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/util"
)

func registerShutdownSignals(sigCh chan os.Signal) {
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM, syscall.SIGTSTP)
}

func startCopyoverSignalHandler() {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGUSR1)
	go func() {
		for range sigCh {
			mudlog.Info("SIGUSR1 received, initiating copyover")
			// This goroutine is off the game loop; hold the world lock as
			// the admin copyover command does, since the saves (plugin
			// OnSave included) read and write game state.
			util.LockMud()
			err := triggerCopyover()
			util.UnlockMud()
			if err != nil {
				mudlog.Error("copyover failed", "error", err)
			}
		}
	}()
}
