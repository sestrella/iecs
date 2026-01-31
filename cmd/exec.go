package cmd

import (
	"context"
	"log"
	"os"
	"os/signal"
	"regexp"
	"syscall"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ecs/types"
	"github.com/sestrella/iecs/client"
	"github.com/sestrella/iecs/selector"
	"github.com/spf13/cobra"
)

type ExecSelection struct {
	cluster   *types.Cluster
	service   *types.Service
	task      *types.Task
	container *types.Container
}

var (
	execTaskStr        string
	execTaskRegex      *regexp.Regexp
	execContainerStr   string
	execContainerRegex *regexp.Regexp
	execCommand        string
	execInteractive    bool
)

var execCmd = &cobra.Command{
	Use:   "exec",
	Short: "Run a remote command on a container",
	Example: `
  aws-vault exec <profile> -- iecs exec [flags] (recommended)
  env AWS_PROFILE=<profile> iecs exec [flags]
  `,
	PreRunE: func(cmd *cobra.Command, args []string) error {
		if execTaskStr != "" {
			regex, err := regexp.Compile(execTaskStr)
			if err != nil {
				return err
			}

			execTaskRegex = regex
		}

		if execContainerStr != "" {
			regex, err := regexp.Compile(execContainerStr)
			if err != nil {
				return err
			}

			execContainerRegex = regex
		}

		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.LoadDefaultConfig(context.TODO())
		if err != nil {
			return err
		}

		awsClient := client.NewClient(cfg)

		selection, err := execSelector(
			context.TODO(),
			selector.NewSelectors(awsClient, *theme),
			rootClusterRegex,
			rootServiceRegex,
			execTaskRegex,
			execContainerRegex,
		)
		if err != nil {
			return err
		}

		err = runExec(
			context.TODO(),
			awsClient,
			*selection,
			execCommand,
			execInteractive,
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
	selection ExecSelection,
	command string,
	interactive bool,
) error {
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

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGHUP, syscall.SIGINT, syscall.SIGQUIT, syscall.SIGTERM)

	// Reference: https://github.com/kubernetes/kubectl/blob/master/pkg/util/interrupt/interrupt.go
	go func() {
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
	clusterRegex *regexp.Regexp,
	serviceRegex *regexp.Regexp,
	taskRegex *regexp.Regexp,
	containerRegex *regexp.Regexp,
) (*ExecSelection, error) {
	cluster, err := selectors.Cluster(ctx, clusterRegex)
	if err != nil {
		return nil, err
	}

	service, err := selectors.Service(ctx, cluster, serviceRegex)
	if err != nil {
		return nil, err
	}

	task, err := selectors.Task(ctx, service, taskRegex)
	if err != nil {
		return nil, err
	}

	container, err := selectors.Container(ctx, task.Containers, containerRegex)
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
	rootCmd.AddCommand(execCmd)

	execCmd.Flags().StringVar(&execTaskStr, "task", "", "A regex pattern for filtering tasks")
	execCmd.Flags().
		StringVar(&execContainerStr, "container", "", "A regex pattern for filtering containers")
	execCmd.Flags().StringVarP(&execCommand, "command", "c", "/bin/bash", "command to run")
	execCmd.Flags().BoolVarP(&execInteractive, "interactive", "i", true, "toggles interactive mode")
}
