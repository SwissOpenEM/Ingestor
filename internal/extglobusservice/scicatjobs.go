package extglobusservice

import (
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"time"
)

// TODO: maybe make these structs public in globus-transfer-service and use them from there, as the code is duplicated from there ad-verbatim
type ScicatJobDatasetElement struct {
	Pid   string   `json:"pid"`
	Files []string `json:"files"`
}

type ScicatJobParams struct {
	DatasetList []ScicatJobDatasetElement `json:"datasetList"`
}

type GlobusTransferScicatJobResultObject struct {
	GlobusTaskId     string `json:"globusTaskId"`
	BytesTransferred uint   `json:"bytesTransferred"`
	FilesTransferred uint   `json:"filesTransferred"`
	FilesTotal       uint   `json:"filesTotal"`
	Completed        bool   `json:"completed"`
	Error            string `json:"error"`
}

type GlobusTransferScicatJob struct {
	CreatedBy       string                              `json:"createdBy"`
	UpdatedBy       string                              `json:"updatedBy"`
	CreatedAt       time.Time                           `json:"createdAt"`
	UpdatedAt       time.Time                           `json:"updatedAt"`
	OwnerGroup      string                              `json:"ownerGroup"`
	AccessGroups    []string                            `json:"accessGroups"`
	ID              string                              `json:"id"`
	OwnerUser       string                              `json:"ownerUser"`
	Type            string                              `json:"type"`
	StatusCode      string                              `json:"statusCode"`
	StatusMessage   string                              `json:"statusMessage"`
	JobParams       ScicatJobParams                     `json:"jobParams"`
	ContactEmail    string                              `json:"contactEmail"`
	ConfigVersion   string                              `json:"configVersion"`
	JobResultObject GlobusTransferScicatJobResultObject `json:"jobResultObject"`
}

func GetGlobusTransferJobsFromScicat(scicatUrl string, scicatToken string, ownerUser string) ([]GlobusTransferScicatJob, error) {
	url, err := url.JoinPath(scicatUrl, "api", "v4", "jobs")
	if err != nil {
		return []GlobusTransferScicatJob{}, err
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return []GlobusTransferScicatJob{}, err
	}

	// TODO: maybe add pagination or a limit to the filter
	q := req.URL.Query()
	q.Set("filter", `{"where":{"type":"globus_transfer_job","ownerUser":"`+ownerUser+`"}}`)
	req.URL.RawQuery = q.Encode()

	req.Header.Set("Authorization", "Bearer "+scicatToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return []GlobusTransferScicatJob{}, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return []GlobusTransferScicatJob{}, err
	}

	jobs := []GlobusTransferScicatJob{}
	err = json.Unmarshal(body, &jobs)
	return jobs, err
}
