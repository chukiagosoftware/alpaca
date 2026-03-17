package main

import (
	"github.com/pulumi/pulumi-gcp/sdk/v7/go/gcp/vertex"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		conf := config.New(ctx, "pulumiVectorIndex")
		projectID := conf.Require("gcpProjectID")
		ctx.Export("projectID", pulumi.String(projectID))
		region := conf.Require("region")
		ctx.Export("region", pulumi.String(region))
		indexName := conf.Require("indexName")
		indexDisplayName := conf.Require("indexDisplayName")
		indexDescription := conf.Require("indexDescription")

		// Vertex AI Vector Search Index
		// This creates the index resource. You'll need to specify
		// the GCS bucket path where your vector data is stored
		// and define the index configuration.
		vertexSearchIndex, err := vertex.NewAiIndex(ctx, indexName, &vertex.AiIndexArgs{
			Project:     pulumi.String(projectID),
			Region:      pulumi.String(region),
			DisplayName: pulumi.String(indexDisplayName), // A user-friendly name for your index
			Description: pulumi.String(indexDescription),
			Metadata: &vertex.AiIndexMetadataArgs{
				ContentsDeltaUri: pulumi.String("gs://alpaca_reviews_gemini-001"),
				Config: &vertex.AiIndexMetadataConfigArgs{
					Dimensions:                pulumi.Int(3072),              // Replace with the dimension of your embeddings (e.g., 768 for BERT, 1536 for OpenAI)
					ApproximateNeighborsCount: pulumi.Int(10),                // Number of neighbors to return during approximate search
					FeatureNormType:           pulumi.String("UNIT_L2_NORM"), // L1_NORM, L2_NORM, or NONE. L2_NORM is common.
					DistanceMeasureType:       pulumi.String("DOT_PRODUCT_DISTANCE"),
					ShardSize:                 pulumi.String("SHARD_SIZE_SMALL"),
					AlgorithmConfig: &vertex.AiIndexMetadataConfigAlgorithmConfigArgs{
						TreeAhConfig: &vertex.AiIndexMetadataConfigAlgorithmConfigTreeAhConfigArgs{
							LeafNodeEmbeddingCount:   pulumi.Int(1000),
							LeafNodesToSearchPercent: pulumi.IntPtr(5),
						},
						// OR uncomment for Brute Force config if preferred for smaller datasets
						// BruteForceConfig: &vertex.AiIndexMetadataConfigAlgorithmConfigBruteForceConfigArgs{},
					},
				},
			},
			// For Prod
			// Network: pulumi.String("projects/your-project-id/global/networks/your-vpc-network-name"),
		})
		if err != nil {
			return err
		}

		ctx.Export("indexName", vertexSearchIndex.DisplayName)
		ctx.Export("indexId", vertexSearchIndex.Name) // This will be the full resource name, e.g., projects/.../locations/.../indexes/...

		return nil
	})
}
