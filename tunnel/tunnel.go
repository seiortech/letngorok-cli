package tunnel

import (
	"fmt"
	"net/http"

	sdk "github.com/seiortech/letngorok-go-sdk"
)

// var onConnectedMessageFormat = `
// ✅ Tunnel established!
//       Tunnel ID: %s
//       🏠 Local Testing URL: %s
//       🌐 Production URL: %s
//       📡 Forwarding traffic to %s
// `

var onConnectedMessageFormat = `
✅ Tunnel established!
      Tunnel ID: %s
      🌐 Production URL: %s
      📡 Forwarding traffic from %s
`

var OnConnected = func(localPort, localUrl, prodUrl, tunnelId string) {
	fmt.Printf(onConnectedMessageFormat, tunnelId, prodUrl, localPort)
}

var OnDisconnected = func() {
	fmt.Println("[Ngorok] ❌ Tunnel disconnected")
}

var OnAuthenticated = func(_ string) {
	fmt.Println("🔒 Authenticating with server...")
}

var OnError = func(err error) {
	fmt.Println("[Ngorok] ❌ Error:", err)
}

var OnRequest = func(msg sdk.TunnelMessage) {
	fmt.Printf("[Ngorok] ↙️ Received request: [%s] %s %s\n", msg.ID, msg.Method, msg.Path)
}

var OnSendingResponse = func(msg sdk.TunnelMessage, resp *http.Response, body []byte) {
	fmt.Printf("[Ngorok] ↗️ Sending response: [%s] %d %s [%d bytes]\n", msg.ID, resp.StatusCode, msg.Path, len(body))
}
