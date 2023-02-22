package main

import (
	"fmt"
	"os"

	"github.com/tendermint/tendermint/libs/log"

	"github.com/spf13/cobra"
	"github.com/tendermint/tendermint/pkg/remote"
)

const (
	pathFlag   = "path"
	listenFlag = "listen"
	portFlag   = "port"
)

func rootCMD() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "event-db [flags]",
		Short: "collects and serves arbitary json data as flat files",
	}
	cmd.AddCommand(startCMD())
	return cmd
}

func startCMD() *cobra.Command {
	command := &cobra.Command{
		Use:   "start [flags]",
		Short: "start the server",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			path, err := cmd.Flags().GetString(pathFlag)
			if err != nil {
				return err
			}

			listen, err := cmd.Flags().GetString(listenFlag)
			if err != nil {
				return err
			}

			port, err := cmd.Flags().GetString(portFlag)
			if err != nil {
				return err
			}

			logger := log.NewTMLogger(log.NewSyncWriter(os.Stdout))

			return remote.NewServer(path, logger).Start(fmt.Sprintf("%s:%s", listen, port))
		},
	}

	command.Flags().String(listenFlag, "0.0.0.0", "specify the listen address")
	command.Flags().String(portFlag, "25570", "specify the port")
	command.Flags().String(pathFlag, ".", "specify the path to the files")

	return command
}

func main() {
	r := rootCMD()
	if err := r.Execute(); err != nil {
		fmt.Println(fmt.Errorf("fatal error: %w", err))
	}
}
