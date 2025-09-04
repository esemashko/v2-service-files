package server

import (
	"fmt"
	"main/utils"
	"os"
	"os/exec"
	"strings"

	"go.uber.org/zap"
)

// DeploySchemaToApollo deploys the GraphQL schema to Apollo Studio as a federated subgraph
func DeploySchemaToApollo(schemaPath string) error {
	// Get Apollo configuration from environment
	apolloKey := os.Getenv("APOLLO_KEY")
	apolloGraph := os.Getenv("APOLLO_GRAPH_ID")
	apolloVariant := os.Getenv("APOLLO_GRAPH_VARIANT")
	apolloSubgraphName := os.Getenv("APOLLO_SUBGRAPH_NAME")
	apolloRoutingURL := os.Getenv("APOLLO_ROUTING_URL")

	// Check if Apollo deployment is enabled
	if apolloKey == "" {
		utils.Logger.Info("Apollo deployment skipped - APOLLO_KEY not set")
		return nil
	}

	// Validate required configuration
	if apolloGraph == "" {
		apolloGraph = "tairo" // Default graph name
	}
	if apolloVariant == "" {
		apolloVariant = "current" // Default variant
	}
	if apolloSubgraphName == "" {
		apolloSubgraphName = "service-tenant" // Default subgraph name for tenant service
	}
	if apolloRoutingURL == "" {
		// Build routing URL from environment
		port := os.Getenv("APP_CORE_PORT")
		if port == "" {
			port = "9024"
		}
		// Use the actual service URL for federation
		apolloRoutingURL = fmt.Sprintf("http://localhost:%s/graphql", port)
	}

	// Check if rover CLI is installed
	if _, err := exec.LookPath("rover"); err != nil {
		utils.Logger.Warn("rover CLI not found - installing instructions: https://www.apollographql.com/docs/rover/getting-started")
		return fmt.Errorf("rover CLI not installed: %w", err)
	}

	// Build rover command
	graphRef := fmt.Sprintf("%s@%s", apolloGraph, apolloVariant)

	utils.Logger.Info("Deploying schema to Apollo Studio",
		zap.String("graph", apolloGraph),
		zap.String("variant", apolloVariant),
		zap.String("subgraph", apolloSubgraphName),
		zap.String("routing_url", apolloRoutingURL),
		zap.String("schema_file", schemaPath),
	)

	// Execute rover subgraph publish command
	cmd := exec.Command("rover", "subgraph", "publish", graphRef,
		"--schema", schemaPath,
		"--name", apolloSubgraphName,
		"--routing-url", apolloRoutingURL,
	)

	// Set APOLLO_KEY as environment variable for the command
	cmd.Env = append(os.Environ(), fmt.Sprintf("APOLLO_KEY=%s", apolloKey))

	// Capture output
	output, err := cmd.CombinedOutput()
	outputStr := string(output)

	if err != nil {
		// Log error but don't fail the entire process
		utils.Logger.Error("Apollo schema deployment failed",
			zap.Error(err),
			zap.String("output", outputStr),
		)

		// Check for common errors
		if strings.Contains(outputStr, "Could not find graph") {
			utils.Logger.Info("Hint: Create the graph first in Apollo Studio or use 'rover graph create'")
		}
		if strings.Contains(outputStr, "401") || strings.Contains(outputStr, "403") {
			utils.Logger.Info("Hint: Check your APOLLO_KEY permissions")
		}

		return fmt.Errorf("apollo deployment failed: %w", err)
	}

	utils.Logger.Info("Schema successfully deployed to Apollo Studio",
		zap.String("output", outputStr),
	)

	return nil
}

// DeploySchemaToApolloStandalone deploys schema as a standalone graph (not federation)
func DeploySchemaToApolloStandalone(schemaPath string) error {
	// Get Apollo configuration from environment
	apolloKey := os.Getenv("APOLLO_KEY")
	apolloGraph := os.Getenv("APOLLO_GRAPH_ID")
	apolloVariant := os.Getenv("APOLLO_GRAPH_VARIANT")

	// Check if Apollo deployment is enabled
	if apolloKey == "" {
		utils.Logger.Info("Apollo deployment skipped - APOLLO_KEY not set")
		return nil
	}

	// Validate required configuration
	if apolloGraph == "" {
		apolloGraph = "tairo" // Default graph name
	}
	if apolloVariant == "" {
		apolloVariant = "current" // Default variant
	}

	// Check if rover CLI is installed
	if _, err := exec.LookPath("rover"); err != nil {
		utils.Logger.Warn("rover CLI not found - installing instructions: https://www.apollographql.com/docs/rover/getting-started")
		return fmt.Errorf("rover CLI not installed: %w", err)
	}

	// Build rover command for standalone graph
	graphRef := fmt.Sprintf("%s@%s", apolloGraph, apolloVariant)

	utils.Logger.Info("Deploying schema to Apollo Studio (standalone)",
		zap.String("graph", apolloGraph),
		zap.String("variant", apolloVariant),
		zap.String("schema_file", schemaPath),
	)

	// Execute rover graph publish command (for non-federated graphs)
	cmd := exec.Command("rover", "graph", "publish", graphRef,
		"--schema", schemaPath,
	)

	// Set APOLLO_KEY as environment variable for the command
	cmd.Env = append(os.Environ(), fmt.Sprintf("APOLLO_KEY=%s", apolloKey))

	// Capture output
	output, err := cmd.CombinedOutput()
	outputStr := string(output)

	if err != nil {
		// If standalone fails, try subgraph publish as fallback
		utils.Logger.Warn("Standalone graph publish failed, trying subgraph publish",
			zap.String("error", outputStr),
		)
		return DeploySchemaToApollo(schemaPath)
	}

	utils.Logger.Info("Schema successfully deployed to Apollo Studio (standalone)",
		zap.String("output", outputStr),
	)

	return nil
}
