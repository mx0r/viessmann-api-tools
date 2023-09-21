package main

import (
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/jxskiss/mcli"
)

func retrieveInstallationsJson(httpClient http.Client, accessToken string, context Context) (string, error) {
	featuresUrl := API_BASE_URL + "/iot/v1/equipment/installations?includeGateways=true"
	req, _ := http.NewRequest("GET", featuresUrl, nil)
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := httpClient.Do(req)
	if err != nil {
		// if there is an error, return empty string
		return "", errors.New("Error getting installations JSON")
	}

	defer resp.Body.Close()
	jsonBody, _ := io.ReadAll(resp.Body)

	return string(jsonBody), nil
}

func getInstallationsCommand() {
	var args struct {
		CommonOptions
	}

	mcli.Parse(&args)

	// prepare command
	context, httpClient, accessToken, err := prepareCommand(args.CommonOptions)

	if err != nil {
		panic(err)
	}

	// retrieve features
	installationsJson, err := retrieveInstallationsJson(*httpClient, accessToken, context)
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Println(installationsJson)
}
