package cmd

import (
	"github.com/onlineque/kvmCsiDriver/pkg/driver"
	"github.com/spf13/cobra"
	"log"
)

// nodeserverCmd represents the nodeserver command
var nodeserverCmd = &cobra.Command{
	Use:   "nodeserver",
	Short: "Starts nodeserver component of KVM CSI Driver",
	Long: `KVM CSI Driver NodeServer component

NodeServer publishes and unpublishes created volume to/from pods`,
	Run: func(cmd *cobra.Command, args []string) {
		log.Print("starting nodeServer...")
		driver.RunServer(false, true)
	},
}

func init() {
	rootCmd.AddCommand(nodeserverCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// nodeserverCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// nodeserverCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
