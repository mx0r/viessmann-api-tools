package main

import (
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/jxskiss/mcli"
)

func retrieveInstallationsJson(httpClient http.Client, accessToken string) (string, error) {
	featuresUrl := ApiBaseUrl + "/iot/v1/equipment/installations?includeGateways=true"
	req, _ := http.NewRequest("GET", featuresUrl, nil)
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := httpClient.Do(req)
	if err != nil {
		// if there is an error, return empty string
		return "", errors.New("error getting installations JSON")
	}

	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			panic(err)
		}
	}(resp.Body)
	jsonBody, _ := io.ReadAll(resp.Body)

	return string(jsonBody), nil
}

func getInstallationsCommand() {
	var args struct {
		CommonOptions
	}

	_, err := mcli.Parse(&args)
	if err != nil {
		panic(err)
	}

	// prepare command
	_, httpClient, accessToken, err := prepareCommand(args.CommonOptions)

	if err != nil {
		panic(err)
	}

	// retrieve features
	installationsJson, err := retrieveInstallationsJson(*httpClient, accessToken)
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Println(installationsJson)
}
