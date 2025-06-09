package cmd

import (
	"github.com/spf13/cobra"
	"go.uber.org/fx"
	"microservice_swd_demo/bundlefx/notifyfx"
)

func init() {
	customerCmd.Flags().IntP("port", "p", 8082, "Port for notify service")
}

var notifyCmd = &cobra.Command{
	Use:   "notify",
	Short: "Start customer service",
	Run: func(cmd *cobra.Command, args []string) {
		port, _ := cmd.Flags().GetInt("port")
		startNotifyService(port)
	},
}

func startNotifyService(port int) {
	fx.New(
		fx.Provide(
			NewGinEngine,
			providePort(port),
		),
		notifyfx.Module,
		fx.Invoke(StartHTTPServer),
	).Run()
}
