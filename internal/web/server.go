package web

import (
	"encoding/json"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"task-herald/internal/taskwarrior"
	"time"
)

var (
	ackFilePath = "ack/acknowledged.json"
	ackMu       sync.Mutex
)

// loadAcknowledged loads the set of acknowledged UUIDs from file
func loadAcknowledged() map[string]struct{} {
	ackMu.Lock()
	defer ackMu.Unlock()
	m := make(map[string]struct{})
	data, err := ioutil.ReadFile(ackFilePath)
	if err != nil {
		return m
	}
	var uuids []string
	if err := json.Unmarshal(data, &uuids); err != nil {
		return m
	}
	for _, u := range uuids {
		m[u] = struct{}{}
	}
	return m
}

// saveAcknowledged saves the set of acknowledged UUIDs to file
func saveAcknowledged(m map[string]struct{}) {
	ackMu.Lock()
	defer ackMu.Unlock()
	uuids := make([]string, 0, len(m))
	for u := range m {
		uuids = append(uuids, u)
	}
	data, _ := json.MarshalIndent(uuids, "", "  ")
	_ = os.MkdirAll("ack", 0755)
	_ = ioutil.WriteFile(ackFilePath, data, 0644)
}

// IsAcknowledged returns true if the task UUID is acknowledged
func IsAcknowledged(uuid string) bool {
	m := loadAcknowledged()
	_, ok := m[uuid]
	return ok
}

// MarkAcknowledged adds the UUID to the acknowledged set
func MarkAcknowledged(uuid string) {
	m := loadAcknowledged()
	m[uuid] = struct{}{}
	saveAcknowledged(m)
}

type Server struct {
	tmpl     *template.Template
	GetTasks func() []taskwarrior.Task
}

func NewServer(getTasks func() []taskwarrior.Task) *Server {
       // Allow override of template/static dir via env var (for Nix, etc)
       assetDir := os.Getenv("TASK_HERALD_ASSET_DIR")
       if assetDir == "" {
	       assetDir = filepath.Join("web")
       }
       tmpl := template.Must(template.ParseFiles(
	       filepath.Join(assetDir, "templates", "layout.html"),
	       filepath.Join(assetDir, "templates", "index.html"),
	       filepath.Join(assetDir, "templates", "tasks.html"),
       ))
       return &Server{tmpl: tmpl, GetTasks: getTasks}
}

func (s *Server) Serve(addr string) error {
       assetDir := os.Getenv("TASK_HERALD_ASSET_DIR")
       if assetDir == "" {
	       assetDir = filepath.Join("web")
       }
       staticDir := filepath.Join(assetDir, "static")
       http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir(staticDir))))
	http.HandleFunc("/", s.handleIndex)
	http.HandleFunc("/api/tasks", s.handleTasks)
	http.HandleFunc("/api/set-notification-date", s.handleSetNotificationDate)
	http.HandleFunc("/acknowledge", s.handleAcknowledge)
	http.HandleFunc("/delay", s.handleDelay)
	log.Printf("Web UI listening on %s", addr)
	return http.ListenAndServe(addr, nil)
}

