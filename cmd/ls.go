package cmd

import (
	"fmt"
	"io"
	"log"
	"os"
	"regexp"
	"strings"

	"github.com/pgm/goconseq/adhoc"
	"github.com/pgm/goconseq/run"
	"github.com/spf13/cobra"
)

var filterExp = regexp.MustCompile("([^=]+)=(.*)")
var groupBy string
var format string
var selectFields string
var outputFilename string

func parseQuery(filters []string) (map[string]string, error) {
	query := make(map[string]string)
	for _, filter := range filters {
		parts := filterExp.FindStringSubmatch(filter)
		if parts == nil {
			return nil, fmt.Errorf("Could not parse \"%s\" as a filter", filter)
		}
		query[parts[1]] = parts[2]
	}
	return query, nil
}

func parseFields(fields string) []string {
	return strings.Split(fields, ",")
}

var (
	lsCmd = &cobra.Command{
		Use:   "ls conseqfile [filter1] [filter2] ... ",
		Short: "List artifacts",
		Args:  cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			conseqFile := args[0]
			query, err := parseQuery(args[1:])

			if err != nil {
				log.Fatal(err)
			}

			_, db, err := run.ReplayAndExport(stateDir, conseqFile)
			if err != nil {
				log.Fatal(err)
			}
			defer db.Close()

			artifacts := db.FindArtifacts(query)

			asKV := make([]adhoc.KVPairs, len(artifacts))
			for i, artifacts := range artifacts {
				asKV[i] = artifacts.Properties.ToStrMap(func(fileID int) string {
					file := db.GetFile(fileID)
					return file.LocalPath
				})
			}

			var out io.Writer
			if outputFilename == "" {
				out = os.Stdout
			} else {
				f, err := os.Create(outputFilename)
				if err != nil {
					log.Fatal(err)
				}
				out = f
				defer f.Close()
			}

			if selectFields != "" {
				fields := parseFields(selectFields)
				asKV = adhoc.Select(asKV, fields)

				if format == "" {
					format = "csv-no-head"
				}
			}

			if format == "" {
				format = "brief"
			}

			if format == "brief" {
				err = adhoc.PrintGroupedBrief(out, asKV, groupBy)
			} else if format == "csv" {
				err = adhoc.PrintCSV(out, asKV, true)
			} else if format == "csv-no-head" {
				err = adhoc.PrintCSV(out, asKV, false)
			} else if format == "json" {
				err = adhoc.PrintJSON(out, asKV)
			} else {
				log.Fatalf("Invalid format")
			}

			if err != nil {
				log.Fatal(err)
			}
		},
	}
)

func init() {
	rootCmd.AddCommand(lsCmd)
	lsCmd.Flags().StringVarP(&outputFilename, "output", "o", "", "If set, the file to write to")
	lsCmd.Flags().StringVarP(&format, "format", "t", "", "The output format (brief, json or csv)")
	lsCmd.Flags().StringVarP(&groupBy, "groupby", "g", "type", "The field to group by when using the 'brief' format (defaults to 'type')")
	lsCmd.Flags().StringVarP(&selectFields, "select", "s", "", "The fields to select (defaults to all)")
}
