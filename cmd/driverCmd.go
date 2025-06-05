package cmd

import (
	"github.com/spf13/cobra"
	"go.uber.org/fx"
	"microservice_swd_demo/bundlefx/driverfx"
)

func init() {
	driverCmd.Flags().IntP("port", "p", 8081, "Port for customer service")
}

var driverCmd = &cobra.Command{
	Use:   "driver",
	Short: "Start driver service",
	Run: func(cmd *cobra.Command, args []string) {
		port, _ := cmd.Flags().GetInt("port")
		startDriverService(port)
	},
}

func startDriverService(port int) {
	fx.New(
		fx.Provide(
			NewGinEngine,
			providePort(port),
		),
		driverfx.Module,
		fx.Invoke(StartHTTPServer),
	).Run()
}
