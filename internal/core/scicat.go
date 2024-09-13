package core

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
)

type userInfo struct {
	CurrentUser      string   `json:"currentUser"`
	CurrentUserEmail string   `json:"currentUserEmail"`
	CurrentGroups    []string `json:"currentGroups"`
}

// this function is temporary until datasetUtils.GetUserInfoFromToken is fixed in scicat-cli
// TODO; delete this once scicat-cli is updated with fixes
func getUserInfoFromToken(client *http.Client, APIServer string, token string) (map[string]string, []string, error) {
	u := make(map[string]string)
	var accessGroups []string

	url := APIServer + "/Users/userInfos?access_token=" + token
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return map[string]string{}, []string{}, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return map[string]string{}, []string{}, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return map[string]string{}, []string{}, err
	}

	if resp.StatusCode != 200 {
		return map[string]string{}, []string{}, fmt.Errorf("could not login with the current token, status code: \"%v\", status: \"%s\", body: \"%v\"", resp.StatusCode, resp.Status, string(body[:]))
	}

	var respObj userInfo
	err = json.Unmarshal(body, &respObj)
	if err != nil {
		log.Fatal(err)
	}

	if respObj.CurrentUser != "" {
		//log.Printf("Found the following user for this token %v", respObj[0])
		u["username"] = respObj.CurrentUser
		u["mail"] = respObj.CurrentUserEmail
		u["displayName"] = respObj.CurrentUser
		u["accessToken"] = token
		log.Printf("User authenticated: %s %s\n", u["displayName"], u["mail"])
		accessGroups = respObj.CurrentGroups
		log.Printf("User is member in following groups: %v\n", accessGroups)
	} else {
		log.Fatalf("Could not map a user to the token %v", token)
	}
	return u, accessGroups, nil
}

func ScicatExtractUserInfo(httpClient *http.Client, apiServer string, token string) (map[string]string, []string, error) {
	if token == "" {
		return map[string]string{}, []string{}, fmt.Errorf("scicat: no access token was provided")
	}

	// use the internal function copy with fixes until upstream scicat-cli is fixed
	// TODO change this to datasetUtils.GetUserInfoFromToken when fixes are merged
	user, accessGroups, err := getUserInfoFromToken(httpClient, apiServer, token)
	if err != nil {
		return map[string]string{}, []string{}, fmt.Errorf("scicat: couldn't get user info from token: %v", err)
	}
	user["password"] = ""
	return user, accessGroups, nil
}
