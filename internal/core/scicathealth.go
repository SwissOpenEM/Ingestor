package core

import (
	"errors"
	"fmt"
	"net/http"
)

func ScicatHealthTest(APIServer string) error {
	// note: there's no function to use the /health endpoint in scicat-cli
	//   so here's a function that uses it.
	resp, err := http.DefaultClient.Get(APIServer + "/health")
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 503 {
		return errors.New("health check failed")
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d %s", resp.StatusCode, resp.Status)
	}

	return nil
}
