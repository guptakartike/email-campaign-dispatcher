package loader

import (
	"encoding/csv"
	"fmt"
	"os"
	"strings"

	"github.com/guptakartike/email-dispatcher/internal/models"
)

// LoadRecipients reads a CSV file and returns a slice of Recipient structs.
// The CSV file must have a header row. Each data row must contain at least
// two columns: email (index 0) and name (index 1).
func LoadRecipients(filename string) ([]models.Recipient, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to open file %q: %w", filename, err)
	}
	defer file.Close()

	reader := csv.NewReader(file)

	// Read the header row.
	header, err := reader.Read()
	if err != nil {
		return nil, fmt.Errorf("failed to read CSV header: %w", err)
	}
	if len(header) < 2 {
		return nil, fmt.Errorf("CSV header must have at least 2 columns, got %d", len(header))
	}

	// Determine column indices based on header names.
	emailIdx := -1
	nameIdx := -1
	for idx, col := range header {
		clean := strings.ToLower(strings.TrimSpace(col))
		if clean == "email" {
			emailIdx = idx
		} else if clean == "name" {
			nameIdx = idx
		}
	}
	if emailIdx == -1 || nameIdx == -1 {
		emailIdx = 0
		nameIdx = 1
	}

	// Read all remaining rows.
	rows, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("failed to read CSV data: %w", err)
	}

	var recipients []models.Recipient
	for i, row := range rows {
		if len(row) < 2 {
			return nil, fmt.Errorf("row %d: expected at least 2 columns, got %d", i+2, len(row))
		}
		recipients = append(recipients, models.Recipient{
			Email: strings.TrimSpace(row[emailIdx]),
			Name:  strings.TrimSpace(row[nameIdx]),
		})
	}

	return recipients, nil
}
