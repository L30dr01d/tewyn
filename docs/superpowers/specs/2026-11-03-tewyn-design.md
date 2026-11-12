# Tewyn

Tewyn is a local tool for keeping a personal library of agent skills and pinning a chosen set into one project. It exists because marketplaces are the wrong catalog, skills installed in a global place show up in work where they do not belong, and every new project otherwise means setting the same skills up again for Claude, Cursor, and Codex.

It is for one person, on their own machine. The library is the catalog. The project pin is what a given repo is actually using, at a known commit.

## Goals

- Keep a local library of favorite skills, stored as GitHub links and arranged in groups.
- Add a folder of skills, or one skill, with a single command.
- In a project, choose skills in a TUI, download them at a specific commit, and remove the ones you no longer want.
- Point Claude, Cursor, and Codex at that one downloaded copy with symlinks.
- Update a project's skills on demand and print a report.

## Non-goals

- A GUI, a marketplace, accounts, or sharing the library with anyone else.
- Installing skills so they apply to every project.
- Agents other than Claude, Cursor, and Codex.
- A screen or command for regrouping, renaming, or deleting library groups. That is a hand edit of the library file.
- A repair screen after an update, and any list of outdated skills before an update is run.
- Signing in to GitHub. The tool uses whatever access the machine already has.
- Writing a gitignore, or creating git commits. Whether `.tewyn` is committed is up to the person using the tool.

## Anti-goals

- A skill turned on once leaking into every project.
- Setting up the same skills separately for each agent.
- An update that changes a project without being asked.
- A failed update that deletes a skill or leaves a half-written folder in its place.

## Constraints

- Implemented in Go. The command is `tewyn`.
- The library UI for day-to-day use is a TUI. Adding skills and updating them are command-line flags.
- A skill URL belongs to one group. The tool never writes an empty group.
- A project can contain only one skill of a given name. The library can hold two skills with the same name, in different groups.
- Library file: `~/.config/tewyn/library.json`. This path is literal. Do not use the operating system's "config dir" (`~/Library/Application Support` on macOS). If `XDG_CONFIG_HOME` is set, the file is `$XDG_CONFIG_HOME/tewyn/library.json` instead. (Call.)
- Project files live in `.tewyn/` inside the project directory.

## Library

`tewyn --add-skills <url>` reads one GitHub URL and creates one new group. It does not need to be run inside a project.

Two URL shapes:

- A folder of skills, such as `https://github.com/obra/superpowers/tree/main/skills`. The folder does not itself contain `SKILL.md`. Each direct child folder that contains `SKILL.md` is a skill.
- One skill, such as `https://github.com/DietrichGebert/ponytail/tree/main/skills/ponytail`. The folder contains `SKILL.md`. That folder is the skill. Child folders are not scanned.

The group is named after the repository (`superpowers`, `ponytail`). The skill is named after its folder (`brainstorming`, `ponytail`). The stored link is the GitHub tree URL of that skill's own folder.

Accepted URLs look like `https://github.com/<owner>/<repo>/tree/<ref>/<path>`. `<ref>` is one path segment, the branch or tag (`main`). The rest is the path inside the repo.

Direct children only. A skill nested further down is not imported. A child that has no `SKILL.md` is skipped. If the folder itself contains `SKILL.md`, it is one skill even if some child also contains `SKILL.md`. (Call.)

On success, stdout is the group name, every skill name, and the path of the library file:

```
added group superpowers
  brainstorming
  systematic-debugging
  writing-plans
saved to ~/.config/tewyn/library.json
```

The command always creates a new group. It does not merge into an existing group. If a group with that repository's name already exists, it prints an error and leaves the file unchanged:

```
error: group superpowers already exists
```

If the URL yields no skills, it prints an error and leaves the file unchanged:

```
error: no skills at that path
```

Errors go to stderr. Success goes to stdout. A failed add exits non-zero and writes nothing. (Call.)

