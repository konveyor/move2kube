package cmd

import (
	"fmt"

	api "github.com/konveyor/move2kube-wasm/lib"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// GetVersionCommand returns a command to print the version
func GetVersionCommand() *cobra.Command {
	viper.AutomaticEnv()

	long := false
	versionCmd := &cobra.Command{
		Use:   "version",
		Short: "Print the version information",
		Long:  "Print the version information",
		Run:   func(*cobra.Command, []string) { fmt.Println(api.GetVersion(long)) },
	}

	versionCmd.Flags().BoolVarP(&long, "long", "l", false, "Print the version details.")

	return versionCmd
}
