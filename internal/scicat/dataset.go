package scicat

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/paulscherrerinstitute/scicat-cli/datasetIngestor"
)

// clone function from scicat-cli, TODO: remove once scicat-cli is updated
func ReadMetadata(client *http.Client, APIServer string, metadatafile string, user map[string]string, accessGroups []string) (metaDataMap map[string]interface{}, sourceFolder string, beamlineAccount bool, err error) {
	metaDataMap, err = datasetIngestor.ReadMetadataFromFile(metadatafile)
	if err != nil {
		return nil, "", false, err
	}

	if keys := datasetIngestor.CollectIllegalKeys(metaDataMap); len(keys) > 0 {
		return nil, "", false, errors.New("illegal keys" + ": \"" + strings.Join(keys, "\", \"") + "\"")
	}

	beamlineAccount, err = datasetIngestor.CheckUserAndOwnerGroup(user, accessGroups, metaDataMap)
	if err != nil {
		return nil, "", false, err
	}

	err = datasetIngestor.GatherMissingMetadata(user, metaDataMap, client, APIServer, accessGroups)
	if err != nil {
		return nil, "", false, err
	}

	delete(metaDataMap, "sourceFolderHost")

	/*err = checkMetadataValidity(client, APIServer, metaDataMap, user["accessToken"])
	if err != nil {
		return nil, "", false, err
	}*/

	sourceFolder, err = datasetIngestor.GetSourceFolder(metaDataMap)
	if err != nil {
		return nil, "", false, err
	}

	return metaDataMap, sourceFolder, beamlineAccount, nil
}

// tied to CreateDataset, to be removed
func sendRequest(client *http.Client, method, url string, accessToken string, body []byte) (*http.Response, error) {
	req, err := http.NewRequest(method, url, bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", accessToken))

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

// tied to CreateDataset, to be removed
func decodePid(resp *http.Response) (string, error) {
	type PidType struct {
		Pid string `json:"pid"`
	}
	decoder := json.NewDecoder(resp.Body)
	var d PidType
	err := decoder.Decode(&d)
	if err != nil {
		return "", fmt.Errorf("could not decode pid from dataset entry: %v", err)
	}

	return d.Pid, nil
}

// temporary clone function, TODO: remove once scicat-cli is updated
func CreateDataset(client *http.Client, APIServer string, metaDataMap map[string]interface{}, user map[string]string) (string, error) {
	cmm, _ := json.Marshal(metaDataMap)
	datasetId := ""

	if _, ok := metaDataMap["type"]; !ok {
		return "", fmt.Errorf("no dataset type defined for dataset %v", metaDataMap)
	}

	resp, err := sendRequest(client, "POST", APIServer+"/datasets", user["accessToken"], cmm)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 200 || resp.StatusCode == 201 {
		datasetId, err = decodePid(resp)
		if err != nil {
			return "", err
		}
	} else {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return "", fmt.Errorf("SendIngestCommand: Failed to create new dataset: status code %v, status %s, unreadable body", resp.StatusCode, resp.Status)
		}
		return "", fmt.Errorf("SendIngestCommand: Failed to create new dataset: status code %v, status %s, body %s", resp.StatusCode, resp.Status, string(body))
	}

	return datasetId, nil
}