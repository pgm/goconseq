package adhoc

import (
	"fmt"
	"strings"
)

func PrintGroupedBrief(pairsList []KVPairs, groupBy string) {
	grouped := GroupBy(pairsList, groupBy)
	for groupKey, pairs := range grouped {
		fmt.Printf("For %s=%s:\n", groupBy, groupKey)
		PrintBrief(SelectWithout(pairs, groupBy))
		fmt.Printf("\n")
	}
}

func PrintBrief(pairsList []KVPairs) {
	sharedFields, shared, distinctFields, distinct := SplitIntoSharedAndDistinct(pairsList)
	if len(sharedFields) > 0 {
		fmt.Printf("  Properties shared by all %d rows:\n", len(distinct))
		PrintTable([]KVPairs{shared}, sharedFields, "    ")
	}
	PrintTable(distinct, distinctFields, "  ")
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

func PrintTable(pairsList []KVPairs, fields []string, padding string) {
	rows := FlattenValuesToRows(pairsList, fields)
	rowsWithHeader := make([][]string, len(rows))
	copy(rowsWithHeader, rows)
	rowsWithHeader = append(rowsWithHeader, fields)
	colWidths := ComputeColumnWidths(rowsWithHeader)

	colFormats := make([]string, len(colWidths))
	for column, width := range colWidths {
		colFormats[column] = fmt.Sprintf("%% %ds  ", width)
	}

	fmt.Printf("%s", padding)
	for column, field := range fields {
		fmt.Printf(colFormats[column], field)
	}
	fmt.Printf("\n")

	fmt.Printf("%s", padding)
	for column := range fields {
		fmt.Printf("%s  ", strings.Repeat("-", colWidths[column]))
	}
	fmt.Printf("\n")

	for _, row := range rows {
		fmt.Printf("%s", padding)
		for column, value := range row {
			fmt.Printf(colFormats[column], value)
		}
		fmt.Printf("\n")
	}
}
