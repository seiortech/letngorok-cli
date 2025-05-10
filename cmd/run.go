package cmd

import (
	"log"

	"github.com/seiortech/letngorok-cli/tunnel"
	sdk "github.com/seiortech/letngorok-go-sdk"
	"github.com/spf13/cobra"
)

func init() {
	var port string
	var token string

	var runCmd = &cobra.Command{
		Use:   "run",
		Short: "Start a tunnel to expose your local service",
		Example: `  # Expose port 8080 using saved token
  ngorok-cli run --port=8080

  # Expose port with specific token
  ngorok-cli run --port=3000 --token=your_token`,
		Run: func(cmd *cobra.Command, args []string) {
			if token == "" {
				loadedToken, err := LoadToken()
				if err != nil {
					log.Fatalf("Error loading token: %v", err)
				}

				if loadedToken == "" {
					log.Fatalf("No token provided. Either use --token flag or set a token with 'ngorok-cli set token YOUR_TOKEN'")
				}

				token = loadedToken
			}

			config := &sdk.DefaultSDKConfig

			config.OnConnected = tunnel.OnConnected
			config.OnDisconnected = tunnel.OnDisconnected
			config.OnError = tunnel.OnError
			config.OnRequest = tunnel.OnRequest
			config.OnSedingResponse = tunnel.OnSendingResponse
			config.OnAuth = tunnel.OnAuthenticated

			client, err := sdk.NewTunnelClient(&sdk.DefaultSDKConfig, token)
			if err != nil {
				log.Fatalln(err)
			}

			client.Start(port, nil)
		},
	}

	runCmd.Flags().StringVarP(&port, "port", "p", "8080", "Port to expose")
	runCmd.Flags().StringVarP(&token, "token", "t", "", "Letngorok token (overrides saved token)")

	rootCmd.AddCommand(runCmd)
}
