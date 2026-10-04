package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/atotto/clipboard"
	"golang.org/x/term"
)

type App struct {
	Client          APIProvider
	in              io.Reader
	reader          *bufio.Reader
	out             io.Writer
	errOut          io.Writer
	copyToClipboard func(string) error
	userRequest     string
	fetchCfg        FetchConfig
}

func NewApp(provider APIProvider, userReq string) *App {
	return newApp(provider, userReq, os.Stdin, os.Stdout, os.Stderr)
}

func newApp(provider APIProvider, userReq string, in io.Reader, out, errOut io.Writer) *App {
	return &App{
		Client:          provider,
		in:              in,
		reader:          bufio.NewReader(in),
		out:             out,
		errOut:          errOut,
		copyToClipboard: clipboard.WriteAll,
		userRequest:     userReq,
		fetchCfg:        provider.newFetchConfig(true),
	}
}

func (a *App) Run() error {

	if ok, err := a.HandleEmptyAPIKey(); err != nil {
		return err
	} else if !ok {
		fmt.Fprintln(a.out, "No API key was provided. Set the OPENAI_KEY environment variable or pass -key. Exiting...")
		return nil
	}
	for {
		cmdstr, err := a.Client.fetch(a.userRequest, a.fetchCfg)
		if err != nil {
			err = a.HandleAPIError(err)
			var keyErr *APIKeyError
			if !errors.As(err, &keyErr) {
				return err
			}
			// Let the user fix a rejected key without restarting aiterm.
			if ok, perr := a.promptForAPIKey("Enter a valid OpenAI API key for this session, or leave empty to exit: "); perr != nil || !ok {
				return err
			}
			continue
		}

		ok, validCmdstr := a.ValidateCmd(cmdstr)
		if !ok {
			if cleaned := cleanCmd(cmdstr); hasHiddenRunes(cleaned) {
				fmt.Fprintf(a.out, "The reply contains control or invisible characters, so it will not be offered to run: %q\n", cleaned)
			} else {
				fmt.Fprintln(a.out, "That does not look like a command.")
			}
			fmt.Fprintln(a.out, "Please retry with a different prompt (or q to quit).")
			fmt.Fprint(a.out, "-> ")
			input, err := a.readInput()
			if err != nil {
				return err
			}
			if strings.EqualFold(input, "q") {
				return nil
			}
			a.userRequest = input
			continue
		}

		end, err := a.HandleCmd(validCmdstr)
		if err != nil {
			return fmt.Errorf("failed handling the command: %w", err)
		}
		if end {
			return nil
		}
	}
}

func (a *App) HandleCmd(cmd string) (bool, error) {
	for {
		fmt.Fprint(a.out, "\n\n\n")
		fmt.Fprintf(a.out, "Here is the command --> %s <-- \n", cmd)
		fmt.Fprintln(a.out, "#####--------#####")
		fmt.Fprintln(a.out, "*) To execute it enter: y")
		fmt.Fprintln(a.out, "*) To copy to clipboard and exit to terminal enter: c")
		fmt.Fprintln(a.out, "*) To copy to clipboard and edit and run with aiterm enter: g")
		fmt.Fprintln(a.out, "*) To send a new request with context enter: r")
		fmt.Fprintln(a.out, "*) To send a new request without context enter: w")
		fmt.Fprintln(a.out, "*) To exit enter: q")
		if lines := strings.Count(cmd, "\n") + 1; lines > 1 {
			fmt.Fprintf(a.out, "Note: this command has %d lines, and all of them run when you enter y.\n", lines)
		}
		fmt.Fprint(a.out, "-> ")
		input, err := a.readInput()
		if err != nil {
			return true, err
		}
		switch strings.ToLower(input) {
		case "y":
			if err := a.executeCmd(cmd); err != nil {
				// Go back to the menu so the command can be edited or re-requested.
				fmt.Fprintf(a.out, "\nThe command failed: %v\n", err)
				continue
			}
			return true, nil
		case "c":
			if err := a.copyToClipboard(cmd); err != nil {
				fmt.Fprintf(a.out, "Could not copy to the clipboard (%v). Here is the command:\n%s\n", err, cmd)
				return true, nil
			}
			fmt.Fprintln(a.out, "Command copied to clipboard. Exiting.")
			return true, nil
		case "g":
			if err := a.copyToClipboard(cmd); err != nil {
				fmt.Fprintf(a.out, "Could not copy to the clipboard (%v). Type the edited command:", err)
			} else {
				fmt.Fprint(a.out, "Command copied to clipboard. You can paste it into the terminal:")
			}
			input, err := a.readInput()
			if err != nil {
				return true, err
			}
			if input != "" {
				cmd = input
			}
		case "r":
			fmt.Fprint(a.out, "(+c)Enter the new prompt:")
			input, err := a.readInput()
			if err != nil {
				return true, err
			}
			a.userRequest = input
			a.fetchCfg = a.Client.newFetchConfig(true)
			return false, nil
		case "w":
			fmt.Fprint(a.out, "(-c)Enter the new prompt:")
			input, err := a.readInput()
			if err != nil {
				return true, err
			}
			a.userRequest = input
			a.fetchCfg = a.Client.newFetchConfig(false)
			return false, nil
		case "q":
			return true, nil
		default:
			fmt.Fprintln(a.out, "unknown command to exit press 'q'.")
		}
	}
}

