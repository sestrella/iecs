package selector

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/pterm/pterm"
)

type Client interface {
	DescribeClusters(ctx context.Context, input *ecs.DescribeClustersInput, options ...func(*ecs.Options)) (*ecs.DescribeClustersOutput, error)
	DescribeServices(ctx context.Context, input *ecs.DescribeServicesInput, options ...func(*ecs.Options)) (*ecs.DescribeServicesOutput, error)
	ListClusters(ctx context.Context, input *ecs.ListClustersInput, options ...func(*ecs.Options)) (*ecs.ListClustersOutput, error)
	ListServices(ctx context.Context, input *ecs.ListServicesInput, options ...func(*ecs.Options)) (*ecs.ListServicesOutput, error)
}

type DefaultSelector struct{}

type Selector interface {
	Select(title string, options []string) (string, error)
}

func (s *DefaultSelector) Select(title string, options []string) (string, error) {
	return pterm.DefaultInteractiveSelect.WithOptions(options).Show(title)
}
