package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	t "github.com/seiortech/letngorok-cli"
)

var (
	localPort    = flag.String("local-port", "", "Port for the local server (required)")
	tunnelServer = flag.String("server", "letngorok.web:9000", "Address of the tunnel server")
	authToken    = flag.String("token", "", "Authentication token for tunnel server (required for the first time or to update token)")
)

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExample:\n")
		fmt.Fprintf(os.Stderr, "%s -local-port=8080 -token=YOUR_AUTH_TOKEN [-server=letngorok.web:9000]\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "\nGet your token from the web dashboard after logging in.\n")
	}

	flag.Parse()

	if *localPort == "" {
		fmt.Fprintln(os.Stderr, "Error: -local-port is required")
		flag.Usage()
		os.Exit(1)
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Fatalln("Error getting home directory: ", err)
	}

	ngorokDir := filepath.Join(homeDir, ".ngorok")
	tokenFilePath := filepath.Join(ngorokDir, "auth.token")

	if *authToken == "" {
		// try to load the token from the local file
		tokenBytes, err := os.ReadFile(tokenFilePath)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error: No token provided and couldn't load from file. Please provide a token using -token flag")
			flag.Usage()
			os.Exit(1)
		}

		*authToken = strings.TrimSpace(string(tokenBytes))
		if *authToken == "" {
			fmt.Fprintln(os.Stderr, "Error: Stored token is empty. Please provide a new token using -token flag")
			flag.Usage()
			os.Exit(1)
		}
	}

	if *authToken != "" {
		// in mac or linux, save the file into ~/.ngorok/auth.token
		// in windows, save the file into C:\Users\<username>\.ngorok\auth.token
		if err := os.MkdirAll(ngorokDir, 0700); err != nil {
			log.Fatalln("Error creating .ngorok directory: ", err)
		}

		if err := os.WriteFile(tokenFilePath, []byte(*authToken), 0600); err != nil {
			log.Fatalf("Error saving token: %v", err)
		}
	}

	if _, err := strconv.Atoi(*localPort); err != nil {
		fmt.Fprintf(os.Stderr, "Error: Invalid -local-port: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Starting tunnel client to forward to localhost:%s\n", *localPort)
	fmt.Printf("Connecting to tunnel server at %s\n", *tunnelServer)

	conn, err := net.Dial("tcp", *tunnelServer)
	if err != nil {
		log.Fatalf("❌ Failed to connect to tunnel server: %v", err)
	}
	defer conn.Close()

	fmt.Printf("🔒 Authenticating with server...\n")
	encoder := json.NewEncoder(conn)
	decoder := json.NewDecoder(conn)

	// ===== AUTHENTICATION SECTION =====
	// TODO: add ssl later

	authReq := t.TunnelMessage{
		Type: t.TunnelAuthRequest,
		Body: *authToken,
	}

	if err := encoder.Encode(authReq); err != nil {
		log.Fatalf("❌ Failed to send authentication request: %v", err)
	}

	var tunnelMsg t.TunnelMessage
	conn.SetReadDeadline(time.Now().Add(15 * time.Second)) // e.g., 15 seconds
	if err := decoder.Decode(&tunnelMsg); err != nil {
		log.Fatalf("❌ Failed to receive authentication response: %v", err)
	}

	conn.SetReadDeadline(time.Time{})

	if tunnelMsg.Type == t.TunnelAuthFailure {
		log.Fatalf("❌ Tunnel Authentication Failed: %s", tunnelMsg.Body)
	}

	// ===== AUTHENTICATION SECTION =====

	if tunnelMsg.Type != t.TunnelCreated {
		log.Fatalf("❌ Expected TunnelCreated message after auth, got Type %d", tunnelMsg.Type)
	}

	localUrl := tunnelMsg.Headers["Local-URL"]
	prodUrl := tunnelMsg.Headers["Prod-URL"]
	tunnelID := tunnelMsg.ID // Get tunnel ID

	fmt.Println("\n✅ Tunnel established!")
	fmt.Printf("   Tunnel ID: %s\n", tunnelID)
	fmt.Printf("   🏠 Local Testing URL: %s\n", localUrl)
	fmt.Printf("   🌐 Production URL: %s\n", prodUrl)
	fmt.Printf("   📡 Forwarding traffic to http://localhost:%s\n\n", *localPort)

	handleTunnelRequests(conn, *localPort)

	log.Println("Client shutting down.")
}

