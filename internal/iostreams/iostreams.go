package iostreams

import (
	"io"
	"os"

	"golang.org/x/term"
)

// IOStreams provides access to standard I/O streams with TTY awareness.
type IOStreams struct {
	In     io.ReadCloser
	Out    io.Writer
	ErrOut io.Writer

	colorEnabled bool
	isTerminal   bool
}

// System returns IOStreams connected to the standard OS streams.
func System() *IOStreams {
	stdoutFd := int(os.Stdout.Fd())
	isTerm := term.IsTerminal(stdoutFd)

	return &IOStreams{
		In:           os.Stdin,
		Out:          os.Stdout,
		ErrOut:       os.Stderr,
		colorEnabled: isTerm && os.Getenv("NO_COLOR") == "",
		isTerminal:   isTerm,
	}
}

// Test returns IOStreams suitable for testing (no color, not a terminal).
func Test() *IOStreams {
	return &IOStreams{
		In:           io.NopCloser(nil),
		Out:          io.Discard,
		ErrOut:       io.Discard,
		colorEnabled: false,
		isTerminal:   false,
	}
}

func (s *IOStreams) ColorEnabled() bool {
	return s.colorEnabled
}

func (s *IOStreams) IsTerminal() bool {
	return s.isTerminal
}

func (s *IOStreams) ColorScheme() *ColorScheme {
	return NewColorScheme(s.colorEnabled)
}

// TerminalWidth returns the width of the terminal, or 80 as a fallback.
func (s *IOStreams) TerminalWidth() int {
	if !s.isTerminal {
		return 80
	}
	w, _, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		return 80
	}
	return w
}
