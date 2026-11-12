# Skill cache Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Download each skill once into the config directory, then copy that folder into a project.

**Architecture:** `library.json` gains the commit the cache folder was downloaded at. `--add-skills` and `-u` download with the existing `source.Tree`. The TUI and `-u` copy `<config>/tewyn/skills/<name>/` into `.tewyn/skills/<name>/`. `-u` reads project skill names from those directories and URLs from the library. It does not read `.tewyn/config.json`.

**Tech Stack:** Go 1.27, existing `source.Tree`, `library.json`, project config.

**Spec:** `docs/superpowers/specs/2026-11-11-skill-cache-design.md`

## Global Constraints

- Library path stays `$XDG_CONFIG_HOME/tewyn/library.json`, or `~/.config/tewyn/library.json`. Skill folders sit beside that file, in `skills/`.
- One skill name, one cache folder. `--add-skills` refuses a skill name that is already in the library.
- The project skill record already stores `name`, `url`, `commit`, and `error`. `-u` does not use that record to decide what to download.
- The library skill stores the same commit string, written when `--add-skills` or `-u` downloads.
- Update is `-u` only. `--update-skills` is not accepted.
- A missing library name, a URL that cannot be read, or a failed download prints a warning at the end of stdout. Other skills still update. That skill's config-dir folder and project folder stay. The command exits 0.
- When the cached commit equals the latest commit, skip the download and still copy.
- Replace a project folder only after the copy has succeeded. Agent symlinks stay pointed at `../.tewyn/skills`.
- Progress words are `downloading <name>` and `copying <name>`. Warnings come last, in the spec's shape.

## Review Focus

- Cached commit already equals the latest commit: no `Download`, and the project folder's bytes become the cache folder's bytes.
- Skill directory name is absent from the library: stdout ends with `warnings` / `  <name>  not in library`, and the project file bytes stay.
- Library URL does not parse: warning reason is `invalid url`, and the project file bytes stay.
- `Download` returns an error: cache file bytes and project file bytes stay, a sibling skill still copies, and `Main` exits 0.
- `--add-skills` fails on the second skill: `library.json` is unchanged, folders created for that invocation are removed, and a cache folder that already existed stays.

---

### Task 1: Library commit and skill-name rule

**Files:**
- Modify: `internal/library/library.go`
- Test: `internal/library/library_test.go`

**Interfaces:**
- Consumes: nothing new
- Produces:
  - `type Skill struct { Name, URL, Commit string }` with json tags `name`, `url`, `commit`
  - `func SkillsDir(libraryPath string) string` — directory beside `library.json`, named `skills`
  - `func (l Library) Find(name string) (Skill, bool)`
  - `Check` error for a repeated skill name: `skill <name> appears twice`

- [ ] **Step 1: Write the failing test**

In `TestLoadSaveAndRules`, set `Commit` to `abc` on the saved brainstorming skill and assert the loaded skill's `Commit` is `abc`. Change the `sameName` assertion so `Check` returns `skill brainstorming appears twice`. Add:

```go
func TestSkillsDirAndFind(t *testing.T) {
	if library.SkillsDir("/cfg/tewyn/library.json") != "/cfg/tewyn/skills" {
		t.Fatal(library.SkillsDir("/cfg/tewyn/library.json"))
	}
	lib := library.Library{Groups: []library.Group{{
		Name: "superpowers",
		Skills: []library.Skill{{Name: "brainstorming", URL: "https://example/b", Commit: "abc"}},
	}}}
	got, ok := lib.Find("brainstorming")
	if !ok || got.Commit != "abc" {
		t.Fatal(got, ok)
	}
	if _, ok := lib.Find("missing"); ok {
		t.Fatal("found missing")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd /Users/l3mac/codes/skills_management_tool && go test ./internal/library -count=1`

Expected: FAIL because `Commit`, `SkillsDir`, or `Find` is missing, or `sameName` still expects `Check` to return nil.

- [ ] **Step 3: Implement the library fields**

Add `Commit` to `Skill`. `SkillsDir` joins `filepath.Dir(libraryPath)` and `skills`. `Find` returns the first skill with that name. `Check` reports a repeated skill name with `skill %s appears twice` after the existing group-name and URL checks.

- [ ] **Step 4: Run test to verify it passes**

