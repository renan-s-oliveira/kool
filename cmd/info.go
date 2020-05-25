package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(infoCmd)
}

var infoCmd = &cobra.Command{
	Use:   "info",
	Short: "Prints out information about fwd setup (like environment variables)",
	Long:  `Prints out information about fwd setup (like environment variables)`,
	Run:   runInfo,
}

func runInfo(cmf *cobra.Command, args []string) {
	for _, envVar := range os.Environ() {
		if strings.Contains(envVar, "FWD_") {
			fmt.Println(envVar)
		}
	}
}