package cmd

import (
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path"

	"github.com/pgm/goconseq/run"

	"github.com/spf13/cobra"
)

var (
	dotCmd = &cobra.Command{
		Use:   "dot",
		Short: "export dot graph",
		Run: func(cmd *cobra.Command, args []string) {
			graph, db, err := run.ReplayAndExport(stateDir, args[0])
			if err != nil {
				log.Fatalf("%s", err)
			}
			db.Close()

			dirName, err := ioutil.TempDir("", "dot")
			if err != nil {
				panic(err)
			}

			if graph == nil {
				panic("nil graph")
			}

			dotFile := path.Join(dirName, "t.dot")
			pngName := path.Join(dirName, "dot.png")
			f, err := os.Create(dotFile)
			if err != nil {
				panic(err)
			}

			graph.PrintGraph(f)
			f.Close()

			log.Printf("args %s %s %s %s", dotFile, "-Tpng", "-o", pngName)
			err = exec.Command("dot", dotFile, "-Tpng", "-o", pngName).Run()
			if err != nil {
				panic(err)
			}

			err = exec.Command("open", pngName).Run()
			if err != nil {
				panic(err)
			}
		},
	}
)

func init() {
	rootCmd.AddCommand(dotCmd)
}
