package cmd

import "github.com/spf13/cobra"

func build(cmd *cobra.Command, args []string) {
	synonymDir := args[0]
	classDir := args[1]
}

var buildCmd = &cobra.Command{
	Use:   "build",
	Short: "Placeholder",
	Long:  "Placeholder",
	Run:   build,
}

func init() {
	rootCmd.AddCommand(buildCmd)
}
