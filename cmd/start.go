package cmd

import (
	"os"

	"github.com/nikochiko/autosaved/daemon"
	"github.com/spf13/cobra"
)

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the autosave-daemon",
	Long: `Attempts to start autosave-daemon.
Will error if there is already a daemon running`,
	Args: cobra.NoArgs,
	Run:  start,
}

func start(cmd *cobra.Command, args []string) {
	minSeconds := getMinSeconds()

	asdFmt.Successf("Initialising autosave daemon\n")
	d, err := daemon.New(globalViper, lockfilePath, os.Stdout, os.Stderr, minSeconds)
	checkError(err)

	asdFmt.Successf("Starting autosave daemon\n")
	err = d.Start()
	if err != nil {
		checkError(err)
	}
}
