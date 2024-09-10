package core

import (
	"context"
	"fmt"

	"github.com/SwissOpenEM/globus"
	"golang.org/x/oauth2"
)

var globusClient globus.GlobusClient

func globusLogIn(gConfig GlobusTransferConfig) (globus.GlobusClient, error) {
	// config setup
	ctx := context.Background()
	clientConfig := globus.AuthGenerateOauthClientConfig(ctx, gConfig.ClientID, gConfig.ClientSecret, gConfig.RedirectURL, gConfig.Scopes)
	verifier := oauth2.GenerateVerifier()
	clientConfig.AuthCodeURL("state", oauth2.AccessTypeOffline, oauth2.S256ChallengeOption(verifier))

	// redirect user to consent page to ask for permission and obtain the code
	url := clientConfig.AuthCodeURL("state", oauth2.AccessTypeOffline, oauth2.S256ChallengeOption(verifier))
	fmt.Printf("Visit the URL for the auth dialog: %v\n\nEnter the received code here: ", url)

	// negotiate token and create client
	var code string
	if _, err := fmt.Scan(&code); err != nil {
		return globus.GlobusClient{}, err
	}
	tok, err := clientConfig.Exchange(ctx, code, oauth2.VerifierOption(verifier))
	if err != nil {
		return globus.GlobusClient{}, fmt.Errorf("oauth2 exchange failed: %v", err)
	}

	// return globus client
	return globus.HttpClientToGlobusClient(clientConfig.Client(ctx, tok)), nil
}

func GlobusTransfer(globusConf GlobusTransferConfig, datasetFolder string) error {
	var err error
	if !globusClient.IsClientSet() {
		globusClient, err = globusLogIn(globusConf)
		if err != nil {
			return fmt.Errorf("couldn't log in to globus: %v", err)
		}
	}

	globusClient.TransferFolderSync(globusConf.SourceCollection, datasetFolder, globusConf.DestinationCollection, "somepath", true)

	return nil
}
