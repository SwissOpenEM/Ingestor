package scicat

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/paulscherrerinstitute/scicat-cli/datasetIngestor"
)

// TODO: this file can be removed once scicat-cli is updated
// NOTE:

func createOrigBlock(start int, end int, filesArray []datasetIngestor.Datafile, datasetId string) (fileblock datasetIngestor.FileBlock) {
	// accumulate sizes
	var totalSize int64
	totalSize = 0
	for i := start; i < end; i++ {
		totalSize += filesArray[i].Size
	}

	return datasetIngestor.FileBlock{Size: totalSize, DataFileList: filesArray[start:end], DatasetId: datasetId}
}

func CreateOrigDatablocks(client *http.Client, APIServer string, fullFileArray []datasetIngestor.Datafile, datasetId string, user map[string]string) error {
	totalFiles := len(fullFileArray)

	if totalFiles > TOTAL_MAXFILES {
		return fmt.Errorf(
			"dataset exceeds the maximum number of files that can be handled by the archiving system per dataset (dataset: %v, max: %v)",
			totalFiles, TOTAL_MAXFILES)
	}

	end := 0
	var blockBytes int64
	for start := 0; end < totalFiles; {
		blockBytes = 0

		for end = start; end-start < BLOCK_MAXFILES && blockBytes < BLOCK_MAXBYTES && end < totalFiles; {
			blockBytes += fullFileArray[end].Size
			end++
		}
		origBlock := createOrigBlock(start, end, fullFileArray, datasetId)

		payloadString, _ := json.Marshal(origBlock)
		myurl := APIServer + "/OrigDatablocks"
		resp, err := sendRequest(client, "POST", myurl, user["accessToken"], payloadString)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		if resp.StatusCode != 200 && resp.StatusCode != 201 {
			return fmt.Errorf("unexpected response code \"%v\" when adding origDatablock for dataset id: \"%v\"", resp.Status, datasetId)
		}

		start = end
	}
	return nil
}
