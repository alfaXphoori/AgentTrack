//go:build !windows

package app

import (
	"io"
	"os"
	"os/exec"
	"os/signal"
	"syscall"

	"github.com/creack/pty"
	"golang.org/x/term"
)

func runCmdWithPTY(agentCmdStr string, out io.Writer) error {
	cmd := exec.Command("sh", "-c", agentCmdStr)
	
	ptmx, err := pty.Start(cmd)
	if err != nil {
		return err
	}
	defer func() { _ = ptmx.Close() }()

	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGWINCH)
	go func() {
		for range ch {
			if err := pty.InheritSize(os.Stdin, ptmx); err != nil {
				// ignoring resize errors
			}
		}
	}()
	ch <- syscall.SIGWINCH
	defer func() { signal.Stop(ch); close(ch) }()

	// Make terminal raw
	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		return err
	}
	defer func() { _ = term.Restore(int(os.Stdin.Fd()), oldState) }()

	go func() {
		_, _ = io.Copy(ptmx, os.Stdin)
	}()

	errCh := make(chan error, 1)
	go func() {
		errCh <- cmd.Wait()
		_ = ptmx.Close() // Force io.Copy to unblock when process exits
	}()

	_, _ = io.Copy(out, ptmx)

	return <-errCh
}
