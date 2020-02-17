package cmd

import (
	"fmt"
	"log"

	"github.com/pgm/goconseq/adhoc"
	"github.com/pgm/goconseq/run"
	"github.com/spf13/cobra"
)

var (
	lsCmd = &cobra.Command{
		Use:   "ls",
		Short: "List artifacts",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("ls called")
			_, artifacts, _, err := run.ReplayAndExport(stateDir, args[0])
			asKV := make([]adhoc.KVPairs, len(artifacts))
			for i, artifacts := range artifacts {
				asKV[i] = artifacts.Properties.ToStrMap()
			}

			adhoc.PrintGroupedBrief(asKV, "type")

			if err != nil {
				log.Fatalf("%s", err)
			}
		},
	}
)

func init() {
	rootCmd.AddCommand(lsCmd)
}
