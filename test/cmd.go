package test

import "github.com/spf13/cobra"

var (
	targetServerAddr  string
	urlForNatGwServer string
)

const (
	DefaultTargetServerAddr  = ":9910"
	DefaultUrlForNatGwServer = "http://127.0.0.1:9920"
)

var TestCmd = &cobra.Command{
	Use:   "test",
	Short: "testing cmd",
	Long:  `testing cmd`,
	Run: func(cmd *cobra.Command, args []string) {

	},
}

var localServerCmd = &cobra.Command{
	Use:   "local-server",
	Short: "start local server",
	Long:  `start local server`,
	Run: func(cmd *cobra.Command, args []string) {
		runLocalServer()
	},
}

var sendCmd = &cobra.Command{
	Use:   "send",
	Short: "send msg to nat gw",
	Long:  `send msg to nat gw`,
	Run: func(cmd *cobra.Command, args []string) {
		sendToNatGwServer()
	},
}

func init() {
	TestCmd.AddCommand(localServerCmd)
	TestCmd.AddCommand(sendCmd)

	localServerCmd.PersistentFlags().StringVarP(&targetServerAddr, "address", "", "", "target server address")
	sendCmd.PersistentFlags().StringVarP(&urlForNatGwServer, "url", "", "", "url for nat gw serverinput")

}
