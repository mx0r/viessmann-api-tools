package main

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"time"
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

const IamBaseUrl = "https://iam.viessmann.com"
const CacheAuthTokenKey = "vs_auth_token"

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
	authorizeUrl := IamBaseUrl + "/idp/v2/authorize?response_type=code&client_id=" + context.ClientId + "&redirect_uri=" + context.RedirectUri + "&scope=IoT%20User&code_challenge=" + codeChallenge + "&code_challenge_method=S256"
	req, _ := http.NewRequest("POST", authorizeUrl, nil)
	req.SetBasicAuth(context.Username, context.Password)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := httpClient.Do(req)
	if err != nil {
		// if there is an error, return empty string
		return "", errors.New("error getting authorization code")
	}

	// otherwise, get location header and parse out "code" from it
	locationHeader := resp.Header.Get("Location")
	redirectUrl, _ := url.Parse(locationHeader)

	// get code from redirect URL
	return redirectUrl.Query().Get("code"), nil
}

func getAccessToken(httpClient http.Client, code string, context Context) (AccessTokenResponse, error) {
	tokenUrl := IamBaseUrl + "/idp/v2/token?grant_type=authorization_code&code_verifier=" + context.CodeVerifier + "&client_id=" + context.ClientId + "&redirect_uri=" + context.RedirectUri + "&code=" + code
	req, _ := http.NewRequest("POST", tokenUrl, nil)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := httpClient.Do(req)
	if err != nil {
		// if there is an error, return empty string
		return AccessTokenResponse{}, errors.New("error getting access token")
	}

	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			panic(err)
		}
	}(resp.Body)

	jsonBody, _ := io.ReadAll(resp.Body)
	accessTokenResponse := AccessTokenResponse{}
	jsonErr := json.Unmarshal(jsonBody, &accessTokenResponse)
	if jsonErr != nil {
		return AccessTokenResponse{}, jsonErr
	}

	return accessTokenResponse, nil
}

func performAuthorizationFlow(httpClient http.Client, context Context) (string, error) {

	if context.Cache != nil && context.Cache.Has(CacheAuthTokenKey) {
		// if cache is used and there is a cached token, retrieve it
		cachedTokenJson, _ := context.Cache.Get(CacheAuthTokenKey)
		cachedToken := AuthTokenCache{}
		jsonErr := json.Unmarshal([]byte(cachedTokenJson), &cachedToken)
		if jsonErr != nil {
			return "", jsonErr
		}

		if cachedToken.ExpiresAt > time.Now().Unix() {
			// when cached token is still valid, return it
			return cachedToken.AccessToken, nil
		}
	}

	// obtain authorize code
	code, cErr := getAuthorizeCode(httpClient, context)
	if cErr != nil {
		return "", cErr
	}

	// obtain access token
	accessTokenResponse, atErr := getAccessToken(httpClient, code, context)
	if atErr != nil {
		return "", atErr
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
		err := context.Cache.Set(CacheAuthTokenKey, string(cachedTokenJson))
		if err != nil {
			return "", err
		}
	}

	return accessTokenResponse.AccessToken, nil
}
