package output

import (
	"fmt"
	"io"
	"os"
	"sync"
	"time"
)

var isSpinnerTerminal = func(w io.Writer) bool {
	return isTerminalWriter(w)
}

const spinnerFrameRate = 80 * time.Millisecond

var spinnerFrames = []string{
	"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏",
}

type Spinner struct {
	message  string
	writer   io.Writer
	done     chan struct{}
	mu       sync.Mutex
	wg       sync.WaitGroup
	active   bool
	started  time.Time
	theme    Theme
	colorful bool
}

func NewSpinner(message string) *Spinner {
	return NewSpinnerTo(message, os.Stderr)
}

func NewSpinnerTo(message string, w io.Writer) *Spinner {
	if w == nil {
		w = os.Stderr
	}
	colorful := NewStylerForWriter(w, false).Enabled()
	return &Spinner{
		message:  message,
		writer:   w,
		done:     make(chan struct{}),
		theme:    NewTheme(colorful),
		colorful: colorful,
	}
}

func (s *Spinner) Start() {
	if !isSpinnerTerminal(s.writer) || isDumbTerminal() {
		return
	}

	s.mu.Lock()
	s.active = true
	s.started = time.Now()
	s.mu.Unlock()

	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		i := 0
		for {
			select {
			case <-s.done:
				fmt.Fprintf(s.writer, "\r\033[K")
				return
			default:
				s.render(i)
				i++
				time.Sleep(spinnerFrameRate)
			}
		}
	}()
}

func (s *Spinner) render(i int) {
	s.mu.Lock()
	msg := s.message
	elapsed := time.Since(s.started)
	s.mu.Unlock()

	frame := spinnerFrames[i%len(spinnerFrames)]
	if s.colorful {
		frame = s.theme.Accent(frame)
		msg = s.theme.Soft(msg)
	}

	if elapsed >= time.Second {
		dim := formatElapsed(elapsed)
		if s.colorful {
			dim = s.theme.Muted("(" + dim + ")")
		} else {
			dim = "(" + dim + ")"
		}
		fmt.Fprintf(s.writer, "\r\033[K%s %s %s", frame, msg, dim)
		return
	}
	fmt.Fprintf(s.writer, "\r\033[K%s %s", frame, msg)
}

// SetMessage replaces the spinner label. It is safe to call from any goroutine
// and takes effect on the next animation tick. Calls before Start or after Stop
// are accepted but nothing is rendered.
func (s *Spinner) SetMessage(message string) {
	s.mu.Lock()
	s.message = message
	s.mu.Unlock()
}

func (s *Spinner) Stop() {
	s.stopWith(StatusNeutral, "")
}

// StopOK clears the spinner and prints a green check + final message in its
// place. Use this for successful command completion.
func (s *Spinner) StopOK(message string) {
	s.stopWith(StatusOK, message)
}

// StopError clears the spinner and prints a red ✗ + final message.
func (s *Spinner) StopError(message string) {
	s.stopWith(StatusErr, message)
}

func (s *Spinner) stopWith(kind StatusKind, finalMsg string) {
	s.mu.Lock()
	wasActive := s.active
	if s.active {
		close(s.done)
		s.active = false
	}
	s.mu.Unlock()
	if wasActive {
		s.wg.Wait()
	}
	if finalMsg != "" {
		if isSpinnerTerminal(s.writer) && !isDumbTerminal() {
			fmt.Fprintln(s.writer, s.theme.Status(kind, finalMsg))
		} else {
			fmt.Fprintln(s.writer, finalMsg)
		}
	}
}

func formatElapsed(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%.1fs", d.Seconds())
	}
	m := int(d / time.Minute)
	sec := int(d/time.Second) % 60
	return fmt.Sprintf("%dm%02ds", m, sec)
}
