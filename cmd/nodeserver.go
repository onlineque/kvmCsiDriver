/*
Copyright © 2024 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"github.com/onlineque/kvmCsiDriver/pkg/driver"
	"github.com/spf13/cobra"
	"log"
)

// nodeserverCmd represents the nodeserver command
var nodeserverCmd = &cobra.Command{
	Use:   "nodeserver",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {
		log.Print("starting nodeServer...")
		driver.RunNodeServer()
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