The library file:

```json
{
  "groups": [
    {
      "name": "superpowers",
      "skills": [
        {
          "name": "brainstorming",
          "url": "https://github.com/obra/superpowers/tree/main/skills/brainstorming"
        }
      ]
    },
    {
      "name": "ponytail",
      "skills": [
        {
          "name": "ponytail",
          "url": "https://github.com/DietrichGebert/ponytail/tree/main/skills/ponytail"
        }
      ]
    }
  ]
}
```

There is no TUI for editing this file. To build a group that is not exactly what one URL produced, edit the JSON by hand.

When the tool reads the file:

- An empty group is skipped and does not appear in the TUI. (Call.)
- The same skill URL in two groups is an error. The TUI and `--add-skills` report it and do not continue. (Call.)
- Two skills with the same name and different URLs are allowed.
- Two groups with the same name are an error, reported the same way. (Call.)

## Project files

```
my-project/
  .tewyn/
    config.json
    skills/
      brainstorming/
        SKILL.md
        scripts/…
      ponytail/
        SKILL.md
  .claude/skills  -> ../.tewyn/skills
  .cursor/skills  -> ../.tewyn/skills
  .agents/skills  -> ../.tewyn/skills
```

`.tewyn/config.json` is the list of skills pinned in this project. Each entry has the skill name, the GitHub URL copied from the library at the time it was added, and the full commit SHA that was downloaded:

```json
{
  "skills": [
    {
      "name": "brainstorming",
      "url": "https://github.com/obra/superpowers/tree/main/skills/brainstorming",
      "commit": "a1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9b0",
      "error": ""
    }
  ]
}
```

`error` is empty when the skill is fine. `--update-skills` sets it when that skill could not be updated, and clears it when a later update succeeds. The TUI shows the text on that skill's row. (Call: the field exists so the mark survives between runs. The words are the update reason, such as `path changed`, not a separate label.)

Downloaded files go in `.tewyn/skills/<skill-name>/`. The download is the whole folder at that commit, including `SKILL.md` and every other file and subdirectory.

Opening the TUI does not create `.tewyn`. The directory and `config.json` are written on the first apply that adds or removes a skill. (Call.)

The tool does not write a gitignore and does not run git.

## Project TUI

`tewyn` with no arguments, run in the project directory, opens the TUI.

The skills screen is two panels.

- Left, "saved": the whole library. Groups are headings. Skills are the rows, and each row has a checkbox. Checking a row marks that skill to be added to the project.
- Right, "this repo": only skills already in `.tewyn/config.json`. They are grouped the way the library groups them, by matching URL. Checking a row marks that skill to be deleted. A project skill whose URL is no longer in the library still appears on the right, by its name, so it can be deleted. (Call.)

Keys:

- Up and down move through the rows. Left and right move between the panels. (Call.)
- Space toggles the skill under the cursor. (Call.)
- `g` toggles every skill in the group under the cursor, on the panel the cursor is in. If any skill in that group is unchecked, `g` checks them all. If all are checked, `g` unchecks them all.
- `a` applies.
- Tab switches between the skills screen and the agents screen, including while a download is running.
- `q` quits. Skills already written to disk stay. A download that has not finished is discarded, and no half-written folder is left behind. (Call.)

Apply does two steps in order:

1. Delete every skill checked on the right. Remove its folder from `.tewyn/skills` and its entry from `config.json`.
2. Download every skill checked on the left. Copy its library URL into `config.json`, resolve the branch or tag in that URL to a commit, download the folder at that commit, and record the full SHA.

A name may appear only once in the project. The check happens after the deletes. The names still in the project, plus the names about to be downloaded, must be unique. If a name would appear twice, the TUI shows an error naming that skill. None of the checked adds that share the colliding name are downloaded. Adds with a unique name still proceed. Deletes have already happened. (Call.)

