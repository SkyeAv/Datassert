package cmd

import "github.com/spf13/cobra"

var rootCmd = &cobra.Command{
	Use:   "datassert",
	Short: "Build a DuckDB assertion store from Babel exports",
	Long:  "Datassert is a high-performance CLI that processes Babel export files and builds a local DuckDB-backed assertion database.",
}

func Execute() {
	rootCmd.Execute()
}
