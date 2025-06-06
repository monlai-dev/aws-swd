package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "main",
	Short: "Microservices application",
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.AddCommand(customerCmd)
	rootCmd.AddCommand(driverCmd)
	//rootCmd.AddCommand(bookingCmd)
	// Add other service commands
}
