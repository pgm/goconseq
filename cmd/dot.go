package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	dotCmd = &cobra.Command{
		Use:   "dot",
		Short: "export dot graph",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("dot called")
			// graph, artifacts, applications, err := run.ReplayAndExport(stateDir, args[0])
			// if err != nil {
			// 	log.Fatalf("%s", err)
			// }
			panic("unfinished")
		},
	}
)

func init() {
	rootCmd.AddCommand(dotCmd)
}
