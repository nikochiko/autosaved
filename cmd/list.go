package cmd

import (
	"strconv"

	"github.com/nikochiko/autosaved/core"
	"github.com/spf13/cobra"
)

const defaultAutosaves = 5

var listCmd = &cobra.Command{
	Use:   "list [--autosaves N] [n]",
	Short: "Lists the last n (default: 10) commits and the related saves",
	Long: `Gets a list of the commits made by the user starting from HEAD,
along with the related autosaves / manual saves done using autosaved. This
format helps in identifying the relevant autosaves and in restoring to one.

--autosaves: number specifying how many max autosaves to show per commit (default: 5)`,
	Args: cobra.MaximumNArgs(1),
	Run:  list,
}

func list(cmd *cobra.Command, args []string) {
	autosaves, err := cmd.Flags().GetInt("autosaves")
	if err != nil {
		asdFmt.Warnf("error while getting autosaves flag: %v. Continuing with the default (%d)\n", err, defaultAutosaves)
		autosaves = defaultAutosaves
	}

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

	err = asdRepo.List(limit, autosaves)
	checkError(err)
}
