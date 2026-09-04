package extglobusservice

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestRequestExternalTransferTask_NonJSONErrorBody reproduces a panic seen in
// production: the external service returned a 403 without a JSON content
// type, leaving parsedResp.JSON403 nil. RequestExternalTransferTask must
// return an error instead of dereferencing that nil pointer.
func TestRequestExternalTransferTask_NonJSONErrorBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte("<html>403 Forbidden</html>"))
	}))
	defer srv.Close()

	_, err := RequestExternalTransferTask(
		context.Background(), srv.URL, "token", "src", "dst", "pid", true, "root",
		&[]FileToTransfer{},
	)
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
	reqErr, ok := err.(*RequestError)
	if !ok {
		t.Fatalf("expected *RequestError, got %T: %v", err, err)
	}
	if reqErr.Code() != 403 {
		t.Errorf("Code() = %d, want 403", reqErr.Code())
	}
}

// TestRequestExternalTransferTask_JSONErrorBody is the existing behavior:
// a well-formed JSON error body should still populate message/details.
func TestRequestExternalTransferTask_JSONErrorBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"message":"forbidden","details":"no access"}`))
	}))
	defer srv.Close()

	_, err := RequestExternalTransferTask(
		context.Background(), srv.URL, "token", "src", "dst", "pid", true, "root",
		&[]FileToTransfer{},
	)
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
	reqErr, ok := err.(*RequestError)
	if !ok {
		t.Fatalf("expected *RequestError, got %T: %v", err, err)
	}
	if reqErr.Code() != 403 {
		t.Errorf("Code() = %d, want 403", reqErr.Code())
	}
	if !strings.Contains(reqErr.Error(), "forbidden") {
		t.Errorf("Error() = %q, want it to contain %q", reqErr.Error(), "forbidden")
	}
	if !strings.Contains(reqErr.Details(), "no access") {
		t.Errorf("Details() = %q, want it to contain %q", reqErr.Details(), "no access")
	}
}

// TestRequestExternalTransferTask_NonJSONSuccessBody covers the same nil-body
// hazard on the 200 path.
func TestRequestExternalTransferTask_NonJSONSuccessBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("<html>ok</html>"))
	}))
	defer srv.Close()

	_, err := RequestExternalTransferTask(
		context.Background(), srv.URL, "token", "src", "dst", "pid", true, "root",
		&[]FileToTransfer{},
	)
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
}

// TestCancelTask_NonJSONErrorBody covers the identical nil-dereference hazard
// in CancelTask.
func TestCancelTask_NonJSONErrorBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte("<html>403 Forbidden</html>"))
	}))
	defer srv.Close()

	err := CancelTask(context.Background(), srv.URL, "token", "job-id", false)
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
	reqErr, ok := err.(*RequestError)
	if !ok {
		t.Fatalf("expected *RequestError, got %T: %v", err, err)
	}
	if reqErr.Code() != 403 {
		t.Errorf("Code() = %d, want 403", reqErr.Code())
	}
}
