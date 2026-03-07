package zellijcommand

import (
	"syscall"
	"time"
)

type Option func(*ZellijCommand)

func WithCloseSignal(signal syscall.Signal) Option {
	return func(zcmd *ZellijCommand) {
		zcmd.closeSignal = signal
	}
}

func WithCloseTimeout(timeout time.Duration) Option {
	return func(zcmd *ZellijCommand) {
		zcmd.closeTimeout = timeout
	}
}
