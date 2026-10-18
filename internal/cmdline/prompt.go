package cmdline

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/term"
)

// ErrPasswordMismatch is returned when the confirmation of a new password differs.
var ErrPasswordMismatch = errors.New("passwords do not match")

// Prompter asks for values interactively. Labels go to the error output, so the regular output of a command stays
// clean for pipes. On a terminal, secrets are read without echo. With piped input, every value is read as a plain line.
type Prompter struct {
	in       *bufio.Reader
	terminal *os.File
	out      io.Writer
}

// NewPrompter creates a Prompter that reads from the command's input.
func NewPrompter(cmd *cobra.Command) *Prompter {
	return newPrompter(cmd.InOrStdin(), cmd.ErrOrStderr())
}

// newPrompter creates a Prompter that reads from in and writes questions to out.
func newPrompter(in io.Reader, out io.Writer) *Prompter {
	p := &Prompter{
		in:  bufio.NewReader(in),
		out: out,
	}
	if file, ok := in.(*os.File); ok && term.IsTerminal(int(file.Fd())) {
		p.terminal = file
	}
	return p
}

// Line asks for a single line and returns it without the line break.
func (p *Prompter) Line(label string) (string, error) {
	_, _ = fmt.Fprint(p.out, label)
	return p.readLine()
}

// Secret asks for a value without showing it on a terminal.
func (p *Prompter) Secret(label string) (string, error) {
	_, _ = fmt.Fprint(p.out, label)
	if p.terminal == nil {
		return p.readLine()
	}

	secret, err := term.ReadPassword(int(p.terminal.Fd()))
	// ReadPassword consumes the line break without echoing it, so the next output needs its own line
	_, _ = fmt.Fprintln(p.out)
	return string(secret), err
}

// NewPassword asks for a password and, on a terminal, for its confirmation.
func (p *Prompter) NewPassword(label, confirmLabel string) (string, error) {
	password, err := p.Secret(label)
	if err != nil || p.terminal == nil {
		return password, err
	}

	confirmation, err := p.Secret(confirmLabel)
	if err != nil {
		return "", err
	}
	if confirmation != password {
		return "", ErrPasswordMismatch
	}
	return password, nil
}

// Confirm asks a yes/no question. Only "y" and "yes" confirm, anything else declines.
func (p *Prompter) Confirm(label string) (bool, error) {
	answer, err := p.Line(label + " [y/N] ")
	if err != nil {
		return false, err
	}
	switch strings.ToLower(strings.TrimSpace(answer)) {
	case "y", "yes":
		return true, nil
	default:
		return false, nil
	}
}

// readLine reads up to the next line break. Input that ends without one still counts as a line.
func (p *Prompter) readLine() (string, error) {
	line, err := p.in.ReadString('\n')
	if err != nil && (!errors.Is(err, io.EOF) || line == "") {
		return "", err
	}
	return strings.TrimRight(line, "\r\n"), nil
}
