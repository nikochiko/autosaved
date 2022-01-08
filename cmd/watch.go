package cmd

import (
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var watchCmd = &cobra.Command{
	Use:   "watch [path]",
	Short: "Start watching this directory for autosaving",
	Long: `Adds the given directory (defaults to current directory)
to the list of directories being watched by autosaved`,
	Args: cobra.MaximumNArgs(1),
	Run:  watch,
}

func watch(cmd *cobra.Command, args []string) {
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

	if _, err := os.Stat(filepath.Join(path, ".git")); err != nil {
		asdFmt.Errorf("Path (or current directory) should have a Git repository\n")
		checkError(err)
	}

	if contains(repositories, path) {
		asdFmt.Errorf("The repo you want to add is already being watched!\n")
		return
	}

	repositories = append(repositories, path)
	globalViper.Set("repositories", repositories)
	err := globalViper.WriteConfig()
	checkError(err)

	asdFmt.Successf("Repo added to autosaved\n")
}

func contains(slice []string, value string) bool {
	for _, v := range slice {
		if v == value {
			return true
		}
	}

	return false
}
