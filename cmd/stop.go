package cmd

import (
	"os"

	"github.com/nikochiko/autosaved/daemon"
	"github.com/spf13/cobra"
)

var stopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop the autosave-daemon",
	Long: `Attempts to stop autosave-daemon.
Will error if there is it isn't running already`,
	Args: cobra.NoArgs,
	Run:  stop,
}

func stop(cmd *cobra.Command, args []string) {
	minSeconds := getMinSeconds()

	d, err := daemon.New(globalViper, lockfilePath, os.Stdout, os.Stderr, minSeconds)
	checkError(err)

	asdFmt.Successf("Stopping autosave daemon\n")
	err = d.Stop()
	if err != nil {
		checkError(err)
	}
	asdFmt.Successf("Stopped daemon successfully\n")
}
