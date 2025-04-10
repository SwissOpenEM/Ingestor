package extglobusservice

import "github.com/SwissOpenEM/globus-transfer-service/jobs"

func GetGlobusTransferJobsFromScicat(scicatUrl string, scicatToken string, ownerUser string) ([]jobs.ScicatJob, error) {
	filter := `{"where":{"type":"globus_transfer_job","ownerUser":"` + ownerUser + `"}}`
	return jobs.GetJobList(scicatUrl, scicatToken, filter)
}
