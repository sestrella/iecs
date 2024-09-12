package cmd

import (
	"context"
	"fmt"
	"log"
	"slices"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/aws/aws-sdk-go-v2/service/ecs/types"
	"github.com/charmbracelet/bubbles/list"
	"github.com/spf13/cobra"
)

var deployClusterId string
var deployServiceId string
var deployImages []string
var deployWait bool

var deployCmd = &cobra.Command{
	Use:   "deploy",
	Short: "A brief description of your command",
	Long:  "TODO",
	Run: func(cmd *cobra.Command, args []string) {
		if len(deployImages) == 0 {
			log.Fatal("Expected at least one image")
		}

		cfg, err := config.LoadDefaultConfig(context.TODO())
		if err != nil {
			log.Fatal(err)
		}

		client := ecs.NewFromConfig(cfg)
		cluster, err := selectCluster(context.TODO(), client, deployClusterId)
		if err != nil {
			log.Fatal(err)
		}

		service, err := selectService(context.TODO(), client, *cluster)
		if err != nil {
			log.Fatal(err)
		}

		// TODO: Try to avoid calling DescribeServices twice
		describeServicesOutput, err := client.DescribeServices(context.TODO(), &ecs.DescribeServicesInput{
			Cluster:  &cluster.arn,
			Services: []string{service.arn},
		})
		if err != nil {
			log.Fatal(err)
		}

		describeTaskDefinitionOutput, err := client.DescribeTaskDefinition(context.TODO(), &ecs.DescribeTaskDefinitionInput{
			TaskDefinition: describeServicesOutput.Services[0].TaskDefinition,
		})
		if err != nil {
			log.Fatal(err)
		}

		// TODO: Update image tag
		currentTaskDefinition := describeTaskDefinitionOutput.TaskDefinition
		newContainerDefinitions := currentTaskDefinition.ContainerDefinitions
		availableContainers := make([]string, len(newContainerDefinitions))
		for idx, containerDefinition := range newContainerDefinitions {
			availableContainers[idx] = *containerDefinition.Name
		}

		for _, deployImage := range deployImages {
			newImageSlices := strings.Split(deployImage, "@")
			if len(newImageSlices) != 2 {
				log.Fatalf("Expected '%s' to be of the form: <container>@<tag>", deployImage)
			}

			index := slices.IndexFunc(newContainerDefinitions, func(containerDefinition types.ContainerDefinition) bool {
				return *containerDefinition.Name == newImageSlices[0]
			})
			if index == -1 {
				log.Fatalf("Container '%s' not found, try one of the following: %s", newImageSlices[0], availableContainers)
			}

			currentImageSlices := strings.Split(*newContainerDefinitions[index].Image, ":")
			if len(currentImageSlices) != 2 {
				log.Fatalf("Expected '%s' to be of the form: <image>:<tag>", *newContainerDefinitions[index].Image)
			}

			newImage := fmt.Sprintf("%s:%s", currentImageSlices[0], newImageSlices[1])
			log.Printf("Updating image for container '%s' from '%s' to '%s'", newImageSlices[0], *newContainerDefinitions[index].Image, newImage)
			newContainerDefinitions[index].Image = &newImage
		}

		registerTaskDefinitionOutput, err := client.RegisterTaskDefinition(context.TODO(), &ecs.RegisterTaskDefinitionInput{
			ContainerDefinitions:    newContainerDefinitions,
			Cpu:                     currentTaskDefinition.Cpu,
			ExecutionRoleArn:        currentTaskDefinition.ExecutionRoleArn,
			Family:                  currentTaskDefinition.Family,
			Memory:                  currentTaskDefinition.Memory,
			NetworkMode:             currentTaskDefinition.NetworkMode,
			PlacementConstraints:    currentTaskDefinition.PlacementConstraints,
			RequiresCompatibilities: currentTaskDefinition.RequiresCompatibilities,
			RuntimePlatform:         currentTaskDefinition.RuntimePlatform,
			TaskRoleArn:             currentTaskDefinition.TaskRoleArn,
			Volumes:                 currentTaskDefinition.Volumes,
		})
		if err != nil {
			log.Fatal(err)
		}

		newTaskDefinitionArn := registerTaskDefinitionOutput.TaskDefinition.TaskDefinitionArn
		log.Printf("Task definition ARN: %s", *newTaskDefinitionArn)
		updateServiceOutput, err := client.UpdateService(context.TODO(), &ecs.UpdateServiceInput{
			Cluster:        &cluster.arn,
			Service:        &service.arn,
			TaskDefinition: newTaskDefinitionArn,
		})
		if err != nil {
			log.Fatal(err)
		}

		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()

		timeout := time.NewTicker(5 * time.Minute)
		defer timeout.Stop()

		if deployWait {
			for {
				select {
				case <-ticker.C:
					describeServicesOutput, err := client.DescribeServices(context.TODO(), &ecs.DescribeServicesInput{
						Cluster:  updateServiceOutput.Service.ClusterArn,
						Services: []string{*updateServiceOutput.Service.ServiceArn},
					})
					if err != nil {
						log.Fatal(err)
					}

					service := describeServicesOutput.Services[0]
					if *service.Status == "ACTIVE" {
						log.Printf("Service '%s' active", *service.ServiceName)
						return
					}
					log.Printf("Waiting for service '%s' to be active...", *service.ServiceName)
				case <-timeout.C:
					log.Print("Timeout")
					return
				}
			}
		}
	},
}

func selectService(ctx context.Context, client *ecs.Client, cluster item) (*item, error) {
	if deployServiceId != "" {
		describeServicesOutput, err := client.DescribeServices(ctx, &ecs.DescribeServicesInput{
			Cluster:  &cluster.arn,
			Services: []string{deployServiceId},
		})
		if err != nil {
			return nil, fmt.Errorf("Unable to describe service '%s': %w", deployServiceId, err)
		}

		if len(describeServicesOutput.Services) == 0 {
			return nil, fmt.Errorf("Service '%s' not found", deployServiceId)
		}

		service := describeServicesOutput.Services[0]
		serviceArnSlices := strings.Split(*service.ServiceArn, "/")
		return &item{
			name: fmt.Sprintf("%s/%s", serviceArnSlices[1], serviceArnSlices[2]),
			arn:  *service.ServiceArn,
		}, nil
	}

	listServicesOutput, err := client.ListServices(ctx, &ecs.ListServicesInput{
		Cluster: &cluster.arn,
	})
	if err != nil {
		return nil, fmt.Errorf("Unable to list services on cluster '%s': %w", cluster.name, err)
	}
	if len(listServicesOutput.ServiceArns) == 0 {
		return nil, fmt.Errorf("No services found on cluster '%s'", cluster.name)
	}

	items := []list.Item{}
	for _, arn := range listServicesOutput.ServiceArns {
		index := strings.LastIndex(arn, "/")
		items = append(items, item{
			name: arn[index+1:],
			arn:  arn,
		})
	}
	return newSelector("Services", items)
}

func init() {
	rootCmd.AddCommand(deployCmd)

	deployCmd.Flags().StringVarP(&deployClusterId, "cluster", "c", "", "TODO")
	deployCmd.Flags().StringVarP(&deployServiceId, "service", "s", "", "TODO")
	deployCmd.Flags().StringArrayVarP(&deployImages, "image", "i", []string{}, "TODO")
	deployCmd.Flags().BoolVarP(&deployWait, "wait", "w", true, "TODO")
}