func (a *App) HandleEmptyAPIKey() (bool, error) {
	if a.Client.hasAPIKey() {
		return true, nil
	}

	fmt.Fprintln(a.out, "No API key found. Set the OPENAI_KEY environment variable (or pass -key) to skip this prompt.")
	return a.promptForAPIKey("Enter your OpenAI API key for this session, or leave empty to exit: ")
}

// promptForAPIKey asks for a key for this session. It reports false when the
// user leaves the answer empty.
func (a *App) promptForAPIKey(prompt string) (bool, error) {
	fmt.Fprint(a.out, prompt)
	input, err := a.readSecret()
	if err != nil {
		return false, err
	}
	if input == "" {
		return false, nil
	}
	a.Client.setAPIKey(input)
	return true, nil
}

func (a *App) HandleInvalidAPIKey() {
	fmt.Fprintln(a.errOut, "The API key was rejected. Please set a valid key in OPENAI_KEY (or pass -key) and retry.")
}

// cleanCmd strips the decorations models sometimes add around a command even when
// told not to: markdown code fences, inline backticks and "$ " prompts.
func cleanCmd(cmd string) string {
	cmd = strings.TrimSpace(strings.ReplaceAll(cmd, "\r\n", "\n"))

	if len(cmd) >= 6 && strings.HasPrefix(cmd, "```") && strings.HasSuffix(cmd, "```") {
		cmd = strings.TrimSuffix(strings.TrimPrefix(cmd, "```"), "```")
		// In a multi line fence the opening line holds the info string
		// (usually a language such as bash), never part of the command.
		if i := strings.IndexByte(cmd, '\n'); i >= 0 {
			cmd = cmd[i+1:]
		}
		cmd = strings.TrimSpace(cmd)
	}

	if len(cmd) >= 2 && strings.HasPrefix(cmd, "`") && strings.HasSuffix(cmd, "`") && strings.Count(cmd, "`") == 2 {
		cmd = strings.TrimSpace(cmd[1 : len(cmd)-1])
	}

	if strings.HasPrefix(cmd, "$ ") {
		lines := strings.Split(cmd, "\n")
		for i, line := range lines {
			lines[i] = strings.TrimPrefix(line, "$ ")
		}
		cmd = strings.Join(lines, "\n")
	}

	return strings.TrimSpace(cmd)
}

// hiddenRune reports whether r can change how a command looks in the terminal
// without being visible itself: control characters (a carriage return or an
// escape sequence can overwrite what was printed), bidi overrides and zero width
// characters.
func hiddenRune(r rune) bool {
	if r == '\n' || r == '\t' {
		return false
	}
	return unicode.IsControl(r) || unicode.In(r, unicode.Cf, unicode.Zl, unicode.Zp)
}

func hasHiddenRunes(s string) bool {
	return !utf8.ValidString(s) || strings.IndexFunc(s, hiddenRune) >= 0
}

// ValidateCmd cleans a model reply and reports whether it can be offered as a
// command. Replies with hidden characters are refused, because what the
// terminal shows could differ from what sh would run.
func (a *App) ValidateCmd(cmd string) (bool, string) {
	cmd = cleanCmd(cmd)
	if normalized := strings.TrimSuffix(strings.ToLower(cmd), "."); cmd == "" || normalized == "not a command" || hasHiddenRunes(cmd) {
		return false, ""
	}
	return true, cmd
}

func (a *App) HandleAPIError(err error) error {
	err = a.Client.handleAPIError(err)
	var apiErr *APIKeyError
	if errors.As(err, &apiErr) {
		a.HandleInvalidAPIKey()
	}
	return err
}

func (a *App) readInput() (string, error) {
	input, err := a.reader.ReadString('\n')
	if err != nil {
		// Accept a final line that is not newline terminated (e.g. piped input).
		if errors.Is(err, io.EOF) && input != "" {
			return strings.TrimSpace(input), nil
		}
		return "", &InputReadError{err}
	}
	return strings.TrimSpace(input), nil
}

// readSecret reads a line without echoing it when stdin is a terminal, so a key
// typed at the prompt does not end up in the scrollback.
func (a *App) readSecret() (string, error) {
	if f, ok := a.in.(*os.File); ok && a.reader.Buffered() == 0 && term.IsTerminal(int(f.Fd())) {
		b, err := term.ReadPassword(int(f.Fd()))
		fmt.Fprintln(a.out)
		if err != nil {
			return "", &InputReadError{err}
		}
		return strings.TrimSpace(string(b)), nil
	}
	return a.readInput()
}

// executeCmd runs cmd through sh, streaming its output so long running and
// interactive commands behave as if they were typed into the terminal.
func (a *App) executeCmd(cmd string) error {
	fmt.Fprintf(a.out, "\nRunning: %s\n\n", cmd)
	c := exec.Command("sh", "-c", cmd)
	c.Stdin = a.in
	c.Stdout = a.out
	c.Stderr = a.errOut

	// Ctrl+C should stop the command, not aiterm: the command still receives the
	// interrupt from the terminal, and aiterm goes back to the menu.
	interrupts := make(chan os.Signal, 1)
	signal.Notify(interrupts, os.Interrupt)
	defer signal.Stop(interrupts)

	return c.Run()
}
