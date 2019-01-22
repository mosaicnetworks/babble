package commands

import (
	"strconv"

	runtime "github.com/mosaicnetworks/babble/cmd/network/lib"
	"github.com/spf13/cobra"
)

func NewLog() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "log",
		Short: "Show readtime logs for a node",
		RunE: func(cmd *cobra.Command, args []string) error {
			runtime.ReadLog(strconv.Itoa(config.Node))

			return nil
		},
	}

	AddLogFlags(cmd)

	return cmd
}

func AddLogFlags(cmd *cobra.Command) {
	cmd.Flags().IntVar(&config.Node, "node", config.Node, "Node index to connect to (starts from 0)")
}
