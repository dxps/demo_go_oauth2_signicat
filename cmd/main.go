package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
)

const clientID = "dev-orange-sponge-701" // Your client ID
const clientSecret = "futi...THwB"       // Your client Secret.

func main() {

	fs := http.FileServer(http.Dir("webui"))
	http.Handle("/", fs)

	handleOIDCRedirect()
	handleUserInfo()

	_ = http.ListenAndServeTLS(":8087", "server.crt", "server.key", nil)
}

// Handle the redirect from OIDC Provider (OP) back to us. It contains the authorization `code`,
// plus other things, that is `scope`, `state` and `session_state`.
func handleOIDCRedirect() {

	http.HandleFunc("/oidc/authz-code", func(w http.ResponseWriter, r *http.Request) {

		// ----------------------------------------------------------
		// First, we need to get the value of the `code` query param.
		// ----------------------------------------------------------

		err := r.ParseForm()
		if err != nil {
			fmt.Fprintf(os.Stdout, "Could not parse URL query params: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		code := r.FormValue("code")
		fmt.Fprintf(os.Stdout, "[dbg] Got (authz) code: %v\n", code)

		// ----------------------------------------------------------------------
		// Second, we go back to OIDC OP to get the (id, access, refresh) tokens,
		// based on the received authorization code.
		// ----------------------------------------------------------------------

		body := url.Values{}
		body.Set("grant_type", "authorization_code")
		body.Set("code", code)
		body.Set("redirect_uri", "https://localhost:8087/oidc/authz-code")
		bodyReader := strings.NewReader(body.Encode())

		reqURL := "https://demo-signicat-oidc-go.sandbox.signicat.dev/auth/open/connect/token"
		req, err := http.NewRequest(http.MethodPost, reqURL, bodyReader)
		if err != nil {
			fmt.Fprintf(os.Stdout, "Could not create HTTP request: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		req.SetBasicAuth(clientID, clientSecret)
		req.Header.Set("Accept", "application/json")
		req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Add("Content-Length", strconv.Itoa(len(body.Encode())))
		// Perform the request.
		httpClient := http.Client{}
		resp, err := httpClient.Do(req)
		if err != nil {
			fmt.Fprintf(os.Stdout, "Could not send HTTP request: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			bodyBytes, err := io.ReadAll(resp.Body)
			if err != nil {
				fmt.Fprintf(os.Stdout, "Failed to get the body of the unsuccessful token response: %v", err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			fmt.Fprintf(os.Stdout, "Got unsuccessful response: %v", string(bodyBytes))
			return
		}

		// --------------------------------------------------------
		// Parse the response body into the `TokenResponse` struct.
		// --------------------------------------------------------

		var tr TokenResponse
		if err := json.NewDecoder(resp.Body).Decode(&tr); err != nil {
			fmt.Fprintf(os.Stdout, "Could not parse JSON based token response: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		fmt.Fprintf(os.Stdout, "[dbg] Got token response: %+v\n", tr)
		fmt.Fprintf(os.Stdout, "[dbg] Got token response body: %v\n", resp.Body)

		// -------------------------------------------------------------------
		// Finally, send a response to redirect the user to the "welcome" page
		// with the access token.
		// -------------------------------------------------------------------
		w.Header().Set("Location", "/welcome.html?access_token="+tr.AccessToken)
		w.WriteHeader(http.StatusFound)
	})
}

func handleUserInfo() {

	http.HandleFunc("/users/me", func(w http.ResponseWriter, r *http.Request) {
		// TODO
	})
}

type TokenResponse struct {
	AccessToken          string `json:"access_token"`
	RefreshToken         string `json:"refresh_token"`
	IdToken              string `json:"id_token"`
	AccessTokenExpiresIn int    `json:"expires_in"`
}
