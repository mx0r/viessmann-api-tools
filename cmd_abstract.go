package main

import "net/http"

func prepareCommand(commonOptions CommonOptions) (Context, *http.Client, string, error) {
	// create base context
	context := Context{
		Username:     commonOptions.Username,
		Password:     commonOptions.Password,
		ClientId:     commonOptions.ClientId,
		RedirectUri:  commonOptions.RedirectUri,
		CodeVerifier: generateCodeVerifier(),
	}

	httpClient := &http.Client{
		// disable redirect following in http client
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	if commonOptions.UseCache {
		// when cache is used, create cache instance and assign it to context
		context.Cache = FileCache{Path: commonOptions.CachePath}
	}

	// perform authorization flow to get access token
	accessToken, err := performAuthorizationFlow(*httpClient, context)
	if err != nil {
		return Context{}, nil, "", err
	}

	return context, httpClient, accessToken, nil
}
