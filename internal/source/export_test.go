package source

// SetAfterGit runs fn after each git command. Tests cancel a read between commands.
func SetAfterGit(fn func([]string)) { afterGit = fn }
