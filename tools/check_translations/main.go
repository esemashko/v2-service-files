package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// Represents a nested map structure for JSON locale files
type LocaleMap map[string]interface{}

func main() {
	var (
		rootPath     string
		fixOption    bool
		removeUnused bool
	)

	flag.StringVar(&rootPath, "path", ".", "Project root path")
	flag.BoolVar(&fixOption, "fix", false, "Generate translation template for missing keys")
	flag.BoolVar(&removeUnused, "remove-unused", false, "Remove unused keys from locale files")
	flag.Parse()

	localesDir := filepath.Join(rootPath, "locales/build")

	// Load locale files
	enFile := filepath.Join(localesDir, "en.json")
	ruFile := filepath.Join(localesDir, "ru.json")

	enMap, err := loadLocaleFile(enFile)
	if err != nil {
		fmt.Printf("Failed to load English locale file: %v\n", err)
		os.Exit(1)
	}

	ruMap, err := loadLocaleFile(ruFile)
	if err != nil {
		fmt.Printf("Failed to load Russian locale file: %v\n", err)
		os.Exit(1)
	}

	// Find all translation keys in the code
	usedKeys, err := findTranslationKeys(rootPath)
	if err != nil {
		fmt.Printf("Error finding translation keys: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Found %d translation keys in the code\n", len(usedKeys))

	// Convert used keys slice to map for easier lookup
	usedKeysMap := make(map[string]bool)
	for _, key := range usedKeys {
		usedKeysMap[key] = true
	}

	// Check each key against locale files
	missingInEn := []string{}
	missingInRu := []string{}

	for _, key := range usedKeys {
		if !hasKey(enMap, key) {
			missingInEn = append(missingInEn, key)
		}
		if !hasKey(ruMap, key) {
			missingInRu = append(missingInRu, key)
		}
	}

	// Find unused keys in locale files
	enAllKeys := getAllKeys(enMap, "")
	ruAllKeys := getAllKeys(ruMap, "")

	unusedInEn := []string{}
	unusedInRu := []string{}

	for _, key := range enAllKeys {
		if !usedKeysMap[key] {
			unusedInEn = append(unusedInEn, key)
		}
	}

	for _, key := range ruAllKeys {
		if !usedKeysMap[key] {
			unusedInRu = append(unusedInRu, key)
		}
	}

	// Sort keys for consistent output
	sort.Strings(missingInEn)
	sort.Strings(missingInRu)
	sort.Strings(unusedInEn)
	sort.Strings(unusedInRu)

	// Print results
	fmt.Println("\n=== RESULTS ===")

	if len(missingInEn) > 0 {
		fmt.Println("\nKeys missing in English translation:")
		for _, key := range missingInEn {
			fmt.Println("  -", key)
		}
	} else {
		fmt.Println("\nAll keys present in English translation!")
	}

	if len(missingInRu) > 0 {
		fmt.Println("\nKeys missing in Russian translation:")
		for _, key := range missingInRu {
			fmt.Println("  -", key)
		}
	} else {
		fmt.Println("\nAll keys present in Russian translation!")
	}

	// Show unused keys
	if len(unusedInEn) > 0 {
		fmt.Printf("\n\u26a0 Unused keys in English translation (%d):\n", len(unusedInEn))
		for _, key := range unusedInEn {
			fmt.Println("  -", key)
		}
	}

	if len(unusedInRu) > 0 {
		fmt.Printf("\n\u26a0 Unused keys in Russian translation (%d):\n", len(unusedInRu))
		for _, key := range unusedInRu {
			fmt.Println("  -", key)
		}
	}

	// Check for keys that exist in one locale but not in another
	enOnlyKeys := findKeysInOneLocaleOnly(enMap, ruMap, "")
	ruOnlyKeys := findKeysInOneLocaleOnly(ruMap, enMap, "")

	if len(enOnlyKeys) > 0 {
		fmt.Println("\nKeys present in English but missing in Russian:")
		for _, key := range enOnlyKeys {
			fmt.Println("  -", key)
		}
	}

	if len(ruOnlyKeys) > 0 {
		fmt.Println("\nKeys present in Russian but missing in English:")
		for _, key := range ruOnlyKeys {
			fmt.Println("  -", key)
		}
	}

	// Remove unused keys if requested
	if removeUnused && (len(unusedInEn) > 0 || len(unusedInRu) > 0) {
		fmt.Println("\n=== REMOVING UNUSED KEYS ===")

		// Process individual locale files in /locales directory
		localesSourceDir := filepath.Join(rootPath, "locales")

		// Find all individual locale files
		entries, err := os.ReadDir(localesSourceDir)
		if err != nil {
			fmt.Printf("Error reading locales directory: %v\n", err)
			os.Exit(1)
		}

		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
				continue
			}

			// Skip the build directory files
			if strings.Contains(entry.Name(), "build") {
				continue
			}

			filePath := filepath.Join(localesSourceDir, entry.Name())

			// Determine which unused keys list to use
			var unusedKeys []string
			if strings.Contains(entry.Name(), "_en.json") {
				unusedKeys = unusedInEn
			} else if strings.Contains(entry.Name(), "_ru.json") {
				unusedKeys = unusedInRu
			} else {
				continue
			}

			// Load the file
			fileMap, err := loadLocaleFile(filePath)
			if err != nil {
				fmt.Printf("Error loading %s: %v\n", filePath, err)
				continue
			}

			// Remove unused keys from this file
			modified := false
			for _, key := range unusedKeys {
				if removeKeyFromMap(fileMap, key) {
					modified = true
					fmt.Printf("  Removed '%s' from %s\n", key, entry.Name())
				}
			}

			// Save the file if modified
			if modified {
				if err := saveLocaleFile(filePath, fileMap); err != nil {
					fmt.Printf("Error saving %s: %v\n", filePath, err)
				} else {
					fmt.Printf("  \u2713 Updated %s\n", entry.Name())
				}
			}
		}
	}

	// Generate fix template if requested
	if fixOption {
		if len(missingInEn) > 0 {
			fmt.Println("\n=== ENGLISH TEMPLATE ===")
			for _, key := range missingInEn {
				fmt.Printf("  \"%s\": \"TRANSLATION NEEDED\",\n", key)
			}
		}

		if len(missingInRu) > 0 {
			fmt.Println("\n=== RUSSIAN TEMPLATE ===")
			for _, key := range missingInRu {
				// If key exists in English, get English text as reference
				var enText string
				if hasKey(enMap, key) {
					enText = getKeyValue(enMap, key)
					fmt.Printf("  \"%s\": \"ПЕРЕВОД: %s\",\n", key, enText)
				} else {
					fmt.Printf("  \"%s\": \"ТРЕБУЕТСЯ ПЕРЕВОД\",\n", key)
				}
			}
		}

		if len(enOnlyKeys) > 0 {
			fmt.Println("\n=== ENGLISH KEYS MISSING IN RUSSIAN ===")
			for _, key := range enOnlyKeys {
				enText := getKeyValue(enMap, key)
				fmt.Printf("  \"%s\": \"ПЕРЕВОД: %s\",\n", key, enText)
			}
		}

		if len(ruOnlyKeys) > 0 {
			fmt.Println("\n=== RUSSIAN KEYS MISSING IN ENGLISH ===")
			for _, key := range ruOnlyKeys {
				ruText := getKeyValue(ruMap, key)
				fmt.Printf("  \"%s\": \"TRANSLATION: %s\",\n", key, ruText)
			}
		}
	}

	// Exit with error code if any issues found (except unused keys unless in strict mode)
	if len(missingInEn) > 0 || len(missingInRu) > 0 || len(enOnlyKeys) > 0 || len(ruOnlyKeys) > 0 {
		os.Exit(1)
	}
}