func handleTunnelRequests(tunnel net.Conn, localPort string) {
	decoder := json.NewDecoder(tunnel)
	for {
		var msg t.TunnelMessage
		if err := decoder.Decode(&msg); err != nil {
			if err == io.EOF || strings.Contains(err.Error(), "use of closed network connection") {
				log.Println("Connection closed.") // More graceful message
			} else {
				log.Printf("Error decoding message: %v", err)
			}
			break
		}

		if msg.Type == t.TunnelRequest {
			go handleLocalRequest(tunnel, msg, localPort)
		} else {
			log.Printf("Received unexpected message type from tunnel: %d", msg.Type)
		}
	}
}

func handleLocalRequest(tunnel net.Conn, msg t.TunnelMessage, localPort string) {
	fmt.Printf("↘️  [%s] %s %s\n", msg.ID, msg.Method, msg.Path) // Log request ID

	targetURL := fmt.Sprintf("http://localhost:%s%s", localPort, msg.Path) // Renamed variable
	req, err := http.NewRequest(msg.Method, targetURL, strings.NewReader(msg.Body))
	if err != nil {
		log.Printf("Error creating request for %s: %v", targetURL, err)
		sendErrorResponse(tunnel, msg.ID, http.StatusInternalServerError, "Failed to create local request")
		return
	}

	for key, value := range msg.Headers {
		if strings.EqualFold(key, "Host") {
			continue
		}

		if strings.EqualFold(key, "X-Forwarded-Host") {
			req.Host = value
		}
		req.Header.Set(key, value)
	}

	if req.Host == "" {
		req.Host = "localhost:" + localPort
	}

	client := &http.Client{
		Timeout: 20 * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
			log.Printf("Timeout connecting to local service %s: %v", targetURL, err)
			sendErrorResponse(tunnel, msg.ID, http.StatusGatewayTimeout, "Local service timed out")
		} else {
			log.Printf("Error connecting to local service %s: %v", targetURL, err)
			sendErrorResponse(tunnel, msg.ID, http.StatusBadGateway, "Failed to connect to local service")
		}
		return
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Error reading response body from %s: %v", targetURL, err)
		sendErrorResponse(tunnel, msg.ID, http.StatusInternalServerError, "Failed to read local response body")
		return
	}

	fmt.Printf("↖️  [%s] %d %s [%d bytes]\n", msg.ID, resp.StatusCode, msg.Path, len(bodyBytes)) // Log request ID

	responseHeaders := make(map[string]string)
	for key, values := range resp.Header {
		if len(values) > 0 {
			responseHeaders[key] = values[0]
		}
	}

	if _, ok := responseHeaders["Content-Type"]; !ok && len(bodyBytes) > 0 {
		// Default to application/octet-stream or try to detect?
		// For simplicity, let's rely on the local server setting it.
	}

	responseHeaders["X-Status-Code"] = strconv.Itoa(resp.StatusCode)

	responseMsg := t.TunnelMessage{
		Type:    t.TunnelResponse,
		ID:      msg.ID,
		Headers: responseHeaders,
		Body:    string(bodyBytes),
	}

	encoder := json.NewEncoder(tunnel)
	if err := encoder.Encode(responseMsg); err != nil {
		if strings.Contains(err.Error(), "broken pipe") || strings.Contains(err.Error(), "connection reset by peer") {
			log.Printf("Client disconnected before response for request %s could be sent.", msg.ID)
		} else {
			log.Printf("Error sending response for request %s through tunnel: %v", msg.ID, err)
		}
	}
}

func sendErrorResponse(tunnel net.Conn, requestID string, statusCode int, message string) {
	if statusCode < 100 || statusCode > 599 {
		statusCode = http.StatusInternalServerError
	}

	responseMsg := t.TunnelMessage{
		Type: t.TunnelResponse,
		ID:   requestID,
		Headers: map[string]string{
			"X-Status-Code": strconv.Itoa(statusCode),
			"Content-Type":  "text/plain; charset=utf-8",
		},
		Body: fmt.Sprintf("%d %s: %s", statusCode, http.StatusText(statusCode), message),
	}

	encoder := json.NewEncoder(tunnel)
	log.Printf("Sending Error Response for %s: %d %s", requestID, statusCode, message)
	if err := encoder.Encode(responseMsg); err != nil {
		log.Printf("Error sending error response for request %s: %v", requestID, err)
	}
}
