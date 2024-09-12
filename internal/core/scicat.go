package core

import (
	"net/http"

	"github.com/paulscherrerinstitute/scicat-cli/datasetUtils"
)

func ScicatExtractUserInfo(httpClient *http.Client, apiServer string, token string) (map[string]string, []string) {
	if token != "" {
		user, accessGroups := datasetUtils.GetUserInfoFromToken(httpClient, apiServer, token)
		user["password"] = ""
		return user, accessGroups
	}

	return map[string]string{}, []string{}
}
