package main

import (
	"os"
	"os/exec"

	"github.com/creack/pty"
)

type terminal struct {
	ptmx *os.File
	cmd  *exec.Cmd
}

func newTerminal(shell string, rows, cols uint16) (*terminal, error) {
	cmd := exec.Command(shell)
	cmd.Env = append(os.Environ(), "TERM=xterm-256color")

	ptmx, err := pty.StartWithSize(cmd, &pty.Winsize{
		Rows: rows,
		Cols: cols,
	})
	if err != nil {
		return nil, err
	}

	return &terminal{ptmx: ptmx, cmd: cmd}, nil
}

func (t *terminal) resize(rows, cols uint16) error {
	return pty.Setsize(t.ptmx, &pty.Winsize{Rows: rows, Cols: cols})
}

func (t *terminal) close() {
	t.ptmx.Close()
	t.cmd.Process.Kill()
	t.cmd.Wait()
}
