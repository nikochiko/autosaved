package cmd

import (
	"git.kausm.in/kaustubh/autosaved/core"
	"github.com/go-git/go-git/v5"
	"github.com/spf13/cobra"
)

const defaultCommitMessage = "manual save"

var saveCmd = &cobra.Command{
	Use:   "save [-m commit-msg] [path-to-repo]",
	Short: "Save the current state of a repository",
	Long: `Saves the current state of a repository regardless
of how long ago the last save was done or how many new
characters were written.`,
	Args: cobra.MaximumNArgs(1),
	Run:  save,
}

func save(cmd *cobra.Command, args []string) {
	msg, err := cmd.Flags().GetString("message")
	if err != nil {
		asdFmt.Warnf("error while getting message flag: %v. Continuing with the default message\n", err)
		msg = defaultCommitMessage
	}

	repoPath := "."
	if len(args) > 0 {
		repoPath = args[0]
	}

	gitRepo, err := git.PlainOpen(repoPath)
	if err != nil {
		asdFmt.Errorf("Couldn't access Git repository: %v\n", err)
		checkError(err) // exits with 1 exit code if err != nil
	}

	asdRepo := core.AsdRepository{
		Repository: gitRepo,
	}

	err = asdRepo.Save(msg)
	checkError(err)

	asdFmt.Successf("Saved successfully\n")
}