// handleAcknowledge marks a task as acknowledged (writes to file)
func (s *Server) handleAcknowledge(w http.ResponseWriter, r *http.Request) {
	uuid := r.URL.Query().Get("uuid")
	if uuid == "" {
		http.Error(w, "Missing uuid", http.StatusBadRequest)
		return
	}
	MarkAcknowledged(uuid)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// handleDelay delays a task's notification_date by N minutes
func (s *Server) handleDelay(w http.ResponseWriter, r *http.Request) {
	uuid := r.URL.Query().Get("uuid")
	minutes := r.URL.Query().Get("minutes")
	if uuid == "" || minutes == "" {
		http.Error(w, "Missing uuid or minutes", http.StatusBadRequest)
		return
	}
	// Get current notification_date
	tasks := s.GetTasks()
	var task *taskwarrior.Task
	for i := range tasks {
		if tasks[i].UUID == uuid {
			task = &tasks[i]
			break
		}
	}
	if task == nil || task.NotificationDate == "" {
		http.Error(w, "Task not found or no notification_date", http.StatusNotFound)
		return
	}
	// Parse and add minutes
	t, err := task.ParseNotificationDate()
	if err != nil {
		http.Error(w, "Invalid notification_date", http.StatusBadRequest)
		return
	}
	var min int
	_, err = fmt.Sscanf(minutes, "%d", &min)
	if err != nil || min <= 0 {
		http.Error(w, "Invalid minutes", http.StatusBadRequest)
		return
	}
	newTime := t.Add(time.Duration(min) * time.Minute)
	formatted := newTime.Format("2006-01-02 15:04:05")
	cmd := exec.Command("task", uuid, "modify", "notification_date:"+formatted)
	if err := cmd.Run(); err != nil {
		http.Error(w, "Failed to delay task", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// ...existing code...
// handleSetNotificationDate sets the notification_date UDA for a task by UUID using task modify.
func (s *Server) handleSetNotificationDate(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form", http.StatusBadRequest)
		return
	}
	uuid := r.FormValue("uuid")
	dateTimeLocal := r.FormValue("notification_date")
	if uuid == "" || dateTimeLocal == "" {
		http.Error(w, "Missing uuid or notification_date", http.StatusBadRequest)
		return
	}
	// Convert from HTML5 datetime-local (YYYY-MM-DDTHH:MM) to Taskwarrior format (YYYY-MM-DD HH:MM:SS)
	formatted := dateTimeLocal
	if len(dateTimeLocal) >= 16 {
		// Insert a space instead of 'T' and add :00 seconds if not present
		formatted = dateTimeLocal[:10] + " " + dateTimeLocal[11:16] + ":00"
	}
	cmd := exec.Command("task", uuid, "modify", "notification_date:"+formatted)
	if err := cmd.Run(); err != nil {
		http.Error(w, "Failed to set notification_date", http.StatusInternalServerError)
		return
	}
	// Return the updated row (HTMX swaps it in)
	// Find the updated task and render its row only
	tasks := s.GetTasks()
	var updated *taskwarrior.Task
	for i := range tasks {
		if tasks[i].UUID == uuid {
			updated = &tasks[i]
			break
		}
	}
	if updated == nil {
		http.Error(w, "Task not found", http.StatusNotFound)
		return
	}
	// Render a single row using the same template logic
	// We'll use a mini-template for just one row
	rowTmpl := `
<tr>
    <td>{{ .ID }}</td>
    <td>{{ .Description }}</td>
    <td>{{ .Project }}</td>
    <td>{{ .NotificationDate }}</td>
    <td>{{ range .Tags }}<span class="tag">{{ . }}</span> {{ end }}</td>
    <td>{{ .Priority }}</td>
    <td>{{ .Status }}</td>
    <td>
	<form hx-post="/api/set-notification-date" hx-params="*" hx-target="closest tr" hx-swap="outerHTML">
	    <input type="hidden" name="uuid" value="{{ .UUID }}">
	    <input type="date" name="notification_date" value="{{ .NotificationDate }}">
	    <button type="submit">Set</button>
	</form>
    </td>
</tr>
`
	tmpl, err := template.New("row").Parse(rowTmpl)
	if err != nil {
		http.Error(w, "Template error", http.StatusInternalServerError)
		return
	}
	tmpl.Execute(w, updated)
}

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	s.tmpl.ExecuteTemplate(w, "layout.html", nil)
}

func (s *Server) handleTasks(w http.ResponseWriter, r *http.Request) {
	tasks := s.GetTasks()
	log.Printf("[DEBUG] handleTasks: %d tasks returned to UI", len(tasks))
	s.tmpl.ExecuteTemplate(w, "tasks.html", tasks)
}
