package main

import (
	"strings"
	"testing"
)

// FuzzValidateCmd checks the guarantees the menu relies on for any model reply:
// an accepted command is never empty, is trimmed, and has no characters that
// could make the terminal show something other than what sh would run.
func FuzzValidateCmd(f *testing.F) {
	for _, seed := range []string{
		"ls -la", "```bash\nls\n```", "```\ndu -sh *\n```", "`ls`", "echo `date`", "$ df -h",
		"not a command", "Not a command.", "", "```", "``````", "``", "touch x #\rls", "a\x1b[2Kb",
		"echo \u202eabc", "$ cd /tmp\n$ ls",
	} {
		f.Add(seed)
	}

	app := &App{}
	f.Fuzz(func(t *testing.T, in string) {
		ok, got := app.ValidateCmd(in)
		if !ok {
			if got != "" {
				t.Fatalf("rejected %q but returned %q", in, got)
			}
			return
		}
		if got == "" || got != strings.TrimSpace(got) {
			t.Fatalf("accepted %q as %q, want a non empty trimmed command", in, got)
		}
		if hasHiddenRunes(got) {
			t.Fatalf("accepted %q with hidden characters: %q", in, got)
		}
		if strings.TrimSuffix(strings.ToLower(got), ".") == "not a command" {
			t.Fatalf("accepted the refusal %q", got)
		}
	})
}
