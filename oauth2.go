package gcputil

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"golang.org/x/oauth2"
)

// GetClientFromTokenOrWeb retrieves a token from a local file or requests a new one.
func GetClientFromTokenOrWeb(localTokenFilename string, credentialsConfig *oauth2.Config) (*http.Client, error) {
	tok, err := GetTokenFromFile(localTokenFilename)
	if err != nil {
		// Start browser-based flow if no local token exists
		tok = GetTokenFromWeb(credentialsConfig)
		err = SaveTokenFile(localTokenFilename, tok)
	}
	return credentialsConfig.Client(context.Background(), tok), err
}

// GetTokenFromWeb requests a token by prompting the user to visit a URL.
func GetTokenFromWeb(config *oauth2.Config) *oauth2.Token {
	// 1. Create a channel to receive the code
	codeCh := make(chan string)

	// 2. Start a temporary local server
	server := &http.Server{Addr: ":8080"}
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		code := r.URL.Query().Get("code")
		if code != "" {
			w.Write([]byte("Authentication successful! You can close this tab."))
			codeCh <- code // Send code back to main thread
		}
	})

	go func() {
		if err := server.ListenAndServe(); err != http.ErrServerClosed {
			log.Fatalf("HTTP server failed: %v", err)
		}
	}()

	// 3. Open browser for the user
	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	fmt.Printf("Opening browser for authentication...\n%v\n", authURL)
	// (Optional: use an 'open' library to trigger the browser automatically)

	// 4. Wait for the code and shut down
	authCode := <-codeCh
	server.Shutdown(context.Background())

	// 5. Exchange code for token
	tok, _ := config.Exchange(context.TODO(), authCode)
	return tok
}

// GetTokenFromFile retrieves a token from a local file.
func GetTokenFromFile(filename string) (*oauth2.Token, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	tok := &oauth2.Token{}
	err = json.NewDecoder(f).Decode(tok)
	return tok, err
}

// SaveTokenFile saves a token to a local file path.
func SaveTokenFile(path string, token *oauth2.Token) error {
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	defer f.Close()
	json.NewEncoder(f).Encode(token)
	return nil
}
