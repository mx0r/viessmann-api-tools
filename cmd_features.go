package main

import (
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/jxskiss/mcli"
)

func retrieveFeaturesJson(httpClient http.Client, accessToken string, context Context) (string, error) {
	featuresUrl := ApiBaseUrl + "/iot/v2/features/installations/" + context.InstallationId + "/gateways/" + context.GatewayId + "/devices/" + context.DeviceId + "/features"
	req, _ := http.NewRequest("GET", featuresUrl, nil)
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := httpClient.Do(req)
	if err != nil {
		// if there is an error, return empty string
		return "", errors.New("error getting features JSON")
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

func getFeaturesCommand() {
	var args struct {
		CommonOptions
		GatewayId      string `cli:"#R, -g, --gate, Gateway ID"`
		InstallationId string `cli:"#R, -i, --inst, Installation ID"`
		DeviceId       string `cli:"#O, -d, --dev, Device ID" default:"0"`
		RedirectUri    string `cli:"#O, -r, --redirect, Redirect URI" default:"http://localhost:4200/"`
	}

	_, err := mcli.Parse(&args)
	if err != nil {
		panic(err)
	}

	// prepare command
	context, httpClient, accessToken, err := prepareCommand(args.CommonOptions)

	if err != nil {
		panic(err)
	}

	// update context with additional information
	context.GatewayId = args.GatewayId
	context.InstallationId = args.InstallationId
	context.DeviceId = args.DeviceId

	// retrieve features
	featuresJson, err := retrieveFeaturesJson(*httpClient, accessToken, context)
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Println(featuresJson)
}
