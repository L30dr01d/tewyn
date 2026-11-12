# Skill cache

## Intent

Adding the same skills to each new project downloads them again. Download each skill once into the config directory, then copy those files into the project.

## Goals

- `tewyn --add-skills <url>` records the group in the library and downloads each skill into `<config>/tewyn/skills/<name>/`.
- The TUI copies that folder into `.tewyn/skills/<name>/` when a skill is applied to a project.
- Update is `-u` only. In a project that has `.tewyn`, it takes skill names from `.tewyn/skills/`, refreshes those names in the config directory from the library's GitHub links when the cached commit is not the latest, then replaces the project folders with the cached folders.
- Download and copy each print a line on stdout as they happen.
- A missing library name, a URL that cannot be read, or a failed download prints a warning at the end of stdout. Other skills still update. That skill's config-dir folder and project folder stay. The command exits 0.

## Non-goals

- Keeping `--update-skills`, or migrating existing project files.
- Changing agent symlinks. They stay pointed at `../.tewyn/skills`.
- A git clone per repository.
- A new way to resolve a branch or tag to a commit. The current list-and-download path stays.

## Anti-goals

- A second GitHub download of a skill that is already in the config directory at the latest commit.
- Deleting a project skill folder when that skill could not be refreshed.
- A symlink from the project to the config directory.

## Constraints

- Go. Library file path stays `$XDG_CONFIG_HOME/tewyn/library.json`, or `~/.config/tewyn/library.json`. Skill folders sit beside that file, in `skills/`.
- One skill name, one cache folder. `--add-skills` refuses a skill name that is already in the library.
- The project skill record already stores `name`, `url`, `commit`, and `error`. `-u` does not use that record to decide what to download. Freshness comes from the cache.
- Call: the library skill gains the same `commit` string the project skill already has, written when `--add-skills` or `-u` downloads. `--add-skills` runs outside a project, so the project file cannot hold the cache's commit.

## Approaches

- Folders in the config directory. Picked. `--add-skills` and `-u` download with the existing GitHub folder download. The TUI and `-u` copy that folder into `.tewyn/skills/<name>/`. The library skill stores the commit the folder was downloaded at.
- One clone per repository. A fetch would be shared by every skill in that repo, and the project would still need a copied folder. It adds a second way to talk to GitHub beside the folder download already in the tool.
- Download into each project. This is the current apply and update path. It is the repeated download this change removes.

## Decided

- Cache path: `<config>/tewyn/skills/<name>/`.
- Project path: `.tewyn/skills/<name>/`, a real copy.
- `-u` reads directory names under `.tewyn/skills/`. It loads URLs and cached commits from the library.
- When the cached commit equals the latest commit for that URL, `-u` copies the cached folder and skips the download.
- The project folder is replaced only after the cache has the new files and the copy has succeeded.
- Stdout for `-u` is progress lines, then a `warnings` section when any skill was skipped. Example:

```
downloading brainstorming
copying brainstorming
copying ponytail
warnings
  writing-plans  not in library
  systematic-debugging  invalid url
```

- Agent symlink behavior is unchanged.

## Left to the builder

- The exact progress words, as long as download and copy are visible as they happen and warnings come last.
- How the copy is made, and how a project folder is swapped in only after the copy succeeds.
- Warning text for a bad URL or a failed download, using the errors the GitHub reader already returns where those fit.
- Usage text, so `-u` replaces `--update-skills`.
