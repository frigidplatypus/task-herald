package web

import (
    "encoding/json"
    "errors"
    "net/http"
)

// HTTP payloads
type CreateTaskRequest struct {
    Description      string   `json:"description"`
    Project          string   `json:"project,omitempty"`
    Tags             []string `json:"tags,omitempty"`
    NotificationDate string   `json:"notification_date,omitempty"`
}

type CreateTaskResponse struct {
    UUID    string `json:"uuid"`
    Message string `json:"message"`
}

type AcknowledgeRequest struct {
    UUID        string `json:"uuid"`
    RepeatDelay string `json:"repeat_delay,omitempty"`
}

type AcknowledgeResponse struct {
    Acknowledged bool `json:"acknowledged"`
}

// Package-level hooks so tests can override behavior
var (
    CreateTaskFunc = func(req CreateTaskRequest) (string, error) {
        return "", errors.New("not implemented")
    }
    AcknowledgeFunc = func(uuid string, repeatDelay string) error {
        return errors.New("not implemented")
    }
)

// NewRouter returns an http.Handler with the API routes mounted
func NewRouter() http.Handler {
    mux := http.NewServeMux()
    mux.HandleFunc("/api/health", healthHandler)
    mux.HandleFunc("/api/create-task", createTaskHandler)
    mux.HandleFunc("/api/acknowledge", acknowledgeHandler)
    mux.HandleFunc("/api/debug", debugHandler)
    return mux
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodGet {
        http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
        return
    }
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusOK)
    _ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func createTaskHandler(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost {
        http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
        return
    }
    var req CreateTaskRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, "invalid request", http.StatusBadRequest)
        return
    }
    if req.Description == "" {
        http.Error(w, "description required", http.StatusBadRequest)
        return
    }
    uuid, err := CreateTaskFunc(req)
    if err != nil {
        http.Error(w, "failed to create task", http.StatusInternalServerError)
        return
    }
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusCreated)
    _ = json.NewEncoder(w).Encode(CreateTaskResponse{UUID: uuid, Message: "created"})
}

func acknowledgeHandler(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost {
        http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
        return
    }
    var req AcknowledgeRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, "invalid request", http.StatusBadRequest)
        return
    }
    if req.UUID == "" {
        http.Error(w, "uuid required", http.StatusBadRequest)
        return
    }
    if err := AcknowledgeFunc(req.UUID, req.RepeatDelay); err != nil {
        http.Error(w, "failed to acknowledge", http.StatusInternalServerError)
        return
    }
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusOK)
    _ = json.NewEncoder(w).Encode(AcknowledgeResponse{Acknowledged: true})
}

func debugHandler(w http.ResponseWriter, r *http.Request) {
    // simple debug endpoint
    if r.Method != http.MethodGet {
        http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
        return
    }
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusOK)
    _ = json.NewEncoder(w).Encode(map[string]string{"debug": "ok"})
}

// AuthMiddleware enforces an Authorization: Bearer <token> header when token is non-empty
func AuthMiddleware(handler http.Handler, token string) http.Handler {
    if token == "" {
        return handler
    }
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        auth := r.Header.Get("Authorization")
        if auth != "Bearer "+token {
            http.Error(w, "unauthorized", http.StatusUnauthorized)
            return
        }
        handler.ServeHTTP(w, r)
    })
}
