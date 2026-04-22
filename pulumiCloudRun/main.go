package main

import (
	"github.com/pulumi/pulumi-gcp/sdk/v7/go/gcp/cloudrun"
	"github.com/pulumi/pulumi-gcp/sdk/v7/go/gcp/cloudrunv2"
	"github.com/pulumi/pulumi-gcp/sdk/v7/go/gcp/projects"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		cfg := config.New(ctx, "pulumiCloudRun")

		// Read sensitive values from pulumi config (set via `pulumi config set --secret`)
		// Non-sensitive values are hardcoded from config.yaml
		project := cfg.Require("gcpProjectID")
		region := cfg.Require("region")
		image := cfg.Require("image")
		endpointPublicDomain := cfg.Require("endpointPublicDomainName")
		endpointID := cfg.Require("endpointID")
		deployedIndexID := cfg.Require("deployedIndexID")
		indexID := cfg.Require("indexID")

		// -- Service Account Vertex permission
		sa, err := projects.NewIAMMember(ctx, "alpaca-cloudrun-sa-binding", &projects.IAMMemberArgs{
			Project: pulumi.String(project),
			Role:    pulumi.String("roles/vectorsearch.viewer"),                                               //aiplatform.user
			Member:  pulumi.String("serviceAccount:alpaca-cloudrun-sa@golang1212025.iam.gserviceaccount.com"), // placeholder
		})
		if err != nil {
			return err
		}
		_ = sa

		bqViewer, err := projects.NewIAMMember(ctx, "alpaca-cloudrun-sa-binding-bq", &projects.IAMMemberArgs{
			Project: pulumi.String(project),
			Role:    pulumi.String("roles/bigquery.dataViewer"),
			Member:  pulumi.String("serviceAccount:alpaca-cloudrun-sa@golang1212025.iam.gserviceaccount.com"), // placeholder
		})
		if err != nil {
			return err
		}
		_ = bqViewer

		bqJobUser, err := projects.NewIAMMember(ctx, "alpaca-cloudrun-sa-binding-bqJob", &projects.IAMMemberArgs{
			Project: pulumi.String(project),
			Role:    pulumi.String("roles/bigquery.jobUser"),
			Member:  pulumi.String("serviceAccount:alpaca-cloudrun-sa@golang1212025.iam.gserviceaccount.com"), // placeholder
		})
		if err != nil {
			return err
		}
		_ = bqJobUser

		// -- Cloud Run v2 Service --
		service, err := cloudrunv2.NewService(ctx, "alpaca-search", &cloudrunv2.ServiceArgs{
			Name:     pulumi.String("alpaca-search"),
			Project:  pulumi.String(project),
			Location: pulumi.String(region),

			Template: &cloudrunv2.ServiceTemplateArgs{
				// Smallest/cheapest: 1 vCPU, 512Mi RAM, scale to 0
				Scaling: &cloudrunv2.ServiceTemplateScalingArgs{
					MinInstanceCount: pulumi.Int(0),
					MaxInstanceCount: pulumi.Int(2),
				},
				Containers: cloudrunv2.ServiceTemplateContainerArray{
					&cloudrunv2.ServiceTemplateContainerArgs{
						Image: pulumi.String(image),
						Resources: &cloudrunv2.ServiceTemplateContainerResourcesArgs{
							Limits: pulumi.StringMap{
								"cpu":    pulumi.String("2"),
								"memory": pulumi.String("1024Mi"),
							},
							CpuIdle: pulumi.Bool(true), // only use CPU during request processing
						},
						// Mirror all values from config.yaml as env vars
						Envs: cloudrunv2.ServiceTemplateContainerEnvArray{
							&cloudrunv2.ServiceTemplateContainerEnvArgs{
								Name:  pulumi.String("PROJECT_ID"),
								Value: pulumi.String(project),
							},
							&cloudrunv2.ServiceTemplateContainerEnvArgs{
								Name:  pulumi.String("LOCATION"),
								Value: pulumi.String(region),
							},
							&cloudrunv2.ServiceTemplateContainerEnvArgs{
								Name:  pulumi.String("DATASET_ID"),
								Value: pulumi.String("alpacaCentral"),
							},
							&cloudrunv2.ServiceTemplateContainerEnvArgs{
								Name:  pulumi.String("GIN_MODE"),
								Value: pulumi.String("release"),
							},
							&cloudrunv2.ServiceTemplateContainerEnvArgs{
								Name:  pulumi.String("INDEX_ID"),
								Value: pulumi.String(indexID),
							},
							&cloudrunv2.ServiceTemplateContainerEnvArgs{
								Name:  pulumi.String("ENDPOINT_ID"),
								Value: pulumi.String(endpointID),
							},
							&cloudrunv2.ServiceTemplateContainerEnvArgs{
								Name:  pulumi.String("DEPLOYED_INDEX_ID"),
								Value: pulumi.String(deployedIndexID),
							},
							&cloudrunv2.ServiceTemplateContainerEnvArgs{
								Name:  pulumi.String("ENDPOINT_PUBLIC_DOMAIN_NAME"),
								Value: pulumi.String(endpointPublicDomain),
							},
							&cloudrunv2.ServiceTemplateContainerEnvArgs{
								Name:  pulumi.String("GOOGLE_GENAI_USE_VERTEXAI"),
								Value: pulumi.String("1"),
							},
							&cloudrunv2.ServiceTemplateContainerEnvArgs{
								Name:  pulumi.String("LIMIT"),
								Value: pulumi.String("15"),
							},
							&cloudrunv2.ServiceTemplateContainerEnvArgs{
								Name:  pulumi.String("COMPLETION_MODEL"),
								Value: pulumi.String("gemini-2.5-pro"),
							},
							&cloudrunv2.ServiceTemplateContainerEnvArgs{
								Name:  pulumi.String("QUERY"),
								Value: pulumi.String("Find a quality hotel that is not noisy. This means that it does not have street noise. It does not have thin walls. It does not have loud neighbors or loud noises. You can't hear neighbors upstairs, music outside or traffic on the street."),
							},
							&cloudrunv2.ServiceTemplateContainerEnvArgs{
								Name:  pulumi.String("PROMPT"),
								Value: pulumi.String("You are a Hotel Review Assistant. Choose 5 reviews from the following review_list which best fit the query: %s and include the hotel name, the review, city and country. If the review list is empty say you didn't receive any list of reviews. Review_list: %s"),
							},
						},
						Ports: cloudrunv2.ServiceTemplateContainerPortArray{
							&cloudrunv2.ServiceTemplateContainerPortArgs{
								ContainerPort: pulumi.Int(8080),
							},
						},
					},
				},
			},
		})
		if err != nil {
			return err
		}

		// -- Allow unauthenticated access (no domain yet, public URL) --
		_, err = cloudrun.NewIamMember(ctx, "alpaca-search-public", &cloudrun.IamMemberArgs{
			Project:  pulumi.String(project),
			Location: pulumi.String(region),
			Service:  service.Name,
			Role:     pulumi.String("roles/run.invoker"),
			Member:   pulumi.String("allUsers"),
		})
		if err != nil {
			return err
		}

		// -- Outputs --
		ctx.Export("serviceUrl", service.Uri)
		ctx.Export("serviceName", service.Name)

		return nil
	})
}
