package cmd

import (
	"log"
	"os"

	"github.com/pgm/goconseq/run"

	"github.com/spf13/cobra"
)

var (
	dotCmd = &cobra.Command{
		Use:   "dot",
		Short: "export dot graph",
		Run: func(cmd *cobra.Command, args []string) {
			graph, _, _, err := run.ReplayAndExport(stateDir, args[0])
			if err != nil {
				log.Fatalf("%s", err)
			}
			if graph == nil {
				panic("nil graph")
			}
			graph.PrintGraph(os.Stdout)
		},
	}
)

func init() {
	rootCmd.AddCommand(dotCmd)
}
