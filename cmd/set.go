package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func init() {
	var setCmd = &cobra.Command{
		Use:   "set",
		Short: "Set configuration options",
	}
	rootCmd.AddCommand(setCmd)

	var tokenCmd = &cobra.Command{
		Use:   "token [token]",
		Short: "Set the Letngorok authentication token",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			token := args[0]
			if err := SaveToken(token); err != nil {
				fmt.Printf("Error saving token: %v\n", err)
				return
			}
			fmt.Println("Token saved successfully!")
		},
	}
	setCmd.AddCommand(tokenCmd)
}