// Get all keys from locale map recursively
func getAllKeys(localeMap LocaleMap, prefix string) []string {
	var result []string

	for key, value := range localeMap {
		fullKey := key
		if prefix != "" {
			fullKey = prefix + "." + key
		}

		// Check if value is a nested map
		if nestedMap, ok := value.(map[string]interface{}); ok {
			// Recursively get keys from nested map
			nestedKeys := getAllKeys(LocaleMap(nestedMap), fullKey)
			result = append(result, nestedKeys...)
		} else {
			// This is a leaf node (actual translation)
			result = append(result, fullKey)
		}
	}

	return result
}

// Remove a key from locale map (supports nested keys)
func removeKeyFromMap(localeMap LocaleMap, key string) bool {
	parts := strings.Split(key, ".")

	if len(parts) == 1 {
		// Simple key
		if _, exists := localeMap[key]; exists {
			delete(localeMap, key)
			return true
		}
		return false
	}

	// Nested key
	currentMap := localeMap
	for i, part := range parts {
		if i == len(parts)-1 {
			// Last part, delete the key
			if _, exists := currentMap[part]; exists {
				delete(currentMap, part)
				return true
			}
			return false
		}

		// Navigate to nested map
		if nextMap, ok := currentMap[part].(map[string]interface{}); ok {
			currentMap = LocaleMap(nextMap)
		} else {
			return false
		}
	}

	return false
}

