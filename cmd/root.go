package cmd

import "github.com/spf13/cobra"

var rootCmd = &cobra.Command{
	Use:   "datassert",
	Short: "Placeholder",
	Long:  "Placeholder",
}

func Execute() {
	rootCmd.Execute()
}
