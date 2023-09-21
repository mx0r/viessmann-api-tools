package main

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"time"

	"github.com/jxskiss/mcli"
)

type AuthTokenCache struct {
	ClientId     string `json:"client_id"`
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresAt    int64  `json:"expires_at"`
}

type AccessTokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int64  `json:"expires_in"`
}

type Context struct {
	Username       string
	Password       string
	ClientId       string
	GatewayId      string
	InstallationId string
	DeviceId       string
	RedirectUri    string
	CodeVerifier   string
	Cache          Cache
}

const IAM_BASE_URL = "https://iam.viessmann.com"
const API_BASE_URL = "https://api.viessmann-platform.io"
const CACHE_AUTH_TOKEN_KEY = "vs_auth_token"

func generateRandomString(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[rand.Int63()%int64(len(letters))]
	}
	return string(b)
}

func generateCodeVerifier() string {
	return generateRandomString(12) + "-_" + generateRandomString(4) + "-" + generateRandomString(32)
}

func deriveCodeChallenge(codeVerifier string) string {
	s256 := sha256.Sum256([]byte(codeVerifier))
	return base64.RawURLEncoding.EncodeToString(s256[:])
}

func getAuthorizeCode(httpClient http.Client, context Context) (string, error) {
	codeChallenge := deriveCodeChallenge(context.CodeVerifier)
	authorizeUrl := IAM_BASE_URL + "/idp/v2/authorize?response_type=code&client_id=" + context.ClientId + "&redirect_uri=" + context.RedirectUri + "&scope=IoT%20User&code_challenge=" + codeChallenge + "&code_challenge_method=S256"
	req, _ := http.NewRequest("POST", authorizeUrl, nil)
	req.SetBasicAuth(context.Username, context.Password)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := httpClient.Do(req)
	if err != nil {
		// if there is an error, return empty string
		return "", errors.New("Error getting authorization code")
	}

	// otherwise, get location header and parse out "code" from it
	locationHeader := resp.Header.Get("Location")
	redirectUrl, _ := url.Parse(locationHeader)

	// get code from redirect URL
	return redirectUrl.Query().Get("code"), nil
}

func getAccessToken(httpClient http.Client, code string, context Context) (AccessTokenResponse, error) {
	tokenUrl := IAM_BASE_URL + "/idp/v2/token?grant_type=authorization_code&code_verifier=" + context.CodeVerifier + "&client_id=" + context.ClientId + "&redirect_uri=" + context.RedirectUri + "&code=" + code
	req, _ := http.NewRequest("POST", tokenUrl, nil)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := httpClient.Do(req)
	if err != nil {
		// if there is an error, return empty string
		return AccessTokenResponse{}, errors.New("Error getting access token")
	}

	defer resp.Body.Close()
	jsonBody, _ := io.ReadAll(resp.Body)
	accessTokenResponse := AccessTokenResponse{}
	json.Unmarshal(jsonBody, &accessTokenResponse)

	return accessTokenResponse, nil
}

func performAuthorizationFlow(httpClient http.Client, context Context) (string, error) {

	if context.Cache != nil && context.Cache.Has(CACHE_AUTH_TOKEN_KEY) {
		// if cache is used and there is a cached token, retrieve it
		cachedTokenJson, _ := context.Cache.Get(CACHE_AUTH_TOKEN_KEY)
		cachedToken := AuthTokenCache{}
		json.Unmarshal([]byte(cachedTokenJson), &cachedToken)

		if cachedToken.ExpiresAt > time.Now().Unix() {
			// when cached token is still valid, return it
			return cachedToken.AccessToken, nil
		}
	}

	// obtain authorize code
	code, c_err := getAuthorizeCode(httpClient, context)
	if c_err != nil {
		return "", c_err
	}

	// obtain access token
	accessTokenResponse, at_err := getAccessToken(httpClient, code, context)
	if at_err != nil {
		return "", at_err
	}

	if context.Cache != nil {
		// when cache is used, create cached token information
		cachedTokenJson, _ := json.Marshal(AuthTokenCache{
			ClientId:     context.ClientId,
			AccessToken:  accessTokenResponse.AccessToken,
			RefreshToken: "",
			ExpiresAt:    time.Now().Unix() + (accessTokenResponse.ExpiresIn - 60), // one minute less than returned
		})

		// store token in cache
		context.Cache.Set(CACHE_AUTH_TOKEN_KEY, string(cachedTokenJson))
	}

	return accessTokenResponse.AccessToken, nil
}

func retrieveFeaturesJson(httpClient http.Client, accessToken string, context Context) (string, error) {
	featuresUrl := API_BASE_URL + "/iot/v2/features/installations/" + context.InstallationId + "/gateways/" + context.GatewayId + "/devices/" + context.DeviceId + "/features"
	req, _ := http.NewRequest("GET", featuresUrl, nil)
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := httpClient.Do(req)
	if err != nil {
		// if there is an error, return empty string
		return "", errors.New("Error getting features JSON")
	}

	defer resp.Body.Close()
	jsonBody, _ := io.ReadAll(resp.Body)

	return string(jsonBody), nil
}

func getFeaturesCommand() {
	var args struct {
		Username       string `cli:"#R, -u, --user, Username"`
		Password       string `cli:"#R, -p, --pass, Password"`
		ClientId       string `cli:"#R, -c, --client, Client ID"`
		GatewayId      string `cli:"#R, -g, --gate, Gateway ID"`
		InstallationId string `cli:"#R, -i, --inst, Installation ID"`
		DeviceId       string `cli:"#O, -d, --dev, Device ID" default:"0"`
		RedirectUri    string `cli:"#O, -r, --redirect, Redirect URI" default:"http://localhost:4200/"`
		UseCache       bool   `cli:"#O, -C, --use-cache, Use cache" default:"false"`
		CachePath      string `cli:"#O, -P, --cache-path, Cache path" default:"/tmp"`
	}

	mcli.Parse(&args)

	httpClient := &http.Client{
		// disable redirect following in http client
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	// create context
	context := Context{
		Username:       args.Username,
		Password:       args.Password,
		ClientId:       args.ClientId,
		GatewayId:      args.GatewayId,
		InstallationId: args.InstallationId,
		DeviceId:       args.DeviceId,
		RedirectUri:    args.RedirectUri,
		CodeVerifier:   generateCodeVerifier(),
	}

	if args.UseCache {
		// when cache is used, create cache instance and assign it to context
		context.Cache = FileCache{Path: args.CachePath}
	}

	// perform authorization flow to get access token
	accessToken, err := performAuthorizationFlow(*httpClient, context)
	if err != nil {
		fmt.Println(err)
		return
	}

	// retrieve features
	featuresJson, err := retrieveFeaturesJson(*httpClient, accessToken, context)
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Println(featuresJson)
}

func main() {
	mcli.AddRoot(getFeaturesCommand)
	mcli.Run()
}
