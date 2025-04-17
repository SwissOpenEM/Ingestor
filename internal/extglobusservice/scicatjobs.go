package extglobusservice

import (
	"fmt"

	"github.com/SwissOpenEM/globus-transfer-service/jobs"
)

func GetGlobusTransferJobsFromScicat(scicatUrl string, scicatToken string, skip uint, limit uint) ([]jobs.ScicatJob, error) {
	filter := `{"where":{"type":"globus_transfer_job"}`
	if skip > 0 || limit > 0 {
		filter += `,"limits":{`
		if skip > 0 {
			filter += fmt.Sprintf(`"skip":%d`, skip)
			if limit > 0 {
				filter += ","
			}
		}
		if limit > 0 {
			filter += fmt.Sprintf(`"limit":%d`, limit)
		}
		filter += `}`
	}
	filter += `}`
	fmt.Printf("DEBUG LOL: %s\n", filter)
	return jobs.GetJobList(scicatUrl, scicatToken, filter)
}
