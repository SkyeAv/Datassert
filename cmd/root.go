package cmd

import "github.com/spf13/cobra"

var rootCmd = &cobra.Command{
	Use:   "datassert",
	Short: "Placeholder",
	Long:  "Placeholder",
}

func Execute() {
	rootCmd.Execute()
	rootCmd.GenBashCompletionFile("completions.bash")
	rootCmd.GenZshCompletionFile("completions.zsh")
	rootCmd.GenFishCompletionFile("completions.fish")
	rootCmd.GenPowerShellCompletionFile("completions.ps1")
}
