package main

import (
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"

	"github.com/svent/go-nbreader"

	"../../pkg/systemd"
	"github.com/tomclegg/nbtee"
)

func main() {
	var err error
	var listen, file string
	var bufsize int
	var args = make([]string, 1)
	var info os.FileInfo
	var cmd *exec.Cmd
	var stdout io.ReadCloser
	var stderr io.ReadCloser
	var stdin io.WriteCloser

	// Parse arguments
	flag.StringVar(&listen, "l", "", "Listen path/address")
	flag.StringVar(&file, "f", "", "File to execute")
	flag.IntVar(&bufsize, "b", 0, "Amount of lines to store in buffer")
	flag.Parse()

	// Ensure buffer size is valid
	if bufsize < 0 {
		os.Stderr.WriteString("prochost: Invalid buffer size\n")
		systemd.NotifyErrno(syscall.EINVAL)
		os.Exit(1)
	}

	// Create stdio writers
	var bufWriter = &LineBufferedWriter{buf: make([]string, bufsize)}
	var outWriter = nbtee.NewWriter(1)
	outWriter.Add(os.Stdout)
	outWriter.Add(bufWriter)
	var errWriter = nbtee.NewWriter(1)
	errWriter.Add(os.Stderr)
	errWriter.Add(bufWriter)
	var inReader = &AsyncMultiReader{readers: []io.Reader{os.Stdin}}

	// Get child process arguments
	for i := 0; i < len(os.Args); i++ {
		if os.Args[i] == "--" && i+1 < len(os.Args) {
			args = os.Args[i:]
			break
		}
	}

	// Ensure file argument is set
	if file == "" || file[0] != '/' || len(file) == 1 {
		os.Stderr.WriteString("prochost: Invalid buffer size\n")
		systemd.NotifyErrno(syscall.EINVAL)
		os.Exit(1)
	}

	// Ensure file directory exists
	info, err = os.Stat(filepath.Dir(file))
	if os.IsNotExist(err) {
		os.Stderr.WriteString("prochost: Executable parent directory does not exist\n")
		systemd.NotifyErrno(syscall.ENOENT)
		os.Exit(1)
	}
	if !info.Mode().IsDir() {
		os.Stderr.WriteString("prochost: Executable parent directory is not a directory\n")
		systemd.NotifyErrno(syscall.ENOTDIR)
		os.Exit(1)
	}

	// Ensure file exists
	_, err = os.Stat(file)
	if os.IsNotExist(err) {
		os.Stderr.WriteString("prochost: Executable does not exist\n")
		systemd.NotifyErrno(syscall.ENOENT)
		os.Exit(1)
	}

	// Set file argument
	args[0] = file

	// Validate listen argument if it is set
	if listen != "" {
		if len(listen) == 1 {
			os.Stderr.WriteString("prochost: Listen path/address is invalid\n")
			systemd.NotifyErrno(syscall.EINVAL)
			os.Exit(1)
		}

		if listen[0] == '/' { // Unix socket
			// Ensure socket directory exists
			info, err = os.Stat(filepath.Dir(listen))
			if os.IsNotExist(err) {
				os.Stderr.WriteString("prochost: Socket directory does not exist\n")
				systemd.NotifyErrno(syscall.ENOENT)
				os.Exit(1)
			}
			if !info.Mode().IsDir() {
				os.Stderr.WriteString("prochost: Socket directory is not a directory\n")
				systemd.NotifyErrno(syscall.ENOTDIR)
				os.Exit(1)
			}

			// Ensure socket does not exist
			_, err = os.Stat(listen)
			if os.IsExist(err) {
				os.Stderr.WriteString("prochost: Socket already exists\n")
				systemd.NotifyErrno(syscall.EEXIST)
				os.Exit(1)
			}
		}

		go func(
			address string,
			bufWriter *LineBufferedWriter,
			outWriter, errWriter *nbtee.Writer,
			inReader *AsyncMultiReader,
		) {
			var err error
			var listener net.Listener
			var conn net.Conn

			if address[0] == '/' {
				listener, err = net.Listen("unix", address)
			} else {
				listener, err = net.Listen("tcp", address)
			}

			if err != nil {
				os.Stderr.WriteString(fmt.Sprintf("prochost: %s\n", err.Error()))
				systemd.NotifyErrno(syscall.EPERM)
				os.Exit(1)
			}

			for {
				conn, err = listener.Accept()
				if err != nil {
					continue
				}

				var lines = bufWriter.ReadLines()
				for _, line := range lines {
					if line != "" {
						conn.Write([]byte(line))
					}
				}

				outWriter.Add(conn)
				errWriter.Add(conn)
				inReader.AddReaders(nbreader.NewNBReader(conn, 1<<15))
			}
		}(listen, bufWriter, outWriter, errWriter, inReader)
	}

	// Create command structure
	cmd = exec.Command(file)
	cmd.Args = args

	stdout, err = cmd.StdoutPipe()
	stderr, err = cmd.StderrPipe()
	stdin, err = cmd.StdinPipe()

	// Forward stdout/stderr from process to prochost
	go func(w io.Writer, stdout io.ReadCloser) {
		io.Copy(w, stdout)
	}(outWriter, stdout)
	go func(w io.Writer, stderr io.ReadCloser) {
		io.Copy(w, stderr)
	}(errWriter, stderr)

	// Forward stdin from prochost to process
	go func(r io.Reader, stdin io.WriteCloser) {
		for {
			io.Copy(stdin, r)
		}
	}(inReader, stdin)

	// Start process
	err = cmd.Start()
	if err != nil {
		os.Stderr.WriteString("prochost: Failed to start executable\n")
		systemd.NotifyErrno(syscall.ENOEXEC)
		os.Exit(1)
	}

	// Notify systemd that process is started
	systemd.NotifyReady()

	// Start watchdog ticker
	var done = make(chan struct{})
	systemd.WatchdogTicker(done, nil)

	// Wait for process to exit
	cmd.Wait()

	// Clean up
	stdout.Close()
	stderr.Close()
	stdin.Close()

	// Stop watchdog ticker
	close(done)

	// Clean up unix socket
	if listen != "" && listen[0] == '/' {
		os.Remove(listen)
	}
}
