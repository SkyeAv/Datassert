package cmd

import (
	"sync"

	"github.com/spf13/cobra"
)

type ClassData struct {
	mu   sync.Mutex
	data map[string][]string
}

func (cd *ClassData) Set(key string, value []string) {
	cd.mu.Lock()
	cd.data[key] = value
	cd.mu.Unlock()
}

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
