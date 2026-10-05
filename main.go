// Command aiterm translates a natural language request into a shell command
// with an OpenAI compatible chat completions API, then lets you run it, copy it,
// edit it or refine it with a follow-up request. Nothing runs until you confirm.
//
// Usage:
//
//	aiterm [flags] "natural language request"
//
// Run aiterm -h for the list of flags and environment variables.
package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"runtime/debug"
	"strings"
	"syscall"
	"time"
)

const (
	varKeyName     = "OPENAI_KEY"
	varModelName   = "AITERM_MODEL"
	varURLName     = "AITERM_URL"
	varTimeoutName = "AITERM_TIMEOUT"
)

// Set at build time by GoReleaser through -ldflags.
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

// errUsage is returned when aiterm was invoked incorrectly; the details have
// already been printed by the time it is returned.
var errUsage = errors.New("usage error")

type config struct {
	apiKey      string
	url         string
	model       string
	timeout     time.Duration
	prompt      string
	showVersion bool
}

func main() {
	if err := run(os.Args[1:], os.Getenv, os.Stdin, os.Stdout, os.Stderr); err != nil {
		if errors.Is(err, errUsage) {
			os.Exit(2)
		}
		reportRuntimeError(os.Stderr, err)
		os.Exit(1)
	}
}

func isLoopbackHost(host string) bool {
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func reportRuntimeError(w io.Writer, err error) {
	fmt.Fprintf(w, "aiterm: %v\n", err)
	if hint := runtimeErrorHint(err); hint != "" {
		fmt.Fprintf(w, "hint: %s\n", hint)
	}
}

func runtimeErrorHint(err error) string {
	var apiErr *OAIAPIError
	if errors.As(err, &apiErr) {
		if apiErr.StatusCode == http.StatusUnauthorized || apiErr.Code == "invalid_api_key" {
			return ""
		}
		switch {
		case apiErr.StatusCode == http.StatusNotFound || apiErr.Code == "model_not_found":
			return "The model is wrong or not available on this endpoint; check -model or AITERM_MODEL."
		case apiErr.StatusCode == http.StatusTooManyRequests && apiErr.Code == "insufficient_quota":
			return "The account has no credit left; check billing."
		case apiErr.StatusCode == http.StatusTooManyRequests:
			return "Rate limited; wait a moment and retry."
		case apiErr.StatusCode >= 500 && apiErr.StatusCode <= 599:
			return "The service had a problem; retry later."
		}
	}

	var requestErr *url.Error
	if errors.As(err, &requestErr) && errors.Is(err, syscall.ECONNREFUSED) {
		endpoint, parseErr := url.Parse(requestErr.URL)
		if parseErr == nil && isLoopbackHost(endpoint.Hostname()) {
			return "The local server (Ollama or LM Studio) is not running."
		}
	}

	return ""
}

func run(args []string, getenv func(string) string, in io.Reader, out, errOut io.Writer) error {
	cfg, err := parseArgs(args, getenv, errOut)
	if errors.Is(err, flag.ErrHelp) {
		return nil
	}
	if err != nil {
		return errUsage
	}

	if cfg.showVersion {
		fmt.Fprintln(out, versionString())
		return nil
	}

	if cfg.prompt == "" {
		fmt.Fprintln(errOut, "No prompt was provided.")
		fmt.Fprintln(errOut, `Usage: aiterm [flags] "natural language request" (see aiterm -h)`)
		return errUsage
	}

	if insecureRemoteURL(cfg.url) {
		fmt.Fprintf(errOut, "warning: %s uses plain http, so your API key and requests are sent unencrypted\n", cfg.url)
	}

	openAIClient := NewOpenAIProvider(cfg.apiKey, &OpenAIOptions{
		ProviderOptions: &ProviderOptions{
			URL: cfg.url,
		},
		model:       cfg.model,
		timeout:     cfg.timeout,
		withContext: true,
	})

	return newApp(openAIClient, cfg.prompt, in, out, errOut).Run()
}

// parseArgs reads the flags and the prompt. Flags win over environment
// variables, which win over the built in defaults. The API key is never used as
// a flag default so it cannot leak through the -h output.
func parseArgs(args []string, getenv func(string) string, output io.Writer) (*config, error) {
	cfg := &config{}
	fs := flag.NewFlagSet("aiterm", flag.ContinueOnError)
	fs.SetOutput(output)
	fs.StringVar(&cfg.apiKey, "key", "", "OpenAI API key (default $"+varKeyName+")")
	fs.StringVar(&cfg.url, "url", "", fmt.Sprintf("chat completions endpoint of an OpenAI compatible API (default $%s or %q)", varURLName, defaultEndPoint))
	fs.StringVar(&cfg.model, "model", "", fmt.Sprintf("model to use (default $%s or %q)", varModelName, defaultModel))
	fs.DurationVar(&cfg.timeout, "timeout", 0, fmt.Sprintf("how long to wait for the API, e.g. 30s or 5m (default $%s or %v)", varTimeoutName, defaultTimeout))
	fs.BoolVar(&cfg.showVersion, "version", false, "print version information and exit")
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), `Usage: aiterm [flags] "natural language request"`)
		fmt.Fprintln(fs.Output(), "\nTranslate a natural language request into a shell command, then run, copy or refine it.")
		fmt.Fprintln(fs.Output(), "Flags go before the request.")
		fmt.Fprintln(fs.Output(), "\nFlags:")
		fs.PrintDefaults()
	}

	if err := fs.Parse(args); err != nil {
		return nil, err
	}

	// The flag package stops at the first word of the request, so a flag typed
	// after it would silently become part of the request (a -key value would
	// even be sent to the model). Reject that unless the flags were ended with "--".
	rest := fs.Args()
	if n := len(args) - len(rest); n == 0 || args[n-1] != "--" {
		for _, a := range rest {
			name, _, _ := strings.Cut(strings.TrimLeft(a, "-"), "=")
			if strings.HasPrefix(a, "-") && fs.Lookup(name) != nil {
				err := fmt.Errorf("flag %s must come before the request (put -- before the request to send it as text)", a)
				fmt.Fprintln(fs.Output(), err)
				fs.Usage()
				return nil, err
			}
		}
	}

	if cfg.timeout == 0 {
		if raw := getenv(varTimeoutName); raw != "" {
			d, err := time.ParseDuration(raw)
			if err != nil {
				err = fmt.Errorf("invalid %s %q: use a duration such as 30s or 5m", varTimeoutName, raw)
				fmt.Fprintln(fs.Output(), err)
				return nil, err
			}
			cfg.timeout = d
		}
	}
	if cfg.timeout < 0 {
		err := fmt.Errorf("the timeout must be positive, got %v", cfg.timeout)
		fmt.Fprintln(fs.Output(), err)
		return nil, err
	}
	if cfg.timeout == 0 {
		cfg.timeout = defaultTimeout
	}

	cfg.apiKey = firstNonEmpty(cfg.apiKey, getenv(varKeyName))
	cfg.url = firstNonEmpty(cfg.url, getenv(varURLName), defaultEndPoint)
	cfg.model = firstNonEmpty(cfg.model, getenv(varModelName), defaultModel)
	// Accept unquoted prompts too: aiterm list all go files
	cfg.prompt = strings.TrimSpace(strings.Join(rest, " "))
	return cfg, nil
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

// insecureRemoteURL reports whether raw is a plain http URL that leaves this
// machine. Plain http to localhost (Ollama, LM Studio) is fine.
func insecureRemoteURL(raw string) bool {
	u, err := url.Parse(raw)
	if err != nil || !strings.EqualFold(u.Scheme, "http") {
		return false
	}
	if isLoopbackHost(u.Hostname()) {
		return false
	}
	return true
}

// versionString reports the GoReleaser stamped version, falling back to the
// module version recorded by `go install github.com/Thakay/aiterm@vX.Y.Z`.
func versionString() string {
	v := version
	if v == "dev" {
		if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version != "" && info.Main.Version != "(devel)" {
			v = info.Main.Version
		}
	}
	v = strings.TrimPrefix(v, "v")
	if commit == "none" {
		return "aiterm " + v
	}
	return fmt.Sprintf("aiterm %s (commit %s, %s)", v, commit, date)
}
