# Version flag Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Print tewyn 1.0.0 for `-v`/`--version`, print a command menu for `-h`/`--help`, and mention both flags in the usage error and the README.

**Architecture:** `cmd/tewyn/main.go` keeps a `version` constant and matches `-v`, `--version`, `-h`, and `--help` as single-argument checks in `Main`, before `-u` and the TUI. Help and version text go to stdout through the existing `emit`. Unknown arguments still hit the usage error, with the new usage line.

**Tech Stack:** Go 1.27.1, `Main` in `cmd/tewyn`, no new dependencies.

**Spec:** `docs/superpowers/specs/2026-11-12-version-flag-design.md`

## Global Constraints

- Version is the constant `1.0.0` in the main package. Both flags read that one value.
- Flags stay exact string matches in `Main`.
- One argument only. `-v` and `-h` do not combine with other arguments.
- `-v` and `--version` print `tewyn 1.0.0` on stdout and exit 0.
- `-h` and `--help` print the version and the command menu on stdout and exit 0.
- Unknown arguments print `error: usage: tewyn [--add-skills <url>|-u|-v|-h]` on stderr and exit 2.
- Help and version text stay off stderr. Those commands exit 0.
- Do not change `-u`, `--add-skills`, or the no-argument TUI.
- No release pipeline, git tags, build-time version stamp, or CLI framework.

## Review Focus

- `tewyn -v -h` and `tewyn -h extra`: usage error on stderr, exit 2, empty stdout. A second argument does not print version or help.
- `tewyn --version` and `tewyn --help`: same stdout and exit 0 as `-v` and `-h`.
- Version and help writes: stdout is the exact text, stderr is empty, exit 0.
- `tewyn -V`, `tewyn version`, and `tewyn --v`: usage error on stderr, exit 2, empty stdout. Only the four exact flag strings match.
- `tewyn --update-skills`: still exit 2, and stderr is the new usage line.

---

### Task 1: Version and help flags

**Files:**
- Modify: `cmd/tewyn/main.go`
- Modify: `cmd/tewyn/main_test.go`
- Modify: `README.md`

**Interfaces:**
- Consumes: `func Main(args []string, projectDir string, stdout, stderr io.Writer, getenv func(string) string, tree source.Tree) int` and `func emit(w io.Writer, code int, msg string) int`
- Produces:
  - `const version = "1.0.0"` in package `main`
  - `Main([]string{"-v"}, ...)` and `Main([]string{"--version"}, ...)` return 0 and write `tewyn 1.0.0\n` to stdout
  - `Main([]string{"-h"}, ...)` and `Main([]string{"--help"}, ...)` return 0 and write the help text below to stdout
  - Any other argument list that is not the existing no-arg, `-u`, or `--add-skills` forms returns 2 and writes `error: usage: tewyn [--add-skills <url>|-u|-v|-h]\n` to stderr

- [x] **Step 1: Write the failing test**

Add `TestVersionAndHelp` to `cmd/tewyn/main_test.go`. In `TestUpdateCommandExitCodes`, change the `--update-skills` expected stderr to `error: usage: tewyn [--add-skills <url>|-u|-v|-h]\n`.

```go
func TestVersionAndHelp(t *testing.T) {
	help := "" +
		"tewyn 1.0.0\n" +
		"usage: tewyn [--add-skills <url>|-u|-v|-h]\n" +
		"\n" +
		"  tewyn                         open the project TUI\n" +
		"  tewyn --add-skills <url>      add a GitHub skill tree to the library\n" +
		"  tewyn -u                      refresh cached skills and copy them into the project\n" +
		"  tewyn -v, --version           print the version\n" +
		"  tewyn -h, --help              print this help\n"
	usage := "error: usage: tewyn [--add-skills <url>|-u|-v|-h]\n"
	cases := []struct {
		args []string
		code int
		out  string
		err  string
	}{
		{[]string{"-v"}, 0, "tewyn 1.0.0\n", ""},
		{[]string{"--version"}, 0, "tewyn 1.0.0\n", ""},
		{[]string{"-h"}, 0, help, ""},
		{[]string{"--help"}, 0, help, ""},
		{[]string{"-v", "-h"}, 2, "", usage},
		{[]string{"-h", "extra"}, 2, "", usage},
		{[]string{"-V"}, 2, "", usage},
		{[]string{"version"}, 2, "", usage},
		{[]string{"--v"}, 2, "", usage},
	}
	for _, tc := range cases {
		var out, errb bytes.Buffer
		code := Main(tc.args, t.TempDir(), &out, &errb, func(string) string { return "" }, nil)
		if code != tc.code || out.String() != tc.out || errb.String() != tc.err {
			t.Fatalf("%q: code %d out %q err %q", tc.args, code, out.String(), errb.String())
		}
	}
}
```

- [x] **Step 2: Run test to verify it fails**

Run: `cd /Users/l3mac/codes/skills_management_tool && go test ./cmd/tewyn -count=1 -run 'TestVersionAndHelp|TestUpdateCommandExitCodes'`

Expected: FAIL. `-v` and `-h` take the usage path, and `--update-skills` still prints `error: usage: tewyn [--add-skills <url>|-u]\n`.

- [x] **Step 3: Implement the flags in `cmd/tewyn/main.go`**

Add `const version = "1.0.0"`. Before the `-u` check, when `len(args) == 1`, send `-v` and `--version` to stdout as `"tewyn " + version + "\n"` and exit 0. Send `-h` and `--help` to stdout as the help text asserted by `TestVersionAndHelp`, with the first line built from `version`, and exit 0. Change the usage error to `error: usage: tewyn [--add-skills <url>|-u|-v|-h]\n`.

Under `## Usage` in `README.md`, add:

```markdown
- `tewyn -v` or `tewyn --version` prints `tewyn 1.0.0`.
- `tewyn -h` or `tewyn --help` prints the command menu.
```

- [x] **Step 4: Run test to verify it passes**

Run: `cd /Users/l3mac/codes/skills_management_tool && go test ./cmd/tewyn -count=1`

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add cmd/tewyn/main.go cmd/tewyn/main_test.go README.md
git commit -m "feat: print version 1.0.0 and a help menu"
```
