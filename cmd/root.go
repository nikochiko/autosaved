package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	// used for flags
	cfgFile      string
	lockfilePath string
	afterMinutes int
	afterSeconds int
)

var globalViper = viper.GetViper()

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "autosaved",
	Short: "Never lose your work. Code without worrying",
	Long: `autosaved, pronounced autosave-d (for autosave daemon) is a utility written in Go to autosave progress on code projects.

It uses the go-git package to save snapshots without interfering the normal Git flow - branches that are to be pushed upstream, HEAD, or the Git index.

It provides an interface called autosaved (short for autosaved interface), which can be used to interact with the daemon.`,
	// Uncomment the following line if your bare application
	// has an action associated with it:
	// Run: func(cmd *cobra.Command, args []string) { },
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.autosaved.yaml)")
	rootCmd.PersistentFlags().IntVar(&afterMinutes, "after_minutes", 2, "Minutes to wait, at a minimum, before making the next autosave. after_every.seconds gets added in this")
	rootCmd.PersistentFlags().IntVar(&afterSeconds, "after_seconds", 0, "Seconds to wait, at a minimum, before making the next autosave. after_every.minutes gets added in this")
	rootCmd.PersistentFlags().StringVar(&lockfilePath, "lockfilePath", "", "Lockfile for the daemon")

	viper.BindPFlag("after_every.minutes", rootCmd.PersistentFlags().Lookup("after_minutes"))
	viper.BindPFlag("after_every.seconds", rootCmd.PersistentFlags().Lookup("after_seconds"))

	rootCmd.AddCommand(saveCmd)
	saveCmd.Flags().StringP("message", "m", "manual save", "commit message")

	rootCmd.AddCommand(startCmd)

	rootCmd.AddCommand(stopCmd)

	rootCmd.AddCommand(watchCmd)

	rootCmd.AddCommand(unwatchCmd)

	rootCmd.AddCommand(listCmd)
	listCmd.Flags().Int("autosaves", defaultAutosaves, "maximum number of autosaves to display per commit")

	rootCmd.AddCommand(restoreCmd)
}

// get one of available config path from environment variable, $HOME/.config, $HOME
func getConfigHomePath() string {
	if xdg_home := os.Getenv("XDG_CONFIG_HOME"); xdg_home != "" {
		return xdg_home
	}

	userHome, err := os.UserHomeDir()
	cobra.CheckErr(err)

	// default XDG_CONFIG_HOME "$HOME/.config"
	configHome := filepath.Join(userHome, ".config")
	// check directory exist
	if _, err := os.Stat(configHome); err == nil {
		return configHome
	}

	// default path: user's home
	return userHome
}

// compatible previous version
// migrate old config path into new path
func moveOldConfigToNewDir(newConfigDir string) {
	userHome, err := os.UserHomeDir()
	cobra.CheckErr(err)

	oldConfigPath := filepath.Join(userHome, ".autosaved.yaml")
	// check old config exist or not, skip if it not exist
	if _, err := os.Stat(oldConfigPath); os.IsNotExist(err) {
		return
	}

	newConfigPath := filepath.Join(newConfigDir, ".autosaved.yaml")

	if oldConfigPath == newConfigPath {
		return
	}

	// check new config exist or not, skip if it exist
	if _, err := os.Stat(newConfigPath); os.IsExist(err) {
		return
	}

	// mv old config to new config path
	err = os.Rename(oldConfigPath, newConfigPath)
	cobra.CheckErr(err)
}

func initConfig() {
	home := getConfigHomePath()
	moveOldConfigToNewDir(home)

	if cfgFile != "" {
		// set config file from flag
		viper.SetConfigFile(cfgFile)
	} else {
		// Search config in home directory with name ".autosaved.yaml"
		viper.AddConfigPath(home)
		viper.SetConfigType("yaml")
		viper.SetConfigName(".autosaved")
	}

	if lockfilePath == "" {
		lockfilePath = filepath.Join(home, ".autosaved.lock")
	}

	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	} else if reflect.TypeOf(err) == reflect.TypeOf(viper.ConfigFileNotFoundError{}) {
		fmt.Println("Config file not found. Writing config to file now")
		viper.SafeWriteConfig()
	}
}

func getMinSeconds() int {
	return 60*afterMinutes + afterSeconds
}
