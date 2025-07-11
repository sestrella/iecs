package cmd

import (
	"context"
	"log"
	"os"
	"os/exec"
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

type SelectedContainer struct {
	Cluster   *types.Cluster
	Service   *types.Service
	Task      *types.Task
	Container *types.Container
}

var execCmd = &cobra.Command{
	Use:   "exec",
	Short: "Run a remote command on a container",
	Example: `
  aws-vault exec <profile> -- iecs exec [flags] (recommended)
  env AWS_PROFILE=<profile> iecs exec [flags]
  `,
	RunE: func(cmd *cobra.Command, args []string) error {
		theme, err := themeByName(themeName)
		if err != nil {
			return err
		}

		smpPath, err := exec.LookPath("session-manager-plugin")
		if err != nil {
			return err
		}
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
			smpPath,
			awsClient,
			exec.Command,
			selector.NewSelectors(awsClient, *theme),
			cfg.Region,
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
	smpPath string,
	client client.Client,
	commandExecutor func(name string, arg ...string) *exec.Cmd,
	selectors selector.Selectors,
	region string,
	command string,
	interactive bool,
) error {
	selection, err := containerSelector(ctx, selectors)
	if err != nil {
		return err
	}
	cmd, err := client.ExecuteCommand(
		ctx,
		selection.Cluster,
		*selection.Task.TaskArn,
		selection.Container,
		command,
		interactive,
	)
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

func containerSelector(
	ctx context.Context,
	selectors selector.Selectors,
) (*SelectedContainer, error) {
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

	container, err := selectors.Container(ctx, task)
	if err != nil {
		return nil, err
	}

	return &SelectedContainer{
		Cluster:   cluster,
		Service:   service,
		Task:      task,
		Container: container,
	}, nil
}

func init() {
	rootCmd.AddCommand(execCmd)

	execCmd.Flags().StringP(execCommandFlag, "c", "/bin/bash", "command to run")
	execCmd.Flags().BoolP(execInteractiveFlag, "i", true, "toggles interactive mode")
}
