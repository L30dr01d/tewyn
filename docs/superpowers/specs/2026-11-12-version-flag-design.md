# Version flag

## Intent

People who install tewyn from GitHub need to see which version they have, and a help menu for the commands. The version is 1.0.0.

## Goals

- `tewyn -v` and `tewyn --version` print `tewyn 1.0.0` on stdout and exit 0.
- `tewyn -h` and `tewyn --help` print the version and a short command menu on stdout and exit 0.
- Unknown arguments stay an error on stderr, exit 2, and the usage line lists the new flags.
- The README usage section lists `-v` and `-h`.

## Non-goals

- A release pipeline, git tags, or stamping the version at build time.
- A CLI framework.
- Changing `-u`, `--add-skills`, or the no-argument TUI.

## Anti-goals

- A version string that changes unless the constant is edited.
- Help or version text on stderr, or a non-zero exit for `-v` or `-h`.

## Constraints

- Go. Flags stay exact string matches in `Main`, the same way `-u` and `--add-skills` are matched.
- Version is the constant `1.0.0` in the main package.
- One argument only. `-v` and `-h` do not combine with other arguments.

## Approaches

- A version constant and two flags in `Main`. Picked. It matches the existing flag checks, and 1.0.0 is the version to ship.
- Inject the version with `-ldflags` when a release is built. That pays off once releases are cut from git tags. There is no release pipeline yet.
- A CLI library. More machinery than these two flags need.

## Decided

- Version text, stdout, exit 0:

```
tewyn 1.0.0
```

- Help text, stdout, exit 0:

```
tewyn 1.0.0
usage: tewyn [--add-skills <url>|-u|-v|-h]

  tewyn                         open the project TUI
  tewyn --add-skills <url>      add a GitHub skill tree to the library
  tewyn -u                      refresh cached skills and copy them into the project
  tewyn -v, --version           print the version
  tewyn -h, --help              print this help
```

- Unknown arguments print `error: usage: tewyn [--add-skills <url>|-u|-v|-h]` on stderr and exit 2.
- `--version` is the long form of `-v`. `--help` is the long form of `-h`.

## Left to the builder

- Where the `1.0.0` constant sits in `cmd/tewyn`, as long as both flags read that one value.
- README wording, as long as usage documents `-v` and `-h`.
