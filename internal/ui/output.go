package ui

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

func PrintSuccess(msg string) {
	fmt.Println(Success.Render("✓") + " " + msg)
}

func PrintError(msg string) {
	fmt.Fprintln(os.Stderr, Error.Render("✗")+" "+msg)
}

func PrintWarning(msg string) {
	fmt.Println(Warning.Render("!")+" " + msg)
}

func PrintInfo(msg string) {
	fmt.Println(InfoStyle.Render("→") + " " + msg)
}

func PrintJSON(v interface{}) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

func PrintTable(headers []string, rows [][]string) {
	if len(rows) == 0 {
		return
	}

	// Calculate column widths
	widths := make([]int, len(headers))
	for i, h := range headers {
		widths[i] = len(h)
	}
	for _, row := range rows {
		for i, cell := range row {
			if i < len(widths) && len(cell) > widths[i] {
				widths[i] = len(cell)
			}
		}
	}

	// Print header
	headerParts := make([]string, len(headers))
	for i, h := range headers {
		headerParts[i] = TableHeader.Render(fmt.Sprintf("%-*s", widths[i], h))
	}
	fmt.Println(strings.Join(headerParts, "  "))

	// Print separator
	sepParts := make([]string, len(headers))
	for i := range headers {
		sepParts[i] = DimStyle.Render(strings.Repeat("─", widths[i]))
	}
	fmt.Println(strings.Join(sepParts, "  "))

	// Print rows
	for _, row := range rows {
		parts := make([]string, len(headers))
		for i := range headers {
			cell := ""
			if i < len(row) {
				cell = row[i]
			}
			parts[i] = TableCell.Render(fmt.Sprintf("%-*s", widths[i], cell))
		}
		fmt.Println(strings.Join(parts, ""))
	}
}
