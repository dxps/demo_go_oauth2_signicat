package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
)

const clientID = "dev-orange-sponge-701"                                // Your client ID
const clientSecret = "yn3tG1oUA21a0Sko7zihafMS1ldSo564MAHyi3LLIAX3STSa" // Your client Secret.

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
			fmt.Fprintf(os.Stdout, "Could not parse URL query params: %v\n", err)
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
			fmt.Fprintf(os.Stdout, "Could not create HTTP request: %v\n", err)
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
			fmt.Fprintf(os.Stdout, "Could not send HTTP request to get the tokens: %v\n", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			bodyBytes, err := io.ReadAll(resp.Body)
			if err != nil {
				fmt.Fprintf(os.Stdout, "Failed to get the body of the unsuccessful token response: %v\n", err)
				body, _ := io.ReadAll(resp.Body)
				fmt.Fprintf(os.Stdout, "The unsuccessful token response body: %v\n", string(body))
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			fmt.Fprintf(os.Stdout, "Got unsuccessful token response code: %v body: %v\n", resp.StatusCode, string(bodyBytes))
			return
		}

		// --------------------------------------------------------
		// Parse the response body into the `TokenResponse` struct.
		// --------------------------------------------------------

		var tr TokenResponse
		if err := json.NewDecoder(resp.Body).Decode(&tr); err != nil {
			fmt.Fprintf(os.Stdout, "Could not parse JSON based token response: %v\n", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		fmt.Fprintf(os.Stdout, "[dbg] Got token response: %+v\n", tr)

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

		token, err := getBearerAuthHeader(r.Header.Get("Authorization"))
		if err != nil {
			msg := fmt.Sprintf("Failed to get bearer token from request: %v\n", err)
			fmt.Fprintln(os.Stdout, msg)
			bs, _ := json.Marshal(MyUserInfoErrorResponse{Error: msg})
			_, _ = w.Write(bs)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		fmt.Fprintf(os.Stdout, "[dbg] On '/users/me' got token: %v\n", *token)

		reqURL := "https://demo-signicat-oidc-go.sandbox.signicat.dev/auth/open/connect/userinfo"
		req, err := http.NewRequest(http.MethodPost, reqURL, nil)
		if err != nil {
			fmt.Fprintf(os.Stdout, "Could not create HTTP request: %v\n", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		req.Header.Set("Accept", "application/json")
		req.Header.Add("Authorization", "Bearer "+*token)
		// Perform the request.
		httpClient := http.Client{}
		resp, err := httpClient.Do(req)
		if err != nil {
			fmt.Fprintf(os.Stdout, "Could not send HTTP request to userinfo: %v\n", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			bodyBytes, err := io.ReadAll(resp.Body)
			if err != nil {
				fmt.Fprintf(os.Stdout, "Failed to get the body of the unsuccessful token response: %v\n", err)
				fmt.Fprintf(os.Stdout, "The unsuccessful token response body: %v\n", resp.Body)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			fmt.Fprintf(os.Stdout, "Got unsuccessful userinfo response code: %v body: %v\n", resp.StatusCode, string(bodyBytes))
			return
		}

		// --------------------------------------------------------
		// Parse the response body into the `UserInfoResponse` struct.
		// --------------------------------------------------------

		var uir UserInfoResponse
		if err := json.NewDecoder(resp.Body).Decode(&uir); err != nil {
			fmt.Fprintf(os.Stdout, "Could not parse JSON based userinfo response: %v\n", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		uirBytes, err := json.Marshal(uir)
		if err != nil {
			fmt.Fprintf(os.Stdout, "Failed to marshal the userinfo response: %v\n", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		_, _ = w.Write(uirBytes)
		w.WriteHeader(http.StatusOK)
	})
}

// getBearerAuthHeader validates incoming "Authorization header
// and returns the token, otherwise an empty string.
func getBearerAuthHeader(authHeader string) (*string, error) {

	if authHeader == "" {
		return nil, errors.New("The header value is empty")
	}

	parts := strings.Split(authHeader, "Bearer")
	if len(parts) != 2 {
		return nil, errors.New("The header value does not starts with Bearer")
	}

	token := strings.TrimSpace(parts[1])
	if len(token) < 1 {
		return nil, errors.New("The header value does not include anything besides the Bearer")
	}

	return &token, nil
}

type TokenResponse struct {
	AccessToken          string `json:"access_token"`
	RefreshToken         string `json:"refresh_token"`
	IdToken              string `json:"id_token"`
	AccessTokenExpiresIn int    `json:"expires_in"`
}

type UserInfoResponse struct {
	Subject   string `json:"sub"`
	GivenName string `json:"given_name"`
	Name      string `json:"name"`
	Email     string `json:"email"`
}

type MyUserInfoErrorResponse struct {
	Error string `json:"error"`
}
