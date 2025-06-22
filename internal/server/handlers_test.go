package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/TheRealSibasishBehera/init-go/internal/exec"
)

func TestStatusHandler(t *testing.T) {
	req, err := http.NewRequest("GET", "/status", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(statusHandler)

	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("StatusHandler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	expected := `{"status": "OK"}`
	if rr.Body.String() != expected {
		t.Errorf("StatusHandler returned unexpected body: got %v want %v",
			rr.Body.String(), expected)
	}

	contentType := rr.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("StatusHandler returned wrong content type: got %v want %v",
			contentType, "application/json")
	}
}

func TestExecHandler_Success(t *testing.T) {
	requestBody := exec.ExecRequest{
		Cmd: []string{"echo", "hello world"},
	}
	body, _ := json.Marshal(requestBody)

	req, err := http.NewRequest("POST", "/v1/exec", bytes.NewBuffer(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	
	handler := &APIHandler{
		waitPidMutex: &sync.Mutex{},
		envs:         map[string]string{"PATH": "/bin:/usr/bin"},
	}

	handler.ExecHandler(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("ExecHandler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	var response exec.ExecResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response.ExitCode == nil || *response.ExitCode != 0 {
		t.Errorf("Expected exit code 0, got %v", response.ExitCode)
	}

	expectedOutput := "hello world\n"
	if string(response.Stdout) != expectedOutput {
		t.Errorf("Expected stdout '%s', got '%s'", expectedOutput, string(response.Stdout))
	}
}

func TestExecHandler_EmptyCommand(t *testing.T) {
	requestBody := exec.ExecRequest{
		Cmd: []string{},
	}
	body, _ := json.Marshal(requestBody)

	req, err := http.NewRequest("POST", "/v1/exec", bytes.NewBuffer(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	
	handler := &APIHandler{
		waitPidMutex: &sync.Mutex{},
		envs:         map[string]string{},
	}

	handler.ExecHandler(rr, req)

	if status := rr.Code; status != http.StatusBadRequest {
		t.Errorf("ExecHandler returned wrong status code: got %v want %v",
			status, http.StatusBadRequest)
	}
}

func TestExecHandler_InvalidJSON(t *testing.T) {
	invalidJSON := []byte(`{"cmd": [}`)

	req, err := http.NewRequest("POST", "/v1/exec", bytes.NewBuffer(invalidJSON))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	
	handler := &APIHandler{
		waitPidMutex: &sync.Mutex{},
		envs:         map[string]string{},
	}

	handler.ExecHandler(rr, req)

	if status := rr.Code; status != http.StatusBadRequest {
		t.Errorf("ExecHandler returned wrong status code: got %v want %v",
			status, http.StatusBadRequest)
	}
}

func TestExecHandler_CommandFailure(t *testing.T) {
	requestBody := exec.ExecRequest{
		Cmd: []string{"ls", "/nonexistent-directory"},
	}
	body, _ := json.Marshal(requestBody)

	req, err := http.NewRequest("POST", "/v1/exec", bytes.NewBuffer(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	
	handler := &APIHandler{
		waitPidMutex: &sync.Mutex{},
		envs:         map[string]string{"PATH": "/bin:/usr/bin"},
	}

	handler.ExecHandler(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("ExecHandler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	var response exec.ExecResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response.ExitCode == nil || *response.ExitCode == 0 {
		t.Errorf("Expected non-zero exit code, got %v", response.ExitCode)
	}

	if len(response.Stderr) == 0 {
		t.Error("Expected stderr output for failed command")
	}
}