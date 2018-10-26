package systemd

import (
	"os"
	"strconv"
	"time"
)

// WatchdogActive returns whether a systemd watchdog is active
// and the interval, if any
func WatchdogActive(interval *time.Duration) (bool, error) {
	var err error
	var s, p string
	var u uint64
	var pid int

	s = os.Getenv("WATCHDOG_USEC")
	if s == "" {
		return false, nil
	}

	u, err = strconv.ParseUint(s, 10, 64)
	if err != nil {
		return false, err
	}

	p = os.Getenv("WATCHDOG_PID")
	if p != "" {
		pid, err = strconv.Atoi(p)
		if err != nil {
			return false, err
		}

		if pid != os.Getpid() {
			return false, nil
		}
	}

	if interval != nil {
		*interval = time.Duration(u) * time.Microsecond
	}

	return true, nil
}

// WatchdogCallback defines a callback that should be executed
// before sending a watchdog reply
type WatchdogCallback func() bool

// WatchdogTicker checks if a watchdog timer is expected
// and creates a ticker for it
func WatchdogTicker(done <-chan struct{}, cb WatchdogCallback) (bool, error) {
	var err error
	var interval time.Duration
	var active bool
	active, err = WatchdogActive(&interval)
	if !active || err != nil {
		return false, err
	}

	err = NotifyWatchdog()
	if err != nil {
		return false, err
	}

	go func(interval time.Duration, done <-chan struct{}) {
		var ticker = time.NewTicker(interval)

		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				if cb == nil || cb() {
					NotifyWatchdog()
				}
			}
		}
	}(interval, done)

	return true, nil
}
