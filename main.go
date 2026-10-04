package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"runtime/debug"
	"strings"
)

const (
	varKeyName   = "OPENAI_KEY"
	varModelName = "AITERM_MODEL"
	varURLName   = "AITERM_URL"
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
	prompt      string
	showVersion bool
}

func main() {
	if err := run(os.Args[1:], os.Getenv, os.Stdin, os.Stdout, os.Stderr); err != nil {
		if errors.Is(err, errUsage) {
			os.Exit(2)
		}
		fmt.Fprintf(os.Stderr, "aiterm: %v\n", err)
		os.Exit(1)
	}
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

	openAIClient := NewOpenAIProvider(cfg.apiKey, &OpenAIOptions{
		ProviderOptions: &ProviderOptions{
			URL: cfg.url,
		},
		model:            cfg.model,
		temperature:      1.0,
		maxTokens:        256,
		topP:             1.0,
		frequencyPenalty: 0.0,
		presencePenalty:  0.0,
		withContext:      true,
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
	fs.BoolVar(&cfg.showVersion, "version", false, "print version information and exit")
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), `Usage: aiterm [flags] "natural language request"`)
		fmt.Fprintln(fs.Output(), "\nTranslate a natural language request into a shell command, then run, copy or refine it.")
		fmt.Fprintln(fs.Output(), "\nFlags:")
		fs.PrintDefaults()
	}

	if err := fs.Parse(args); err != nil {
		return nil, err
	}

	cfg.apiKey = firstNonEmpty(cfg.apiKey, getenv(varKeyName))
	cfg.url = firstNonEmpty(cfg.url, getenv(varURLName), defaultEndPoint)
	cfg.model = firstNonEmpty(cfg.model, getenv(varModelName), defaultModel)
	// Accept unquoted prompts too: aiterm list all go files
	cfg.prompt = strings.TrimSpace(strings.Join(fs.Args(), " "))
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

// versionString reports the GoReleaser stamped version, falling back to the
// module version recorded by `go install github.com/Thakay/aiterm@vX.Y.Z`.
func versionString() string {
	v := version
	if v == "dev" {
		if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version != "" && info.Main.Version != "(devel)" {
			v = info.Main.Version
		}
	}
	if commit == "none" {
		return "aiterm " + v
	}
	return fmt.Sprintf("aiterm %s (commit %s, built %s)", v, commit, date)
}
