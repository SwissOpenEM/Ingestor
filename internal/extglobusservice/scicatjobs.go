package extglobusservice

import (
	"fmt"

	"github.com/SwissOpenEM/globus-transfer-service/jobs"
)

func GetGlobusTransferJobsFromScicat(scicatUrl string, scicatToken string, skip uint, limit uint) ([]jobs.ScicatJob, uint, error) {
	jobListFilter := `{"where":{"type":"globus_transfer_job"}`
	if skip > 0 || limit > 0 {
		jobListFilter += `,"limits":{`
		if skip > 0 {
			jobListFilter += fmt.Sprintf(`"skip":%d`, skip)
			if limit > 0 {
				jobListFilter += ","
			}
		}
		if limit > 0 {
			jobListFilter += fmt.Sprintf(`"limit":%d`, limit)
		}
		jobListFilter += `}`
	}
	jobListFilter += `}`
	jobsResult, err := jobs.GetJobList(scicatUrl, scicatToken, jobListFilter)
	if err != nil {
		return []jobs.ScicatJob{}, 0, err
	}

	totalJobs, err := jobs.GetJobCount(scicatUrl, scicatToken, `{"type":"globus_transfer_job"}`)
	return jobsResult, totalJobs, err
}
