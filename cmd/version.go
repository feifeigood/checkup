package cmd

import (
	"fmt"
	"os"
	"runtime"

	"github.com/spf13/cobra"
)

// Build information
var (
	app     = "checkup"
	Version string
	GitSHA  string

	versionCmd = &cobra.Command{
		Use:   "version",
		Short: "Print the version number of Checkup",
		Long:  `All software has versions. This is Checkup`,
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("%s Version: %s\n", app, Version)
			fmt.Printf("Git SHA: %s\n", GitSHA)
			fmt.Printf("Go Version: %s\n", runtime.Version())
			fmt.Printf("Go OS/Arch: %s/%s\n", runtime.GOOS, runtime.GOARCH)
			os.Exit(0)
		},
	}
)

func init() {
	rootCmd.AddCommand(versionCmd)
}
