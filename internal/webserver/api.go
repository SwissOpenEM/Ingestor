//go:build go1.22

//go:generate go run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen --config=cfg.yaml ../../api/openapi.yaml
//go:generate go run github.com/swaggo/swag/cmd/swag init -g api.go -o ../../docs
package webserver

import (
	"context"
	"crypto/cipher"
	"encoding/json"
	"fmt"
	"math"
	"net/url"
	"os"

	"github.com/SwissOpenEM/Ingestor/internal/core"
	"github.com/google/uuid"
	"golang.org/x/oauth2"
)

var _ StrictServerInterface = (*IngestorWebServerImplemenation)(nil)

type IngestorWebServerImplemenation struct {
	version      string
	taskQueue    *core.TaskQueue
	oauth2Config *oauth2.Config
	cookieCipher cipher.Block
}

//	@contact.name	SwissOpenEM
//	@contact.url	https://swissopenem.github.io
//	@contact.email	spencer.bliven@psi.ch

// @license.name	Apache 2.0
// @license.url	http://www.apache.org/licenses/LICENSE-2.0.html

func NewIngestorWebServer(version string, taskQueue *core.TaskQueue, oauthConf *oauth2.Config, cookieCipher cipher.Block) *IngestorWebServerImplemenation {
	return &IngestorWebServerImplemenation{version: version, taskQueue: taskQueue, oauth2Config: oauthConf, cookieCipher: cookieCipher}
}

// DatasetControllerIngestDataset implements ServerInterface.
//
//	@Description	Ingest a new dataset
//	@Tags			datasets
//	@Accept			json
//	@Produce		json
//
//	@Router			/datasets [post]
func (i *IngestorWebServerImplemenation) DatasetControllerIngestDataset(ctx context.Context, request DatasetControllerIngestDatasetRequestObject) (DatasetControllerIngestDatasetResponseObject, error) {
	// get sourcefolder from metadata
	metadataString := *request.Body.MetaData
	var metadata map[string]interface{}
	err := json.Unmarshal([]byte(metadataString), &metadata)
	if err != nil {
		return DatasetControllerIngestDataset400TextResponse(err.Error()), nil
	}

	// create and start task
	id := uuid.New()
	err = i.taskQueue.CreateTaskFromMetadata(id, metadata)
	if err != nil {
		if _, ok := err.(*os.PathError); ok {
			return nil, fmt.Errorf("could not create the task due to a path error: %s", err.Error())
		} else {
			return DatasetControllerIngestDataset400TextResponse("You don't have the right to create the task"), nil
		}
	}
	i.taskQueue.ScheduleTask(id)

	// NOTE: because of the way the tasks are created, right now it'll search for a metadata.json
	//   in the dataset folder to get the metadata, we can't pass on the one we got through this
	//   request
	// TODO: change this so that a task will accept a struct containing the dataset
	status := "started"
	idString := id.String()
	return DatasetControllerIngestDataset200JSONResponse{
		IngestId: &idString,
		Status:   &status,
	}, nil
}

// OtherControllerGetVersion implements ServerInterface.
//
//	@Description	Get the used ingestor version
//	@Tags			other
//	@Accept			json
//	@Produce		json
//
//	@Router			/version [get]
func (i *IngestorWebServerImplemenation) OtherControllerGetVersion(ctx context.Context, request OtherControllerGetVersionRequestObject) (OtherControllerGetVersionResponseObject, error) {
	return OtherControllerGetVersion200JSONResponse{
		Version: &i.version,
	}, nil
}

// TransferControllerDeleteTransfer implements ServerInterface.
//
//	@Description	Cancel a data transfer
//	@Tags			transfer
//	@Accept			json
//	@Produce		json
//
//	@Router			/transfer [delete]
func (i *IngestorWebServerImplemenation) TransferControllerDeleteTransfer(ctx context.Context, request TransferControllerDeleteTransferRequestObject) (TransferControllerDeleteTransferResponseObject, error) {
	if request.Body.IngestId == nil {
		return TransferControllerDeleteTransfer400TextResponse("Ingest ID was not specified in the request"), nil
	}

	id := *request.Body.IngestId
	uuid, err := uuid.Parse(id)
	if err != nil {
		return TransferControllerDeleteTransfer400TextResponse(fmt.Sprintf("Ingest ID '%s' could not be parsed as uuid: %s", id, err.Error())), nil
	}

	err = i.taskQueue.RemoveTask(uuid)
	if err != nil {
		return TransferControllerDeleteTransfer400TextResponse(err.Error()), nil
	}

	status := "gone"
	return TransferControllerDeleteTransfer200JSONResponse{
		IngestId: &id,
		Status:   &status,
	}, nil
}

