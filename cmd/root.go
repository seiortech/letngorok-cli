package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:     "ngorok-cli",
	Short:   "Make your local website exposed to the internet with only one command for free!",
	Long:    "Letngorok is a service that allows you to make your local website exposed to the internet with only one command for free!",
	Version: "v0.0.1",
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
}
