package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type LocaleMap map[string]interface{}

// Recursively merge two maps, detecting key conflicts
func mergeLocaleMap(target LocaleMap, source LocaleMap, sourceFile string, conflicts *[]string) {
	for key, value := range source {
		if existing, exists := target[key]; exists {
			// Check if both values are maps - if so, merge recursively
			if existingMap, existingIsMap := existing.(map[string]interface{}); existingIsMap {
				if sourceMap, sourceIsMap := value.(map[string]interface{}); sourceIsMap {
					// Both are maps, merge recursively
					mergeLocaleMap(existingMap, sourceMap, sourceFile, conflicts)
					continue
				}
			}

			// If we get here, there's a conflict
			*conflicts = append(*conflicts, fmt.Sprintf("Key '%s' already exists (source: %s)", key, sourceFile))
		} else {
			// No conflict, safe to add
			if sourceMap, ok := value.(map[string]interface{}); ok {
				// If it's a map, create a copy to avoid reference issues
				target[key] = copyMap(sourceMap)
			} else {
				target[key] = value
			}
		}
	}
}

// Deep copy a map to avoid reference issues
func copyMap(source map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	for k, v := range source {
		if mapValue, ok := v.(map[string]interface{}); ok {
			result[k] = copyMap(mapValue)
		} else {
			result[k] = v
		}
	}
	return result
}

// Sort JSON keys recursively
func sortJSON(data LocaleMap) LocaleMap {
	result := make(LocaleMap)

	// Get all keys and sort them
	keys := make([]string, 0, len(data))
	for k := range data {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// Process keys in sorted order
	for _, k := range keys {
		if nestedMap, ok := data[k].(map[string]interface{}); ok {
			result[k] = sortJSON(nestedMap)
		} else {
			result[k] = data[k]
		}
	}

	return result
}

// Save JSON with proper formatting
func saveJSON(filePath string, data LocaleMap) error {
	// Sort the data
	sortedData := sortJSON(data)

	// Marshal with indentation
	jsonBytes, err := json.MarshalIndent(sortedData, "", "  ")
	if err != nil {
		return err
	}

	// Ensure directory exists
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	// Add newline at the end for proper Git formatting
	jsonBytes = append(jsonBytes, '\n')

	return os.WriteFile(filePath, jsonBytes, 0644)
}

// Load JSON file
func loadJSON(filePath string) (LocaleMap, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	var localeMap LocaleMap
	err = json.Unmarshal(data, &localeMap)
	if err != nil {
		return nil, fmt.Errorf("failed to parse JSON in %s: %v", filePath, err)
	}

	return localeMap, nil
}

func main() {
	localesDir := "locales"
	buildDir := filepath.Join(localesDir, "build")

	// Find all JSON files with _en and _ru suffixes
	files, err := filepath.Glob(filepath.Join(localesDir, "*_en.json"))
	if err != nil {
		fmt.Printf("Error finding _en files: %v\n", err)
		os.Exit(1)
	}

	// Build lists of en and ru files
	var enFiles, ruFiles []string

	for _, enFile := range files {
		enFiles = append(enFiles, enFile)

		// Find corresponding ru file
		baseName := strings.TrimSuffix(filepath.Base(enFile), "_en.json")
		ruFile := filepath.Join(localesDir, baseName+"_ru.json")

		if _, err := os.Stat(ruFile); err == nil {
			ruFiles = append(ruFiles, ruFile)
		} else {
			fmt.Printf("Warning: Missing Russian file for %s\n", enFile)
		}
	}

	// Also check for ru files that don't have corresponding en files
	allRuFiles, err := filepath.Glob(filepath.Join(localesDir, "*_ru.json"))
	if err != nil {
		fmt.Printf("Error finding _ru files: %v\n", err)
		os.Exit(1)
	}

	for _, ruFile := range allRuFiles {
		baseName := strings.TrimSuffix(filepath.Base(ruFile), "_ru.json")
		enFile := filepath.Join(localesDir, baseName+"_en.json")

		if _, err := os.Stat(enFile); err != nil {
			fmt.Printf("Warning: Missing English file for %s\n", ruFile)
			ruFiles = append(ruFiles, ruFile)
		}
	}

	fmt.Printf("Found %d English files and %d Russian files\n", len(enFiles), len(ruFiles))

	// Build English locale
	var enConflicts []string
	enMap := make(LocaleMap)

	for _, file := range enFiles {
		fmt.Printf("Processing EN file: %s\n", file)
		fileMap, err := loadJSON(file)
		if err != nil {
			fmt.Printf("Error loading %s: %v\n", file, err)
			os.Exit(1)
		}
		mergeLocaleMap(enMap, fileMap, file, &enConflicts)
	}

	if len(enConflicts) > 0 {
		fmt.Println("\nERROR: Key conflicts found in English files:")
		for _, conflict := range enConflicts {
			fmt.Printf("  - %s\n", conflict)
		}
		os.Exit(1)
	}

	// Build Russian locale
	var ruConflicts []string
	ruMap := make(LocaleMap)

	for _, file := range ruFiles {
		fmt.Printf("Processing RU file: %s\n", file)
		fileMap, err := loadJSON(file)
		if err != nil {
			fmt.Printf("Error loading %s: %v\n", file, err)
			os.Exit(1)
		}
		mergeLocaleMap(ruMap, fileMap, file, &ruConflicts)
	}

	if len(ruConflicts) > 0 {
		fmt.Println("\nERROR: Key conflicts found in Russian files:")
		for _, conflict := range ruConflicts {
			fmt.Printf("  - %s\n", conflict)
		}
		os.Exit(1)
	}

	// Save built files
	enOutputFile := filepath.Join(buildDir, "en.json")
	ruOutputFile := filepath.Join(buildDir, "ru.json")

	fmt.Printf("Saving English locale to: %s\n", enOutputFile)
	if err := saveJSON(enOutputFile, enMap); err != nil {
		fmt.Printf("Error saving English locale: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Saving Russian locale to: %s\n", ruOutputFile)
	if err := saveJSON(ruOutputFile, ruMap); err != nil {
		fmt.Printf("Error saving Russian locale: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("\nLocale build completed successfully!")
	fmt.Printf("English keys: %d\n", countKeys(enMap))
	fmt.Printf("Russian keys: %d\n", countKeys(ruMap))
}

// Recursively count keys in a locale map
func countKeys(localeMap LocaleMap) int {
	count := 0
	for _, value := range localeMap {
		if nestedMap, ok := value.(map[string]interface{}); ok {
			count += countKeys(nestedMap)
		} else {
			count++
		}
	}
	return count
}
