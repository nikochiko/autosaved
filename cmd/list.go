package cmd

import (
	"strconv"

	"github.com/nikochiko/autosaved/core"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list [n]",
	Short: "Save the current state of a repository",
	Long:  `Gets a list of the autosaves made by `,
	Args:  cobra.MaximumNArgs(1),
	Run:   list,
}

func list(cmd *cobra.Command, args []string) {
	var err error

	limit := 10
	if len(args) > 0 {
		limitString := args[0]
		limit, err = strconv.Atoi(limitString)
		if err != nil {
			asdFmt.Warnf("error while reading argument n. defaulting to 10: %v\n", err)
			limit = 10
		}
	}

	path := "."

	asdRepo, err := core.AsdRepoFromGitRepoPath(path, getMinSeconds())
	checkError(err)

	err = asdRepo.List(limit)
	checkError(err)
}