Checking a skill on the right and a different library skill of the same name on the left is the swap: the old one is deleted first, then the new one is downloaded.

The agents screen lists:

- Claude, `.claude/skills`
- Cursor, `.cursor/skills`
- Codex, `.agents/skills`

The checkbox matches the filesystem. It is checked when that path is a symlink to `../.tewyn/skills`. Nothing about agents is stored in `config.json`. (Call.)

Apply on this screen:

- A checked agent gets a relative symlink, `<agent>/skills` → `../.tewyn/skills`. Parent directories (`.claude`, `.cursor`, `.agents`) are created if they are missing.
- An unchecked agent loses that symlink, and only when it already points at `.tewyn/skills`.
- If the path is a real directory, or a symlink pointing somewhere else, the TUI shows an error for that agent and leaves the path alone.
- If the symlink is already correct, apply leaves it.

`g` does nothing on the agents screen. (Call.)

## Update

`tewyn --update-skills` is run in the project directory. It reads `.tewyn/config.json` and does not read the library. If there is no project config, it prints an error and exits non-zero. (Call.)

For each skill, it resolves the branch or tag in the stored URL to the current commit.

- If that commit differs from the stored commit, it downloads the folder again at the new commit, replaces `.tewyn/skills/<name>/`, writes the new commit, and clears `error`. The skill is reported as updated, with the old SHA and the new SHA. The report may show abbreviated SHAs. The file stores the full SHA. (Call.)
- If the commit is unchanged, the skill is reported as up to date. Files are not touched.
- If the path is gone at that commit, or the URL cannot be read, the skill is reported as could not update, with a short reason (`path changed`, or the read error). The folder, the stored commit, and the stored URL stay as they are. `error` is set to that reason.

Each skill is independent. One failure does not stop the others. (Call.)

The report goes to stdout. A heading is printed only when that list has at least one skill:

```
updated
  brainstorming  a1b2c3d → e4f5a6b
up to date
  ponytail
could not update
  writing-plans  path changed
```

If the project has no skills, it prints `no skills` and exits zero. The command exits non-zero if any skill could not be updated, and zero otherwise. (Call.)

There is no repair screen. Replacing a skill that could not be updated is a separate step in the project TUI: check it on the right to delete it, and check a library skill on the left to add. Adding copies the library's current URL into the project config. A hand edit of `library.json` has no effect on update until that skill is added to the project again.

## Calls

These were settled in the design, not dictated out loud:

- `XDG_CONFIG_HOME` overrides `~/.config` when it is set. The macOS Library path is not used.
- Discovery stops at the folder itself when it contains `SKILL.md`, and otherwise looks only at direct children.
- Add and update exit non-zero on failure. Add errors go to stderr.
- Empty groups in a hand-edited library are hidden. A duplicated skill URL, or two groups with the same name, stops the tool with an error.
- A project skill missing from the library still shows on the right so it can be deleted.
- Arrow keys move. Space toggles one skill. Left and right switch panels.
- `q` keeps finished work and drops an unfinished download without leaving a partial folder.
- A name clash skips every add involved in that clash, after deletes have already been applied. Other adds go through.
- Agent checkboxes are derived from symlinks on disk.
- `g` does nothing on the agents screen.
- Update of one skill does not block the others. The report omits empty sections. No project config is an error. A project with no skills prints `no skills` and exits zero. Any skill that could not update makes the command exit non-zero.
- The `error` field on a project skill is how the TUI keeps showing the update failure.

## Left to the builder

- Which Go TUI library to use.
- How to read a GitHub tree and download a folder at a commit (API, git, or a tarball), as long as the result is the files at that commit and the recorded SHA is the commit the URL's branch or tag resolved to.
- How a branch or tag name is resolved to a commit, including private repositories that the machine can already access.
- The exact abbreviated SHA length in the report.
- Wording of TUI errors beyond the cases above, as long as a name clash names the skill and a blocked agent symlink names the agent.
