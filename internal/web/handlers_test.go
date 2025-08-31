package web

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHealthHandler(t *testing.T) {
	r := httptest.NewServer(NewRouter())
	defer r.Close()
	resp, err := http.Get(r.URL + "/api/health")
	if err != nil {
		t.Fatalf("http get failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", resp.StatusCode)
	}
	var body map[string]string
	_ = json.NewDecoder(resp.Body).Decode(&body)
	if body["status"] != "ok" {
		t.Fatalf("unexpected health body: %v", body)
	}
}

func TestCreateTaskHandler_Success(t *testing.T) {
	// override CreateTaskFunc
	orig := CreateTaskFunc
	defer func() { CreateTaskFunc = orig }()
	CreateTaskFunc = func(req CreateTaskRequest) (string, error) {
		return "uuid-123", nil
	}

	r := httptest.NewServer(NewRouter())
	defer r.Close()

	reqBody := CreateTaskRequest{Description: "do it", Project: "p", Annotations: []string{"note:foo"}}
	b, _ := json.Marshal(reqBody)
	resp, err := http.Post(r.URL+"/api/create-task", "application/json", bytes.NewReader(b))
	if err != nil {
		t.Fatalf("post failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 Created, got %d", resp.StatusCode)
	}
	var res CreateTaskResponse
	_ = json.NewDecoder(resp.Body).Decode(&res)
	if res.UUID != "uuid-123" {
		t.Fatalf("unexpected uuid: %v", res)
	}
}

func TestCreateTaskHandler_AnnotationsPassed(t *testing.T) {
	// verify CreateTaskFunc receives annotations
	orig := CreateTaskFunc
	defer func() { CreateTaskFunc = orig }()
	CreateTaskFunc = func(req CreateTaskRequest) (string, error) {
		if len(req.Annotations) != 2 || req.Annotations[0] != "a1" || req.Annotations[1] != "a2" {
			return "", io.ErrUnexpectedEOF
		}
		return "uuid-xyz", nil
	}

	r := httptest.NewServer(NewRouter())
	defer r.Close()

	reqBody := CreateTaskRequest{Description: "annotate", Annotations: []string{"a1", "a2"}}
	b, _ := json.Marshal(reqBody)
	resp, err := http.Post(r.URL+"/api/create-task", "application/json", bytes.NewReader(b))
	if err != nil {
		t.Fatalf("post failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 Created, got %d", resp.StatusCode)
	}
}

func TestCreateTaskHandler_BadRequest(t *testing.T) {
	r := httptest.NewServer(NewRouter())
	defer r.Close()
	resp, err := http.Post(r.URL+"/api/create-task", "application/json", bytes.NewReader([]byte(`{"project":"p"}`)))
	if err != nil {
		t.Fatalf("post failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 Bad Request, got %d", resp.StatusCode)
	}
}

func TestAcknowledgeHandler(t *testing.T) {
	orig := AcknowledgeFunc
	defer func() { AcknowledgeFunc = orig }()
	AcknowledgeFunc = func(uuid string, repeatDelay string) error {
		if uuid == "fail" {
			return io.ErrUnexpectedEOF
		}
		return nil
	}

	r := httptest.NewServer(NewRouter())
	defer r.Close()

	// success
	b, _ := json.Marshal(AcknowledgeRequest{UUID: "u1"})
	resp, err := http.Post(r.URL+"/api/acknowledge", "application/json", bytes.NewReader(b))
	if err != nil {
		t.Fatalf("post failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", resp.StatusCode)
	}
	var ar AcknowledgeResponse
	_ = json.NewDecoder(resp.Body).Decode(&ar)
	if !ar.Acknowledged {
		t.Fatalf("expected acknowledged true, got %v", ar)
	}

	// missing uuid
	b2, _ := json.Marshal(AcknowledgeRequest{})
	resp2, err := http.Post(r.URL+"/api/acknowledge", "application/json", bytes.NewReader(b2))
	if err != nil {
		t.Fatalf("post failed: %v", err)
	}
	defer resp2.Body.Close()
	if resp2.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 Bad Request for missing uuid, got %d", resp2.StatusCode)
	}

	// internal error
	b3, _ := json.Marshal(AcknowledgeRequest{UUID: "fail"})
	resp3, err := http.Post(r.URL+"/api/acknowledge", "application/json", bytes.NewReader(b3))
	if err != nil {
		t.Fatalf("post failed: %v", err)
	}
	defer resp3.Body.Close()
	if resp3.StatusCode != http.StatusInternalServerError {
		t.Fatalf("expected 500 Internal Server Error for internal failure, got %d", resp3.StatusCode)
	}
}

func TestAuthMiddleware(t *testing.T) {
	handler := NewRouter()
	// wrap with token
	h := AuthMiddleware(handler, "s3cr3t")
	srv := httptest.NewServer(h)
	defer srv.Close()
	// without auth header should be 401
	resp, err := http.Get(srv.URL + "/api/health")
	if err != nil {
		t.Fatalf("get failed: %v", err)
	}
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401 unauthorized, got %d", resp.StatusCode)
	}
	resp.Body.Close()

	// with auth header should be OK
	req, _ := http.NewRequest("GET", srv.URL+"/api/health", nil)
	req.Header.Set("Authorization", "Bearer s3cr3t")
	client := &http.Client{}
	resp2, err := client.Do(req)
	if err != nil {
		t.Fatalf("auth get failed: %v", err)
	}
	defer resp2.Body.Close()
	if resp2.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 OK with auth, got %d", resp2.StatusCode)
	}
}
