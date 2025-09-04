package server

import (
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"main/utils"

	"go.uber.org/zap"
)

// ExportSchema exports the GraphQL schema to a file and optionally deploys to Apollo Studio
func ExportSchema() error {
	schemaPath := filepath.Join(".", "schema.graphql")

	// Build federated SDL by concatenating source SDL files (what _service.sdl would return)
	sdl, err := buildFederatedSDL()
	if err != nil {
		log.Printf("Error building federated SDL: %v", err)
		return err
	}

	// Make common Relay primitives shareable in this subgraph as well
	sdl = addShareableToCommonTypes(sdl)

	file, err := os.Create(schemaPath)
	if err != nil {
		log.Printf("Error creating file: %v", err)
		return err
	}
	defer file.Close()

	if _, err := file.WriteString(sdl); err != nil {
		log.Printf("Error writing schema file: %v", err)
		return err
	}

	log.Printf("Schema generated to file: %s", schemaPath)

	// Deploy to Apollo Studio if configured
	if os.Getenv("APOLLO_DEPLOY_ON_EXPORT") == "true" {
		utils.Logger.Info("Deploying schema to Apollo Studio...")

		// Determine deployment type based on configuration
		useFederation := os.Getenv("APOLLO_USE_FEDERATION")

		if useFederation == "true" {
			// Deploy as federated subgraph
			utils.Logger.Info("Using Federation deployment mode")
			if err := DeploySchemaToApollo(schemaPath); err != nil {
				utils.Logger.Warn("Apollo federation deployment failed",
					zap.Error(err),
					zap.String("hint", "Ensure your graph supports federation in Apollo Studio"),
				)
			}
		} else {
			// Try standalone deployment first, fallback to subgraph
			if err := DeploySchemaToApolloStandalone(schemaPath); err != nil {
				utils.Logger.Warn("Apollo standalone deployment failed, trying federation",
					zap.Error(err),
				)
				// Fallback to federation deployment
				if err := DeploySchemaToApollo(schemaPath); err != nil {
					utils.Logger.Error("Apollo deployment failed",
						zap.Error(err),
						zap.String("hint", "Check your Apollo configuration in .env file"),
					)
				}
			}
		}
	} else {
		utils.Logger.Info("Apollo deployment skipped",
			zap.String("hint", "Set APOLLO_DEPLOY_ON_EXPORT=true to enable automatic deployment"),
		)
	}

	return nil
}

// buildFederatedSDL joins all SDL files under graph/schema into a single SDL string.
// This mirrors what the federation runtime returns via _service.sdl and avoids
// including internal types like _Entity/_Any/_Service in the published schema.
func buildFederatedSDL() (string, error) {
	patterns := []string{filepath.Join("graph", "schema", "*.graphql")}
	var files []string
	for _, p := range patterns {
		matches, err := filepath.Glob(p)
		if err != nil {
			return "", err
		}
		files = append(files, matches...)
	}

	// Stable order for deterministic output
	sort.Strings(files)

	var b strings.Builder
	for _, f := range files {
		data, err := os.ReadFile(f)
		if err != nil {
			return "", err
		}
		b.WriteString(string(data))
		b.WriteString("\n\n")
	}
	return b.String(), nil
}

// addShareableToCommonTypes injects @shareable on the Query and PageInfo type definitions
// inside the SDL string to mark their fields as shareable across subgraphs.
// For Query type, it also removes node/nodes fields as they should be defined by the gateway.
func addShareableToCommonTypes(input string) string {
	input = addDirectiveToTypeLine(input, "Query", "@shareable")
	input = addDirectiveToTypeLine(input, "PageInfo", "@shareable")
	input = removeNodeFieldsFromQuery(input)
	return input
}

func removeNodeFieldsFromQuery(input string) string {
	lines := strings.Split(input, "\n")
	result := []string{}
	inQueryType := false
	skipLines := false
	inDocBlock := false

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Check if we're entering the main Query type (not extend)
		if strings.HasPrefix(trimmed, "type Query") && !strings.Contains(line, "extend") {
			inQueryType = true
			result = append(result, line)
			continue
		}

		// If we're in the Query type
		if inQueryType {
			// Check for end of Query type
			if trimmed == "}" {
				inQueryType = false
				skipLines = false
				inDocBlock = false
				result = append(result, line)
				continue
			}

			// Handle documentation blocks
			if strings.HasPrefix(trimmed, "\"\"\"") {
				// Starting a doc block - check what comes after
				if !inDocBlock {
					inDocBlock = true
					// Look ahead to see what field this documents
					for j := i + 1; j < len(lines); j++ {
						nextLine := strings.TrimSpace(lines[j])
						if strings.Contains(nextLine, "\"\"\"") && j != i {
							// End of doc block, check next line
							if j+1 < len(lines) {
								fieldLine := strings.TrimSpace(lines[j+1])
								if strings.HasPrefix(fieldLine, "node(") || strings.HasPrefix(fieldLine, "nodes(") {
									skipLines = true
								}
							}
							break
						}
					}
				} else {
					// Ending a doc block
					inDocBlock = false
					if skipLines && strings.Contains(trimmed, "\"\"\"") {
						continue
					}
				}

				if skipLines {
					continue
				}
			}

			// Skip content inside doc blocks for node/nodes
			if inDocBlock && skipLines {
				continue
			}

			// Check for node/nodes field definitions
			if strings.HasPrefix(trimmed, "node(") || strings.HasPrefix(trimmed, "nodes(") {
				skipLines = true
				continue
			}

			// If we're skipping and find the end of field definition
			if skipLines {
				if strings.Contains(trimmed, "): Node") || strings.Contains(trimmed, "): [Node]") {
					skipLines = false
					inDocBlock = false
				}
				continue
			}

			// Keep the line if not skipping
			result = append(result, line)
		} else {
			// Not in Query type, keep all lines
			result = append(result, line)
		}
	}

	return strings.Join(result, "\n")
}

func addDirectiveToTypeLine(input, typeName, directive string) string {
	// Split into lines for easier processing
	lines := strings.Split(input, "\n")

	// Look for the exact pattern "type <typeName> {" (not "extend type")
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Check if this is the main type definition (not an extend)
		if strings.HasPrefix(trimmed, "type "+typeName) && !strings.Contains(line, "extend") {
			// Check if it already has the directive
			if strings.Contains(line, directive) {
				continue
			}

			// Add the directive
			if strings.HasSuffix(trimmed, "{") {
				// "type Query {" case
				lines[i] = strings.Replace(line, "type "+typeName+" {", "type "+typeName+" "+directive+" {", 1)
			} else {
				// "type Query" on its own line case
				lines[i] = strings.Replace(line, "type "+typeName, "type "+typeName+" "+directive, 1)
			}
		}
	}

	return strings.Join(lines, "\n")
}