Run: `cd /Users/l3mac/codes/skills_management_tool && go test ./internal/library -count=1`

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/library/library.go internal/library/library_test.go
git commit -m "feat: store the cached skill commit in the library"
```

### Task 2: Copy a skill folder

**Files:**
- Create: `internal/project/copy.go`
- Test: `internal/project/copy_test.go`

**Interfaces:**
- Consumes: nothing
- Produces: `func CopyDir(src, dst string) error` — copies `src` to a temp directory beside `dst`, then replaces `dst`. On a copy error, `dst` is unchanged and the temp directory is gone. The result is a real directory.

- [ ] **Step 1: Write the failing test**

```go
func TestCopyDirReplacesOnlyAfterSuccess(t *testing.T) {
	root := t.TempDir()
	src := filepath.Join(root, "src")
	dst := filepath.Join(root, "dst")
	if err := os.MkdirAll(filepath.Join(src, "scripts"), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(src, "scripts", "run.sh"), []byte("new"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(dst, 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dst, "SKILL.md"), []byte("old"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := project.CopyDir(filepath.Join(root, "missing"), dst); err == nil {
		t.Fatal("missing src copied")
	}
	old, err := os.ReadFile(filepath.Join(dst, "SKILL.md"))
	if err != nil || string(old) != "old" {
		t.Fatal(string(old), err)
	}
	if err = project.CopyDir(src, dst); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(filepath.Join(dst, "scripts", "run.sh"))
	if err != nil || string(got) != "new" {
		t.Fatal(string(got), err)
	}
	if _, err = os.Stat(filepath.Join(dst, "SKILL.md")); !os.IsNotExist(err) {
		t.Fatal("old file remains", err)
	}
	info, err := os.Lstat(dst)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		t.Fatal(info, err)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd /Users/l3mac/codes/skills_management_tool && go test ./internal/project -run TestCopyDirReplacesOnlyAfterSuccess -count=1`

Expected: FAIL with `undefined: project.CopyDir`

- [ ] **Step 3: Implement `CopyDir`**

Copy the tree into a temp directory created beside `dst`. If that copy fails, remove the temp directory and return the error. If it succeeds, replace `dst` with the temp directory.

- [ ] **Step 4: Run test to verify it passes**

Run: `cd /Users/l3mac/codes/skills_management_tool && go test ./internal/project -run TestCopyDirReplacesOnlyAfterSuccess -count=1`

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/project/copy.go internal/project/copy_test.go
git commit -m "feat: copy a skill directory onto its destination"
```

### Task 3: Project apply copies from the cache

**Files:**
- Modify: `internal/project/project.go`
- Modify: `internal/project/project_test.go`
- Modify: `cmd/tewyn/main.go` (the TUI `ApplyNotify` call only)

**Interfaces:**
- Consumes: `project.CopyDir`, `library.Skill.Commit`, `library.SkillsDir`
- Produces:
  - `func Apply(ctx context.Context, projectDir, skillsDir string, sel Selection) (Result, error)`
  - `func ApplyNotify(ctx context.Context, projectDir, skillsDir string, sel Selection, added func(string)) (Result, error)`
  - Missing cache folder error: `skill <name> is not in the cache`
  - A cancelled `ctx` returns `ctx.Err()` before copying that skill

- [ ] **Step 1: Write the failing test**

Change `Apply` and `ApplyNotify` calls in `project_test.go` to pass a cache directory. Replace `TestApplyPathChangedMissingConfig` with:

```go
func TestApplyMissingCache(t *testing.T) {
	dir := t.TempDir()
	cache := t.TempDir()
	_, err := project.Apply(context.Background(), dir, cache, project.Selection{
		Add: []library.Skill{{Name: "ponytail", URL: "https://github.com/DietrichGebert/ponytail/tree/main/skills/ponytail", Commit: strings.Repeat("c", 40)}},
	})
	if err == nil || err.Error() != "skill ponytail is not in the cache" {
		t.Fatal(err)
	}
	if _, statErr := os.Stat(filepath.Join(dir, ".tewyn")); !os.IsNotExist(statErr) {
		t.Fatal("tewyn dir created", statErr)
	}
}
```

Point every other add at a real cache folder `<cache>/<name>/SKILL.md` whose bytes are `new`. Assert the project file equals `new`, the saved project commit is the `library.Skill.Commit` passed in, and no test double lists or downloads. `TestApplyDownloadFailRemovesEmptyDir` uses a missing cache name and still asserts `keep` and `config.json` remain. Add a cancelled-context case: `Apply` returns `context.Canceled` and does not create `.tewyn/skills/ponytail`.

- [ ] **Step 2: Run test to verify it fails**

Run: `cd /Users/l3mac/codes/skills_management_tool && go test ./internal/project ./cmd/tewyn -count=1`

Expected: FAIL to compile because `Apply` still takes `source.Tree`.

- [ ] **Step 3: Copy in `ApplyNotify` and pass the cache from `Main`**

`Apply` forwards to `ApplyNotify`. For each unique add, if `ctx` is done, return `ctx.Err()`. Source path is `filepath.Join(skillsDir, name)`. A missing source returns `skill %s is not in the cache` and does not remove an existing project folder of another name. `CopyDir` into `.tewyn/skills/<name>`, then append `project.Skill` from the library skill's name, URL, and commit. Call `added` only after `CopyDir` returns nil. Deletes stay as they are.

In `cmd/tewyn/main.go`, resolve `library.Path(getenv)` inside the TUI callback and call `project.ApplyNotify(ctx, projectDir, library.SkillsDir(path), sel, progress)`.

- [ ] **Step 4: Run test to verify it passes**

Run: `cd /Users/l3mac/codes/skills_management_tool && go test ./internal/project ./cmd/tewyn -count=1`

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/project/project.go internal/project/project_test.go cmd/tewyn/main.go
git commit -m "feat: copy project skills from the config cache"
```

### Task 4: `--add-skills` downloads into the cache

**Files:**
- Modify: `cmd/tewyn/main.go`
- Modify: `cmd/tewyn/main_test.go`
- Modify: `README.md` (the `--add-skills` usage line)

**Interfaces:**
- Consumes: `library.SkillsDir`, `library.Skill.Commit`, `source.Tree`, `(*Library).Add`
- Produces: `--add-skills` stdout prefix `downloading <name>\n` for each skill, then the existing group summary. Each skill's `Commit` is the `List` commit. Files land in `SkillsDir(path)/<name>/`.

- [ ] **Step 1: Write the failing test**

Make `fakeTree.Download` create `dest/SKILL.md` containing `cached` and record `dest`. Change `TestAddSkillsWritesGroup` expected stdout to:

```text
downloading brainstorming
downloading writing-plans
added group superpowers
  brainstorming
  writing-plans
saved to ~/.config/tewyn/library.json
```

Assert both skills' `Commit` values are `strings.Repeat("a", 40)` and `~/.config/tewyn/skills/brainstorming/SKILL.md` is `cached`.

Add `TestAddSkillsDuplicateName`: library already has skill `brainstorming` in another group; the command exits 1 with stderr `error: skill brainstorming appears twice\n`, `Download` is not called, and the saved library is unchanged.

Add `TestAddSkillsDownloadErrorRemovesNewFolders`: `Download` writes the first skill and returns `errors.New("download failed")` for the second. Exit 1, stderr `error: download failed\n`, no `library.json`, the new skill folder is gone, and a pre-seeded `skills/keep/SKILL.md` remains.

- [ ] **Step 2: Run test to verify it fails**

Run: `cd /Users/l3mac/codes/skills_management_tool && go test ./cmd/tewyn -run 'TestAddSkills' -count=1`

Expected: FAIL because stdout has no `downloading` lines and the cache file is absent.

- [ ] **Step 3: Download after the library accepts the group**

Call `Add` on a copy of the loaded library before any download, and return that error unchanged when it fails. For each skill in discover order, parse its URL, `List` it, write `downloading <name>\n`, and `Download` into `SkillsDir/<name>`. Store the listed commit on that skill. If `List` or `Download` fails, remove every skill folder this call created, including a partial folder left by the failed `Download`, and return that error without `Save`. On success, `Add` the group that now has commits, `Save`, then print the existing summary.

- [ ] **Step 4: Run test to verify it passes**

Run: `cd /Users/l3mac/codes/skills_management_tool && go test ./cmd/tewyn -run 'TestAddSkills' -count=1`

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add cmd/tewyn/main.go cmd/tewyn/main_test.go README.md
git commit -m "feat: download added skills into the config cache"
```

README line: `` `tewyn --add-skills <github-tree-url>` adds one GitHub folder, or one skill, to the local library and downloads it into `~/.config/tewyn/skills`. ``

### Task 5: `-u` refreshes the cache, then copies

**Files:**
- Modify: `internal/update/update.go`
- Modify: `internal/update/update_test.go`
- Modify: `cmd/tewyn/main.go`
- Modify: `cmd/tewyn/main_test.go`
- Modify: `README.md` (the update usage line)

**Interfaces:**
- Consumes: `library.Load`, `library.Save`, `library.Find`, `library.SkillsDir`, `project.CopyDir`, `source.Tree`, `source.Parse`
- Produces: `func Run(ctx context.Context, projectDir, libraryPath string, tree source.Tree, stdout io.Writer) error`
  - Missing `.tewyn` directory: error `no project config`
  - `.tewyn` present and no skill directories: write `no skills\n` and return nil
  - Skill names are the direct child directories of `.tewyn/skills`, sorted by name
  - Warning lines, after every progress line: `warnings\n` then `  <name>  <reason>\n`
  - Reasons: `not in library`, `invalid url` when `source.Parse` fails, otherwise `err.Error()`
  - `Main([]string{"-u"}, ...)` calls `Run`. Any other update flag, including `--update-skills`, exits 2 with `error: usage: tewyn [--add-skills <url>|-u]\n`

- [ ] **Step 1: Write the failing test**

Replace the tests in `update_test.go`. Use a project directory that has `.tewyn/skills/<name>/SKILL.md` and a separate library file. Do not write `.tewyn/config.json` in these tests.

`TestUpdateCopiesCurrentAndSkipsDownload`: library commit for `ponytail` equals the commit `List` returns. Cache file bytes are `fresh`. Project file bytes start as `stale`. A scripted tree records `Download` calls. After `Run`, stdout is `copying ponytail\n`, project bytes are `fresh`, and `Download` was not called.

`TestUpdateWarnsAndContinues`: three project skill dirs, sorted behavior:

- `missing` is not in the library. Project file `keep-missing` stays. No cache dir is created.
- `broken` is in the library with URL `https://not-github/broken`. Warning reason `invalid url`. Project file stays.
- `writing-plans` lists as `source.ErrPathChanged`. Warning reason `path changed`. Cache file and project file stay `keep`.
- `brainstorming` lists a new commit. `Download` writes `SKILL.md` of `new` into its dest. Library commit becomes that commit. Project file becomes `new`.

Stdout is exactly:

```text
downloading brainstorming
copying brainstorming
warnings
  broken  invalid url
  missing  not in library
  writing-plans  path changed
```

`Download` dests include only the brainstorming cache path. `TestUpdateDownloadErrorLeavesFolders`: `Download` returns `errors.New("download failed")` for `writing-plans` and succeeds for `ponytail`. Cache and project bytes for `writing-plans` stay. `ponytail` is copied. Stdout contains `downloading writing-plans\n`, `copying ponytail\n`, and `warnings\n  writing-plans  download failed\n`. `Run` returns nil.

In `main_test.go`, replace `TestUpdateCommandExitCodes` so `--update-skills` exits 2 with the new usage line. Missing `.tewyn` for `-u` exits 1 with `error: no project config\n`. An empty `.tewyn` exits 0 with `no skills\n`. A download failure through `Main` exits 0 and stdout contains the warning.

- [ ] **Step 2: Run test to verify it fails**

Run: `cd /Users/l3mac/codes/skills_management_tool && go test ./internal/update ./cmd/tewyn -count=1`

Expected: FAIL because `Run` still reads project config and `--update-skills` is still the flag.

- [ ] **Step 3: Implement `Run` and `-u`**

`Run` stats `.tewyn` and returns `errors.New("no project config")` when it is missing. Read child directory names, sort them, and skip files. Look up each name with `Find`. Parse a found URL; on parse failure warn `invalid url`. `List` the parsed repo, ref, and dir. When the listed commit equals the library commit, skip `Download`. When it differs, write `downloading <name>\n`, download into a temp dir beside the cache folder, and replace the cache folder only if `Download` returns nil. Then set that skill's `Commit` in the loaded library and `library.Save` before the project copy. `CopyDir` from the cache folder to `.tewyn/skills/<name>`, then write `copying <name>\n`. Any per-skill error appends a warning and leaves both existing folders in place. Write the warnings section after the loop. `Main` accepts only `-u` for update and prints `Run`'s stdout as it is written. `Format` and the old report types leave with their callers.

- [ ] **Step 4: Run test to verify it passes**

Run: `cd /Users/l3mac/codes/skills_management_tool && go test ./... -count=1`

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/update/update.go internal/update/update_test.go cmd/tewyn/main.go cmd/tewyn/main_test.go README.md
git commit -m "feat: update cached skills and copy them into the project"
```

README line: `` `tewyn -u` updates the project's skills in `~/.config/tewyn/skills`, then copies them into the project. ``
