package selector

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/aws/aws-sdk-go-v2/service/ecs/types"
	"github.com/pterm/pterm"
)

func SelectService(ctx context.Context, client *ecs.Client, clusterId string, serviceId string) (*types.Service, error) {
	if serviceId == "" {
		listServices, err := client.ListServices(ctx, &ecs.ListServicesInput{
			Cluster: &clusterId,
		})
		if err != nil {
			return nil, err
		}
		serviceArn, err := pterm.DefaultInteractiveSelect.WithOptions(listServices.ServiceArns).Show("Service")
		if err != nil {
			return nil, err
		}
		serviceId = serviceArn
	}
	describeService, err := client.DescribeServices(ctx, &ecs.DescribeServicesInput{
		Cluster:  &clusterId,
		Services: []string{serviceId},
	})
	if err != nil {
		return nil, err
	}
	if len(describeService.Services) > 0 {
		return &describeService.Services[0], nil
	}
	return nil, fmt.Errorf("no service '%v' found", serviceId)
}
