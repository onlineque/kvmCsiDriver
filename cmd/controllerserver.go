package cmd

import (
	"github.com/onlineque/kvmCsiDriver/pkg/driver"
	"github.com/spf13/cobra"
	"log"
)

// controllerserverCmd represents the controllerserver command
var controllerserverCmd = &cobra.Command{
	Use:   "controllerserver",
	Short: "Starts controllerserver component of KVM CSI Driver",
	Long: `KVM CSI Driver ControllerServer component

ControllerServer watches the PVC is Kubernetes cluster and 
calls the CSI driver to Create and Publish the volume`,
	Run: func(cmd *cobra.Command, args []string) {
		log.Print("starting controllerServer...")
		driver.RunServer(true, false)
	},
}

func init() {
	rootCmd.AddCommand(controllerserverCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// controllerserverCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// controllerserverCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
