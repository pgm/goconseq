package adhoc

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

func PrintGroupedBrief(w io.Writer, pairsList []KVPairs, groupBy string) error {
	grouped := GroupBy(pairsList, groupBy)
	for groupKey, pairs := range grouped {
		_, err := fmt.Fprintf(w, "For %s=%s:\n", groupBy, groupKey)
		if err == nil {
			err = PrintBrief(w, SelectWithout(pairs, groupBy))
		}
		if err == nil {
			_, err = fmt.Fprintf(w, "\n")
		}
		if err != nil {
			return err
		}
	}
	return nil
}

func PrintJSON(w io.Writer, pairsList []KVPairs) error {
	b, err := json.Marshal(pairsList)
	if err != nil {
		return err
	}
	w.Write(b)
	return nil
}

func PrintCSV(w io.Writer, pairsList []KVPairs, includeHeader bool) error {
	keys := UniqueKeys(pairsList)
	if includeHeader {
		printCSVRow(w, keys)
	}

	for _, row := range pairsList {
		rowStr := make([]string, len(keys))
		for i, key := range keys {
			rowStr[i] = row[key]
		}
		printCSVRow(w, rowStr)
	}
	return nil
}

func printCSVRow(w io.Writer, row []string) {
	for i, value := range row {
		if i > 0 {
			fmt.Fprintf(w, ",")
		}
		printCSVValue(w, value)
	}
	fmt.Fprintf(w, "\n")
}

func printCSVValue(w io.Writer, value string) (int, error) {
	if strings.ContainsAny(value, "\\\"\n\r") {
		value = csvEscape(value)
	}
	return w.Write([]byte(value))
}

func csvEscape(s string) string {
	var b strings.Builder
	for _, c := range s {
		if c == '\n' {
			b.WriteString("\\n")
		} else if c == '\\' {
			b.WriteString("\\\\")
		} else if c == '"' {
			b.WriteString("\\\"")
		} else if c == '\r' {
			b.WriteString("\\r")
		} else {
			b.WriteString(string(c))
		}
	}
	return b.String()
}

func UniqueKeys(pairsList []KVPairs) []string {
	sharedFields, _, distinctFields, _ := SplitIntoSharedAndDistinct(pairsList)
	return append(sharedFields, distinctFields...)
}

func PrintBrief(w io.Writer, pairsList []KVPairs) error {
	sharedFields, shared, distinctFields, distinct := SplitIntoSharedAndDistinct(pairsList)
	if len(sharedFields) > 0 {
		fmt.Fprintf(w, "  Properties shared by all %d rows:\n", len(distinct))
		PrintTable(w, []KVPairs{shared}, sharedFields, "    ")
	}
	return PrintTable(w, distinct, distinctFields, "  ")
}

func ComputeColumnWidths(rows [][]string) []int {
	maxWidths := make([]int, len(rows[0]))
	for _, row := range rows {
		for column, value := range row {
			valLen := len(value)
			if valLen > maxWidths[column] {
				maxWidths[column] = valLen
			}
		}
	}
	return maxWidths
}

func FlattenValuesToRows(pairsList []KVPairs, fields []string) [][]string {
	result := make([][]string, len(pairsList))
	for row, pairs := range pairsList {
		result[row] = make([]string, len(fields))
		for column, field := range fields {
			result[row][column] = pairs[field]
		}
	}
	return result
}

func PrintTable(w io.Writer, pairsList []KVPairs, fields []string, padding string) error {
	rows := FlattenValuesToRows(pairsList, fields)
	rowsWithHeader := make([][]string, len(rows))
	copy(rowsWithHeader, rows)
	rowsWithHeader = append(rowsWithHeader, fields)
	colWidths := ComputeColumnWidths(rowsWithHeader)

	colFormats := make([]string, len(colWidths))
	for column, width := range colWidths {
		colFormats[column] = fmt.Sprintf("%% %ds  ", width)
	}

	fmt.Fprintf(w, "%s", padding)
	for column, field := range fields {
		fmt.Fprintf(w, colFormats[column], field)
	}
	fmt.Fprintf(w, "\n")

	fmt.Fprintf(w, "%s", padding)
	for column := range fields {
		fmt.Fprintf(w, "%s  ", strings.Repeat("-", colWidths[column]))
	}
	fmt.Fprintf(w, "\n")

	for _, row := range rows {
		fmt.Fprintf(w, "%s", padding)
		for column, value := range row {
			fmt.Fprintf(w, colFormats[column], value)
		}
		fmt.Fprintf(w, "\n")
	}
	return nil
}
