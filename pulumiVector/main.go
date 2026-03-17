package main

import (
	"fmt"

	"github.com/pulumi/pulumi-gcp/sdk/v7/go/gcp/vertex"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		// --- Configuration Variables from Pulumi Config ---
		conf := config.New(ctx, "pulumiVertexSearch")

		// Retrieve project ID, location, and index ID from Pulumi configuration
		// You would set these using `pulumi config set gcpProjectID golang1212025` etc.
		gcpProjectID := conf.Require("gcpProjectID")
		gcpLocation := conf.Require("gcpLocation")
		indexID := conf.Require("indexID")
		deployedIndexId := conf.Require("deployedIndexId")
		machineType := conf.Require("machineType")

		// --- 1. Create a Vertex AI IndexEndpoint ---
		// This is the server that will host your deployed index.
		indexEndpoint, err := vertex.NewAiIndexEndpoint(ctx, deployedIndexId, &vertex.AiIndexEndpointArgs{
			Project:     pulumi.String(gcpProjectID),
			Region:      pulumi.String(gcpLocation),
			DisplayName: pulumi.String("alpacaReviewsGemini001-Pulumi"), // A user-friendly name for the endpoint
			Description: pulumi.String("Alpaca Hotel Reviews Gemini-001 Pulumi"),
			// Optional: Network configuration if you need private IP access
			// Network: pulumi.String(fmt.Sprintf("projects/%s/global/networks/YOUR_VPC_NETWORK_NAME", gcpProjectID)),
		})
		if err != nil {
			return fmt.Errorf("error creating IndexEndpoint: %w", err)
		}

		// Output the IndexEndpoint ID and Name for reference
		ctx.Export("indexEndpointId", indexEndpoint.ID())
		ctx.Export("indexEndpointName", indexEndpoint.Name) // This is the full resource name

		// --- 2. Deploy the existing Index to the newly created IndexEndpoint ---
		// This makes your index queryable via the IndexEndpoint.
		indexEndpointName := string(pulumi.String(deployedIndexId))
		deployedIndex, err := vertex.NewAiIndexEndpointDeployedIndex(ctx, indexEndpointName, &vertex.AiIndexEndpointDeployedIndexArgs{
			IndexEndpoint:   indexEndpoint.ID(),                                                                                    // Link to the IndexEndpoint created above
			Index:           pulumi.String(fmt.Sprintf("projects/%s/locations/%s/indexes/%s", gcpProjectID, gcpLocation, indexID)), // Full resource name of your existing Index
			DisplayName:     pulumi.String(deployedIndexId),                                                                        // A name for this specific deployment on the endpoint
			DeployedIndexId: pulumi.String(deployedIndexId),
			DedicatedResources: &vertex.AiIndexEndpointDeployedIndexDedicatedResourcesArgs{
				MinReplicaCount: pulumi.Int(1),
				MaxReplicaCount: pulumi.Int(1),
				MachineSpec: &vertex.AiIndexEndpointDeployedIndexDedicatedResourcesMachineSpecArgs{

					MachineType: pulumi.String(machineType),
				},
			},
		})

		if err != nil {
			return fmt.Errorf("error deploying Index to Endpoint: %w", err)
		}

		// Output the DeployedIndex ID for reference
		ctx.Export("deployedIndexId", deployedIndex.DeployedIndexId)

		return nil
	})
}
