package cmd

import (
	"path/filepath"

	"github.com/spf13/cobra"
)

var unwatchCmd = &cobra.Command{
	Use:   "unwatch [path]",
	Short: "Unwatch the given directory (or by default, the current directory)",
	Long: `Removes the given directory (defaults to current directory)
from the list of directories being watched by autosaved`,
	Args: cobra.MaximumNArgs(1),
	Run:  unwatch,
}

func unwatch(cmd *cobra.Command, args []string) {
	repositories := globalViper.GetStringSlice("repositories")

	path := "."
	if len(args) == 1 {
		path = args[0]
	}

	if !filepath.IsAbs(path) {
		var err error
		path, err = filepath.Abs(path)
		checkError(err)
	}

	if !contains(repositories, path) {
		asdFmt.Errorf("The repo you want to unwatch is not being watched in the first place\n")
		return
	}

	for i, watched := range repositories {
		if path == watched {
			repositories = append(repositories[:i], repositories[i+1:]...)
		}
	}

	globalViper.Set("repositories", repositories)
	err := globalViper.WriteConfig()
	checkError(err)

	asdFmt.Successf("Repo unwatched from autosaved\n")
}
