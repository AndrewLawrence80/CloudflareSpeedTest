package domain

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"

	"github.com/AndrewLawrence80/CloudflareSpeedTest/pkg/common"
	"github.com/AndrewLawrence80/CloudflareSpeedTest/pkg/log"
)

// LoadAllDomains reads every file under the domain-list-community/data
// directory and returns a slice of domain names.
//
// Each data file may contain:
//   - blank lines and comment lines beginning with '#'  – skipped
//   - "include:<name>" lines that reference another file – skipped (all files
//     are already visited by the directory walk)
//   - "regexp:<pattern>" lines                          – skipped (not a plain domain)
//   - "domain:<name>" or "full:<name>" prefixed entries – prefix is stripped
//   - bare domain names (no prefix)
func LoadAllDomains() ([]string, error) {
	dataDir := common.EnvOr("DOMAIN_LIST_PATH", "domain-list-community/data")

	domains := make(map[string]bool)

	err := filepath.WalkDir(dataDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			log.GetLogger().Error("failed to access path", "path", path, "error", err)
			return err
		}
		if d.IsDir() {
			return nil
		}

		f, err := os.Open(path)
		if err != nil {
			log.GetLogger().Error("failed to open domain file", "path", path, "error", err)
			return err
		}
		defer f.Close()

		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "" || strings.HasPrefix(line, "#") ||
				strings.HasPrefix(line, "include:") ||
				strings.HasPrefix(line, "regexp:") {
				continue
			}

			// Strip known prefixes to get the bare domain.
			for _, prefix := range []string{"domain:", "full:"} {
				if strings.HasPrefix(line, prefix) {
					line = strings.TrimPrefix(line, prefix)
					break
				}
			}

			// Strip inline attributes (e.g., "example.com @cn").
			if idx := strings.IndexByte(line, ' '); idx != -1 {
				line = line[:idx]
			}

			if line != "" {
				domains[line] = true
			}
		}
		if err := scanner.Err(); err != nil {
			log.GetLogger().Error("error reading domain file", "path", path, "error", err)
			return err
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	var result []string
	for domain := range domains {
		result = append(result, domain)
	}
	return result, nil
}
