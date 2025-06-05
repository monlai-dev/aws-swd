package cmd

import (
	"github.com/spf13/cobra"
	"go.uber.org/fx"
	"microservice_swd_demo/bundlefx/customerfx"
)

func init() {
	customerCmd.Flags().IntP("port", "p", 8080, "Port for customer service")
}

var customerCmd = &cobra.Command{
	Use:   "customer",
	Short: "Start customer service",
	Run: func(cmd *cobra.Command, args []string) {
		port, _ := cmd.Flags().GetInt("port")
		startCustomerService(port)
	},
}

func startCustomerService(port int) {
	fx.New(
		fx.Provide(
			NewGinEngine,
			providePort(port),
		),
		customerfx.Module,
		fx.Invoke(StartHTTPServer),
	).Run()
}
