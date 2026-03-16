package cmd

import (
	"log"
	"path/filepath"
	"sync"

	"github.com/spf13/cobra"
)

func throwError(code uint8, err error) {
	log.Fatal("%d | ERROR | %v", code, err)
}

func globFileNames(dir string) []string {
	abs, err := filepath.Abs(dir)
	if err != nil {
		throwError(1, err)
	}

	pattern := filepath.Join(abs, "*Class.ndjson.zst")
	fileNames, err := filepath.Glob(pattern)
	if err != nil {
		throwError(2, err)
	}

	return fileNames
}

type ClassLookup struct {
	mu   sync.Mutex
	data map[string][]string
}

func (cl *ClassLookup) Set(key string, value []string) {
	cl.mu.Lock()
	cl.data[key] = value
	cl.mu.Unlock()
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
