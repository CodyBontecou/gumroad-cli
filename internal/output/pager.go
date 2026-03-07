package output

import (
	"io"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

var isStdoutTerminal = func() bool {
	return isTerminalWriter(os.Stdout)
}

// Pager wraps output through a system pager (e.g. less) when stdout is
// an interactive terminal. Falls back to direct stdout otherwise.
// Callers must Close the pager to wait for the process to finish.
type Pager struct {
	writer      io.Writer
	widthSource io.Writer
	cmd         *exec.Cmd
	pipe        io.WriteCloser
}

// NewPager starts a pager process if stdout is an interactive TTY with a
// capable terminal. Returns a passthrough to os.Stdout otherwise.
func NewPager() *Pager {
	return NewPagerTo(os.Stdout, os.Stderr)
}

// NewPagerTo writes to the supplied stdout writer and only spawns a system
// pager when that writer is a terminal-backed file.
func NewPagerTo(stdout, stderr io.Writer) *Pager {
	if stdout == nil {
		stdout = os.Stdout
	}
	if stderr == nil {
		stderr = os.Stderr
	}

	file, ok := pagerOutputFile(stdout)
	if !ok || isDumbTerminal() {
		return newPassthroughPager(stdout)
	}

	cmd := pagerCommand(defaultPagerCommand())
	if cmd == nil {
		return newPassthroughPager(stdout)
	}
	cmd.Stdout = file
	if errFile, ok := stderr.(*os.File); ok {
		cmd.Stderr = errFile
	}

	pipe, err := cmd.StdinPipe()
	if err != nil {
		return newPassthroughPager(stdout)
	}

	if err := cmd.Start(); err != nil {
		return newPassthroughPager(stdout)
	}

	return &Pager{writer: pipe, widthSource: stdout, cmd: cmd, pipe: pipe}
}

func newPassthroughPager(w io.Writer) *Pager {
	return &Pager{writer: w, widthSource: w}
}

func pagerOutputFile(w io.Writer) (*os.File, bool) {
	file, ok := w.(*os.File)
	if !ok {
		return nil, false
	}
	if file == os.Stdout {
		return file, isStdoutTerminal()
	}
	return file, isTerminalWriter(w)
}

func defaultPagerCommand() string {
	if pagerCmd, ok := os.LookupEnv("PAGER"); ok {
		return strings.TrimSpace(pagerCmd)
	}
	return "less -FIRX"
}

func pagerCommand(pagerCmd string) *exec.Cmd {
	if pagerCmd == "" {
		return nil
	}

	if needsShellParsing(pagerCmd) {
		return shellCommand(pagerCmd)
	}

	parts := strings.Fields(pagerCmd)
	if len(parts) == 0 {
		return nil
	}
	//nolint:gosec // PAGER is an intentional user-controlled command, matching standard CLI behavior.
	return exec.Command(parts[0], parts[1:]...)
}

func needsShellParsing(value string) bool {
	return strings.ContainsAny(value, `"'|\&;<>$()`)
}

func shellCommand(command string) *exec.Cmd {
	if runtime.GOOS == "windows" {
		//nolint:gosec // PAGER may intentionally contain shell syntax when set by the user.
		return exec.Command("cmd", "/C", command)
	}
	//nolint:gosec // PAGER may intentionally contain shell syntax when set by the user.
	return exec.Command("sh", "-c", command)
}

func (p *Pager) Write(b []byte) (int, error) {
	return p.writer.Write(b)
}

func (p *Pager) terminalWidth() (int, bool) {
	if p == nil || p.widthSource == nil {
		return 0, false
	}
	return terminalWidth(p.widthSource)
}

func (p *Pager) Close() error {
	if p.pipe != nil {
		p.pipe.Close()
	}
	if p.cmd != nil {
		return p.cmd.Wait()
	}
	return nil
}
