package main

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/sestrella/iecs/cmd"
	"github.com/sestrella/iecs/selector"
)

func main() {
	if true {
		cfg, err := config.LoadDefaultConfig(context.TODO())
		if err != nil {
			panic(err)
		}
		client := ecs.NewFromConfig(cfg)
		result, err := selector.RunHuhForm(context.TODO(), client)
		if err != nil {
			panic(err)
		}
		fmt.Print(result)
	} else {
		cmd.Execute()
	}
}
