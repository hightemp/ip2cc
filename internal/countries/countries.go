// Package countries provides ISO-3166 country code and name mappings.
package countries

import (
	"bufio"
	_ "embed"
	"strings"
	"sync"
)

//go:embed iso3166.txt
var iso3166Data string

var (
	codeToName map[string]string
	codes      []string
	once       sync.Once
)

func init() {
	loadData()
}

func loadData() {
	once.Do(func() {
		codeToName = make(map[string]string)
		codes = make([]string, 0, 256)

		scanner := bufio.NewScanner(strings.NewReader(iso3166Data))
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			parts := strings.SplitN(line, ",", 2)
			if len(parts) != 2 {
				continue
			}
			code := strings.ToUpper(strings.TrimSpace(parts[0]))
			name := strings.TrimSpace(parts[1])
			codeToName[code] = name
			codes = append(codes, code)
		}
	})
}

// GetName returns the country name for the given ISO-3166 alpha-2 code.
// Returns empty string if not found.
func GetName(code string) string {
	return codeToName[strings.ToUpper(code)]
}

// IsValid checks if the given code is a valid ISO-3166 alpha-2 code.
func IsValid(code string) bool {
	_, ok := codeToName[strings.ToUpper(code)]
	return ok
}

// AllCodes returns all ISO-3166 alpha-2 codes (uppercase).
func AllCodes() []string {
	result := make([]string, len(codes))
	copy(result, codes)
	return result
}

// AllCodesLower returns all ISO-3166 alpha-2 codes in lowercase.
func AllCodesLower() []string {
	result := make([]string, len(codes))
	for i, c := range codes {
		result[i] = strings.ToLower(c)
	}
	return result
}

// Count returns the number of countries.
func Count() int {
	return len(codes)
}

// LoadFromFile loads country codes from a file (one code per line).
// Returns codes in lowercase.
func LoadFromFile(content string) ([]string, error) {
	var result []string
	scanner := bufio.NewScanner(strings.NewReader(content))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		code := strings.ToLower(line)
		if len(code) == 2 {
			result = append(result, code)
		}
	}
	return result, scanner.Err()
}