// TransferControllerGetTransfer implements ServerInterface.
//
//	@Description	Get list of transfers. Optional use the transferId parameter to only get one item.
//	@Tags			transfer
//	@Accept			json
//	@Produce		json
//	@param			params	path	TransferControllerGetTransferParams	true	"params"
//
//	@Router			/transfer [get]
func (i *IngestorWebServerImplemenation) TransferControllerGetTransfer(ctx context.Context, request TransferControllerGetTransferRequestObject) (TransferControllerGetTransferResponseObject, error) {
	scopes := ctx.Value(OpenIDScopes)
	fmt.Println("scopes: ", scopes)

	if request.Params.TransferId != nil {
		id := *request.Params.TransferId
		uid, err := uuid.Parse(id)
		if err != nil {
			return TransferControllerGetTransfer400TextResponse(fmt.Sprintf("Can't parse UUID: %s", err.Error())), nil
		}

		status, err := i.taskQueue.GetTaskStatus(uid)
		if err != nil {
			return TransferControllerGetTransfer400TextResponse(fmt.Sprintf("No such task with id '%s'", uid.String())), nil
		}
		transferItems := []IngestorUiGetTransferItem{
			{
				Status:     &status.StatusMessage,
				TransferId: &id,
			},
		}

		totalItems := len(transferItems)
		return TransferControllerGetTransfer200JSONResponse{
			Total:     &totalItems,
			Transfers: &transferItems,
		}, nil
	}

	if request.Params.Page != nil {
		var start, end, pageIndex, pageSize uint

		pageSize = 50
		if request.Params.PageSize != nil {
			pageSize = uint(*request.Params.PageSize)
		}

		if *request.Params.Page <= 0 {
			pageIndex = 1
		} else {
			pageIndex = uint(*request.Params.Page)
		}

		start = (pageIndex - 1) * pageSize
		end = pageIndex * pageSize

		resultNo := i.taskQueue.GetTaskCount()
		ids, statuses, err := i.taskQueue.GetTaskStatusList(start, end)
		if err != nil {
			return TransferControllerGetTransfer400TextResponse(err.Error()), nil
		}

		transferItems := []IngestorUiGetTransferItem{}
		for i, status := range statuses {
			idString := ids[i].String()
			s := status.StatusMessage
			if !status.Failed {
				if status.Finished {
					s = "finished"
				} else if status.Started {
					s = fmt.Sprintf(
						"progress: %d%%",
						int(math.Round(float64(status.BytesTransferred)/float64(status.BytesTotal))),
					)
				} else {
					s = "queued"
				}
			} else if status.StatusMessage == "" {
				s = "failed - unknown error"
			}
			transferItems = append(transferItems, IngestorUiGetTransferItem{
				Status:     &s,
				TransferId: &idString,
			})
		}

		return TransferControllerGetTransfer200JSONResponse{
			Total:     &resultNo,
			Transfers: &transferItems,
		}, nil
	}

	return TransferControllerGetTransfer400TextResponse("Not enough parameters"), nil
}

func (i *IngestorWebServerImplemenation) GetLogin(ctx context.Context, request GetLoginRequestObject) (GetLoginResponseObject, error) {
	state, err := generateState()
	if err != nil {
		return GetLogin302Response{}, err
	}

	return GetLogin302Response{
		Headers: GetLogin302ResponseHeaders{
			Location:  i.oauth2Config.RedirectURL,
			SetCookie: fmt.Sprintf("saved-state=%s; HttpOnly; Max-Age=600", url.QueryEscape(i.encrypt(state))),
		},
	}, nil
}

func (i *IngestorWebServerImplemenation) GetCallback(ctx context.Context, request GetCallbackRequestObject) (GetCallbackResponseObject, error) {
	// CSRF attack protection
	if request.Params.State != i.decrypt(request.Params.SavedState) {
		return GetCallback400TextResponse("invalid state"), nil
	}

	// exchange authorization code for accessToken
	token, err := i.oauth2Config.Exchange(context.Background(), request.Params.Code)
	if err != nil {
		return GetCallback400TextResponse(fmt.Sprintf("code exchange failed: %s", err.Error())), nil
	}

	// get id token
	rawIdToken, ok := token.Extra("id_token").(string)
	if !ok {
		return GetCallback400TextResponse("id_token was not found in token"), nil
	}
	idToken, err := oidcVerifier.Verify(context.Background(), rawIdToken)
	if err != nil {
		return GetCallback400TextResponse(fmt.Sprintf("idToken verification failed: %s", err.Error())), nil
	}

	// extract claims
	claims := struct {
		Email string `json:"email"`
		Name  string `json:"name"`
	}{}
	if err := idToken.Claims(&claims); err != nil {
		return GetCallback400TextResponse("could not parse token claims"), nil
	}

	// return response
	dummyIdTokenToReplace := ""
	return GetCallback200JSONResponse{
		AccessToken: &token.AccessToken,
		IdToken:     &dummyIdTokenToReplace,
		Email:       &claims.Email,
		Name:        &claims.Name,
	}, nil
}

func (i *IngestorWebServerImplemenation) encrypt(plaintext string) string {
	ciphertext := make([]byte, len(plaintext))
	i.cookieCipher.Encrypt(ciphertext, []byte(plaintext))
	return string(ciphertext)
}

func (i *IngestorWebServerImplemenation) decrypt(ciphertext string) string {
	plaintext := make([]byte, len(ciphertext))
	i.cookieCipher.Decrypt(plaintext, []byte(ciphertext))
	return string(plaintext)
}