// Save locale file
func saveLocaleFile(filePath string, localeMap LocaleMap) error {
	data, err := json.MarshalIndent(localeMap, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(filePath, data, 0644)
}

// Get the value for a key in locale map
func getKeyValue(localeMap LocaleMap, key string) string {
	parts := strings.Split(key, ".")
	currentMap := localeMap

	for i, part := range parts {
		if i == len(parts)-1 {
			// Last part, get the value
			value, exists := currentMap[part]
			if !exists {
				return ""
			}

			// Convert to string if possible
			strValue, ok := value.(string)
			if !ok {
				return fmt.Sprintf("%v", value)
			}
			return strValue
		}

		// Not the last part, navigate deeper
		nextMap, exists := currentMap[part]
		if !exists {
			return ""
		}

		// Check if the next part is a map
		nextMapTyped, ok := nextMap.(map[string]interface{})
		if !ok {
			return ""
		}

		currentMap = nextMapTyped
	}

	return ""
}

// Load locale file into a nested map
func loadLocaleFile(filePath string) (LocaleMap, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	var result LocaleMap
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}

	return result, nil
}

// Find all translation keys in the codebase
func findTranslationKeys(rootPath string) ([]string, error) {
	keys := make(map[string]bool)

	// Regular expressions for finding utils.T calls
	// 1. Simple format: utils.T(ctx, "key")
	simpleKeyRegex := regexp.MustCompile(`utils\.T\s*\(\s*[^,]+\s*,\s*["']([^"']+)["']\s*\)`)

	// 2. With TemplateData: utils.T(ctx, "key", map[string]interface{}{...})
	// Also matches: utils.T(ctx, "key", data) where data is ...TemplateData
	templateKeyRegex := regexp.MustCompile(`utils\.T\s*\(\s*[^,]+\s*,\s*["']([^"']+)["']\s*,\s*(?:map\[|[^)]+)`)

	err := filepath.WalkDir(rootPath, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Skip .git directory
		if strings.Contains(path, ".git") {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// Skip vendor directory
		if strings.Contains(path, "vendor") || strings.Contains(path, "node_modules") {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// Skip tools directory to avoid matching our own check script
		if strings.Contains(path, "/tools/check_translations") {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// Process only Go files
		if !d.IsDir() && strings.HasSuffix(path, ".go") {
			content, err := os.ReadFile(path)
			if err != nil {
				return err
			}

			// Split content into lines to check for comments
			lines := strings.Split(string(content), "\n")
			for _, line := range lines {
				// Skip commented lines
				trimmedLine := strings.TrimSpace(line)
				if strings.HasPrefix(trimmedLine, "//") {
					continue
				}

				// Find simple format matches
				matches := simpleKeyRegex.FindAllStringSubmatch(line, -1)
				for _, match := range matches {
					if len(match) >= 2 {
						key := match[1]
						keys[key] = true
					}
				}

				// Find template format matches
				matches = templateKeyRegex.FindAllStringSubmatch(line, -1)
				for _, match := range matches {
					if len(match) >= 2 {
						key := match[1]
						keys[key] = true
					}
				}
			}
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	// Convert map keys to slice
	result := make([]string, 0, len(keys))
	for key := range keys {
		result = append(result, key)
	}

	return result, nil
}

// Check if a key exists in the locale map
func hasKey(localeMap LocaleMap, key string) bool {
	parts := strings.Split(key, ".")
	currentMap := localeMap

	for i, part := range parts {
		if i == len(parts)-1 {
			// Last part, check if it exists as a key
			_, exists := currentMap[part]
			return exists
		}

		// Not the last part, check if it exists as a map
		nextMap, exists := currentMap[part]
		if !exists {
			return false
		}

		// Check if the next part is a map
		nextMapTyped, ok := nextMap.(map[string]interface{})
		if !ok {
			return false
		}

		currentMap = nextMapTyped
	}

	return false
}

// Find keys that exist in source but not in target
func findKeysInOneLocaleOnly(source, target LocaleMap, prefix string) []string {
	var result []string

	for key, value := range source {
		fullKey := key
		if prefix != "" {
			fullKey = prefix + "." + key
		}

		// Check if key exists in target
		targetValue, exists := target[key]
		if !exists {
			result = append(result, fullKey)
			continue
		}

		// If both are maps, check recursively
		sourceMap, sourceIsMap := value.(map[string]interface{})
		targetMap, targetIsMap := targetValue.(map[string]interface{})

		if sourceIsMap && targetIsMap {
			subResult := findKeysInOneLocaleOnly(LocaleMap(sourceMap), LocaleMap(targetMap), fullKey)
			result = append(result, subResult...)
		}
	}

	// Sort for consistent output
	sort.Strings(result)
	return result
}
