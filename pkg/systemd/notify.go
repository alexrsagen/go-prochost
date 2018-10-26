package systemd

import (
	"errors"
	"fmt"
	"net"
	"os"
	"strings"
	"syscall"
)

// ErrorNoEnv is returned when the NOTIFY_SOCKET environemnt variable is not set
var ErrorNoEnv = errors.New("No NOTIFY_SOCKET environment variable set")

// ErrorInvalidEnv is returned when the NOTIFY_SOCKET environment variable is invalid
var ErrorInvalidEnv = errors.New("Invalid NOTIFY_SOCKET environment variable")

// ErrorInvalidStatus is returned when a single-line status message contains newlines
var ErrorInvalidStatus = errors.New("Invalid status message")

// Notify replicates the behaviour of sd_notify(3)
// See https://manpages.debian.org/jessie/libsystemd-dev/sd_notify.3.en.html
func Notify(message string) error {
	var err error
	var addr *net.UnixAddr
	var conn *net.UnixConn

	var e = os.Getenv("NOTIFY_SOCKET")
	if e == "" {
		return ErrorNoEnv
	}

	if e[0] != '@' && e[0] != '/' || len(e) == 1 {
		err = ErrorInvalidEnv
		goto finish
	}

	addr, err = net.ResolveUnixAddr("unixgram", e)
	if err != nil {
		goto finish
	}

	conn, err = net.DialUnix(addr.Net, nil, addr)
	if err != nil {
		goto finish
	}

	_, err = conn.Write([]byte(message))
	if err != nil {
		goto finish
	}

finish:
	if conn != nil {
		conn.Close()
	}
	return err
}

// NotifyReady sends a ready ping to the system manager.
func NotifyReady() error {
	return Notify("READY=1")
}

// NotifyWatchdog sends a keep-alive ping to the system manager.
func NotifyWatchdog() error {
	return Notify("WATCHDOG=1")
}

// NotifyStatus passes a single-line status string back to the init system
// that describes the daemon state.
func NotifyStatus(status string) error {
	if strings.ContainsAny(status, "\n\r") {
		return ErrorInvalidStatus
	}

	return Notify(fmt.Sprintf("STATUS=%s", status))
}

// NotifyErrno sends an errno-style error code to the init system.
func NotifyErrno(errno syscall.Errno) error {
	return Notify(fmt.Sprintf("ERRNO=%d", errno))
}

// NotifyMainPid sends the main pid of the daemon, in case this process isn't it.
func NotifyMainPid(pid int) error {
	return Notify(fmt.Sprintf("MAINPID=%d", pid))
}
