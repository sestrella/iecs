package selector

import (
	"context"
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go-v2/service/ecs/types"
	"github.com/charmbracelet/huh"
	"github.com/sestrella/iecs/client"
)

func serviceSelector(
	ctx context.Context,
	client client.Client,
	clusterArn string,
) (*types.Service, error) {
	serviceArns, err := client.ListServices(ctx, clusterArn)
	if err != nil {
		return nil, err
	}

	var selectedServiceArn string
	if len(serviceArns) == 1 {
		log.Printf("Pre-select the only available service")
		selectedServiceArn = serviceArns[0]
	} else {
		form := huh.NewForm(
			huh.NewGroup(
				huh.NewSelect[string]().
					Title("Select service").
					Options(huh.NewOptions(serviceArns...)...).
					Value(&selectedServiceArn).
					WithHeight(5),
			),
		)

		if err = form.Run(); err != nil {
			return nil, err
		}
	}

	service, err := client.DescribeService(ctx, clusterArn, selectedServiceArn)
	if err != nil {
		return nil, fmt.Errorf("failed to describe service after selection: %w", err)
	}

	fmt.Printf("%s %s\n", titleStyle.Render("Service:"), *service.ServiceArn)
	return service, nil
}
