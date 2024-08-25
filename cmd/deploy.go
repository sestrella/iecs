package cmd

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/charmbracelet/bubbles/list"
	"github.com/spf13/cobra"
)

var deployClusterId string
var deployServiceId string

var deployCmd = &cobra.Command{
	Use:   "deploy",
	Short: "A brief description of your command",
	Long:  "TODO",
	Run: func(cmd *cobra.Command, args []string) {
		cfg, err := config.LoadDefaultConfig(context.TODO())
		if err != nil {
			log.Fatal(err)
		}

		client := ecs.NewFromConfig(cfg)
		cluster, err := selectCluster(context.TODO(), client, sshClusterId)
		if err != nil {
			log.Fatal(err)
		}

		service, err := selectService(context.TODO(), client, *cluster)
		if err != nil {
			log.Fatal(err)
		}

		// TODO: Try to avoid calling DescribeServices twice
		services, err := client.DescribeServices(context.TODO(), &ecs.DescribeServicesInput{
			Cluster:  &cluster.arn,
			Services: []string{service.arn},
		})
		if err != nil {
			log.Fatal(err)
		}

		output, err := client.DescribeTaskDefinition(context.TODO(), &ecs.DescribeTaskDefinitionInput{
			TaskDefinition: services.Services[0].TaskDefinition,
		})
		if err != nil {
			log.Fatal(err)
		}

		var taskDefinition = output.TaskDefinition
		_, err = client.RegisterTaskDefinition(context.TODO(), &ecs.RegisterTaskDefinitionInput{
			Family:               taskDefinition.Family,
			ContainerDefinitions: taskDefinition.ContainerDefinitions,
			Volumes:              taskDefinition.Volumes,
			TaskRoleArn:          taskDefinition.TaskRoleArn,
			ExecutionRoleArn:     taskDefinition.ExecutionRoleArn,
		})
		if err != nil {
			log.Fatal(err)
		}
	},
}

func selectService(ctx context.Context, client *ecs.Client, cluster item) (*item, error) {
	if deployServiceId != "" {
		output, err := client.DescribeServices(ctx, &ecs.DescribeServicesInput{
			Cluster:  &cluster.arn,
			Services: []string{deployServiceId},
		})
		if err != nil {
			return nil, err
		}

		if len(output.Services) == 0 {
			return nil, fmt.Errorf("Service '%s' not found", deployServiceId)
		}

		task := output.Services[0]
		slices := strings.Split(*task.ServiceArn, "/")
		return &item{
			name: fmt.Sprintf("%s/%s", slices[1], slices[2]),
			arn:  *task.ServiceArn,
		}, nil
	}

	output, err := client.ListServices(ctx, &ecs.ListServicesInput{
		Cluster: &cluster.arn,
	})
	if err != nil {
		return nil, err
	}
	if len(output.ServiceArns) == 0 {
		return nil, errors.New("No services found")
	}

	items := []list.Item{}
	for _, arn := range output.ServiceArns {
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
}
