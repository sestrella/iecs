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

		cfg, err := config.LoadDefaultConfig(context.Background())
		if err != nil {
			return err
		}

		awsClient := client.NewClient(cfg)

		selector, err := newSelector(context.Background(), awsClient)
		if err != nil {
			return err
		}

		err = runExec(
			context.Background(),
			awsClient,
			selector,
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
	selector *Selector,
	command string,
	interactive bool,
) error {
	cmd, err := client.ExecuteCommand(
		ctx,
		selector.cluster,
		*selector.task.TaskArn,
		selector.container,
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

func init() {
	Cmd.Flags().StringP(execCommandFlag, "c", "/bin/bash", "command to run")
	Cmd.Flags().BoolP(execInteractiveFlag, "i", true, "toggles interactive mode")
}
