# Tewyn

Tewyn keeps a personal library of agent skills and pins a chosen set into one project. It downloads those skills right from github and links them for coding agents.

## Usage
- `tewyn --add-skills <github-tree-url>` adds one GitHub folder, or one skill, to the local library and downloads it into `~/.config/tewyn/skills`.
  For example, `tewyn --add-skills https://github.com/obra/superpowers/tree/main/skills` downloads superpowers skills.
- `tewyn` in a project to open the TUI to add skills and symlinks for agents in current folder.
- `tewyn -u` updates the project's skills in `~/.config/tewyn/skills`, then copies them into the project.
- `tewyn -v` or `tewyn --version` prints `tewyn 1.0.0`.
- `tewyn -h` or `tewyn --help` prints the command menu.

## Configs

- The skills library is at `~/.config/tewyn/library.json`
- Skills are downloaded into  `~/.config/tewyn/skills/`

## Commands

### Build

```sh
go build -o tewyn ./cmd/tewyn
```

### Test

```sh
go test ./...
```

### Install

The app is installed to `$GOBIIN`, which is default to `$HOME/go/bin`. Make sure `$HOME/go/bin` is in your `$PATH`.

```sh
go install ./cmd/tewyn
```

## Inspiration
This project is named after an imaginary station from Ivor the Engine by Oliver Postgate and Peter Firmin (Smallfilms).
Used as a homage; no endorsement or affiliation implied.
