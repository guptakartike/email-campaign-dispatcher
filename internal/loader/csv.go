package loader

import (
	"encoding/csv"
	"os"
)

type Recipient struct {
	Email string
	Name  string
}

func LoadRecipient(filepath string) ([]Recipient, error) {
	file, err := os.Open(filepath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	reader := csv.NewReader(file)
	data, err := reader.ReadAll()
	if err != nil {
		return nil, err
	}

	var recipients []Recipient
	for i, row := range data {
		if i == 0 {
			continue
		}

		recipients = append(recipients, Recipient{
			Email: row[0],
			Name:  row[1],
		})
	}
	return recipients, nil
}
