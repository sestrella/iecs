package exec

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ecs/types"
	"github.com/sestrella/iecs/client"
	"github.com/sestrella/iecs/selector"
	"github.com/spf13/cobra"
)

const (
	execCommandFlag     = "command"
	execInteractiveFlag = "interactive"
)

type ExecSelection struct {
	cluster   *types.Cluster
	service   *types.Service
	task      *types.Task
	container *types.Container
}

var Cmd = &cobra.Command{
	Use:   "exec",
	Short: "Run a remote command on a container",
	Example: `
  aws-vault exec <profile> -- iecs exec [flags] (recommended)
  env AWS_PROFILE=<profile> iecs exec [flags]
  `,
	RunE: func(cmd *cobra.Command, args []string) error {
		command, err := cmd.Flags().GetString(execCommandFlag)
		if err != nil {
			return err
		}

		interactive, err := cmd.Flags().GetBool(execInteractiveFlag)
		if err != nil {
			return err
		}

		cfg, err := config.LoadDefaultConfig(context.TODO())
		if err != nil {
			return err
		}

		awsClient := client.NewClient(cfg)
		err = runExec(
			context.TODO(),
			awsClient,
			selector.NewSelectors(awsClient, cmd.Flag("theme").Value.String()),
			command,
			interactive,
		)
		if err != nil {
			return err
		}
		return nil
	},
	Aliases: []string{"ssh"},
}

func runExec(
	ctx context.Context,
	client client.Client,
	selectors selector.Selectors,
	command string,
	interactive bool,
) error {
	selection, err := execSelector(ctx, selectors)
	if err != nil {
		return err
	}
	cmd, err := client.ExecuteCommand(
		ctx,
		selection.cluster,
		*selection.task.TaskArn,
		selection.container,
		command,
		interactive,
	)
	if err != nil {
		return err
	}
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err = cmd.Start(); err != nil {
		return err
	}

	// Reference: https://github.com/kubernetes/kubectl/blob/master/pkg/util/interrupt/interrupt.go
	go func() {
		sigs := make(chan os.Signal, 1)
		signal.Notify(sigs, syscall.SIGHUP, syscall.SIGINT, syscall.SIGQUIT, syscall.SIGTERM)
		sig := <-sigs
		err = cmd.Process.Signal(sig)
		if err != nil {
			log.Fatal(err)
		}
	}()

	return cmd.Wait()
}

func execSelector(
	ctx context.Context,
	selectors selector.Selectors,
) (*ExecSelection, error) {
	cluster, err := selectors.Cluster(ctx)
	if err != nil {
		return nil, err
	}

	service, err := selectors.Service(ctx, cluster)
	if err != nil {
		return nil, err
	}

	task, err := selectors.Task(ctx, service)
	if err != nil {
		return nil, err
	}

	container, err := selectors.Container(ctx, task.Containers)
	if err != nil {
		return nil, err
	}

	return &ExecSelection{
		cluster:   cluster,
		service:   service,
		task:      task,
		container: container,
	}, nil
}

func init() {
	Cmd.Flags().StringP(execCommandFlag, "c", "/bin/bash", "command to run")
	Cmd.Flags().BoolP(execInteractiveFlag, "i", true, "toggles interactive mode")
}
