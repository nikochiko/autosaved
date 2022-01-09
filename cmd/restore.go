package cmd

import (
	"git.kausm.in/kaustubh/autosaved/core"
	"github.com/spf13/cobra"
)

var restoreCmd = &cobra.Command{
	Use:   "restore commit-hash",
	Short: "Restores the state of a repository to a previous checkpoint (commit). Input is commit hash that we want to restore",
	Long: `Restores the state of the repository to a previous state,
whose commit Hash is given as an argument`,
	Args: cobra.ExactArgs(1),
	Run:  restore,
}

func restore(cmd *cobra.Command, args []string) {
	repoPath := "."
	asdRepo, err := core.AsdRepoFromGitRepoPath(repoPath, getMinSeconds())
	checkError(err)

	hashString := args[0]

	err = asdRepo.RestoreByCommitHash(hashString)
	checkError(err)

	asdFmt.Successf("Restored successfully\n")
}
