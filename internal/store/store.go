package store

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/gocarina/gocsv"
	"golang.org/x/text/encoding/charmap"
)

func init() {
	gocsv.SetCSVReader(func(in io.Reader) gocsv.CSVReader {
		r := csv.NewReader(in)
		r.LazyQuotes = true
		r.Comma = ';'
		r.FieldsPerRecord = -1
		return &trimmingReader{r}
	})
}

// readCSV loads a Windows-1252 encoded CSV file into target
func readCSV[T any](fileName string, target *[]*T) error {
	path, err := findFile(fileName)
	if err != nil {
		return err
	}
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer func(f *os.File) {
		_ = f.Close()
	}(f)

	decoder := charmap.Windows1252.NewDecoder()
	decodedReader := decoder.Reader(f)

	return gocsv.Unmarshal(decodedReader, target)
}

// fileExists checks if a data file exists (with any valid extension)
func fileExists(fileName string) bool {
	_, err := findFile(fileName)
	return err == nil
}

// findFile looks for a data file with any of the valid extensions
func findFile(fileName string) (string, error) {
	extensions := []string{".txt", ".TXT", ".csv", ".CSV"}
	for _, ext := range extensions {
		path := fileName + ext
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}
	return "", fmt.Errorf("no data file found for %s", fileName)
}

// trimmingReader wraps a csv.Reader and removes surrounding quotes and whitespace from each field
type trimmingReader struct {
	*csv.Reader
}

// Read removes surrounding quotes and whitespace from each field
func (tr *trimmingReader) Read() ([]string, error) {
	record, err := tr.Reader.Read()
	if err != nil {
		return record, err
	}
	for i := range record {
		normalize(&record[i])
	}
	return record, nil
}

// ReadAll removes surrounding quotes and whitespace from each field
func (tr *trimmingReader) ReadAll() ([][]string, error) {
	records, err := tr.Reader.ReadAll()
	if err != nil {
		return records, err
	}
	for i := range records {
		for j := range records[i] {
			normalize(&records[i][j])
		}
	}
	return records, nil
}

// normalize removes all quotes and surrounding whitespace
func normalize(s *string) {
	*s = strings.TrimSpace(strings.Trim(strings.TrimSpace(*s), "\""))
}

// loadMap loads a CSV file into a map using the provided key function to determine the map keys.
func loadMap[T any, K comparable](fileName string, dest map[K]*T, keyFunc func(*T) K) error {
	var items []*T
	if err := readCSV(fileName, &items); err != nil {
		return err
	}
	clear(dest)
	for _, item := range items {
		dest[keyFunc(item)] = item
	}
	return nil
}
