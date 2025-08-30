package web

import (
	"html/template"
	"log"
	"net/http"
	"os/exec"
	"path/filepath"
	"os"
	"encoding/json"

	"task-herald/internal/taskwarrior"
)

type Server struct {
	tmpl     *template.Template
	GetTasks func() []taskwarrior.Task
}

func NewServer(getTasks func() []taskwarrior.Task) *Server {
       // Find the directory of the running binary
       exePath, err := os.Executable()
       baseDir := "."
       if err == nil {
	       baseDir = filepath.Dir(exePath)
       }
       // In Nix, assets are at ../web/templates relative to the binary
       templatesDir := filepath.Join(baseDir, "..", "web", "templates")
       tmpl := template.Must(template.ParseFiles(
	       filepath.Join(templatesDir, "layout.html"),
	       filepath.Join(templatesDir, "index.html"),
	       filepath.Join(templatesDir, "tasks.html"),
       ))
       return &Server{tmpl: tmpl, GetTasks: getTasks}
}

func (s *Server) Serve(addr string) error {
       exePath, err := os.Executable()
       baseDir := "."
       if err == nil {
	       baseDir = filepath.Dir(exePath)
       }
       // In Nix, assets are at ../web/static relative to the binary
       staticDir := filepath.Join(baseDir, "..", "web", "static")
       http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir(staticDir))))
       http.HandleFunc("/", s.handleIndex)
       http.HandleFunc("/api/tasks", s.handleTasks)
       http.HandleFunc("/api/set-notification-date", s.handleSetNotificationDate)
       http.HandleFunc("/api/create-task", s.handleCreateTask)
       log.Printf("Web UI listening on %s", addr)
       return http.ListenAndServe(addr, nil)
}
// handleCreateTask handles POST /api/create-task to create a new Taskwarrior task from JSON
func (s *Server) handleCreateTask(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	type reqBody struct {
		Description      string   `json:"Description"`
		Project          string   `json:"Project"`
		Tags             []string `json:"Tags"`
		Due              string   `json:"Due"`
		NotificationDate string   `json:"NotificationDate"`
	}
	var body reqBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	if body.Description == "" {
		http.Error(w, "Description is required", http.StatusBadRequest)
		return
	}
	args := []string{"add", body.Description}
	if body.Project != "" {
		args = append(args, "project:"+body.Project)
	}
	if len(body.Tags) > 0 {
		for _, tag := range body.Tags {
			if tag != "" {
				args = append(args, "+"+tag)
			}
		}
	}
	if body.Due != "" {
		args = append(args, "due:"+body.Due)
	}
	if body.NotificationDate != "" {
		args = append(args, "notification_date:"+body.NotificationDate)
	}
	cmd := exec.Command("task", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		http.Error(w, "Failed to create task: "+string(output), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	w.Write([]byte(`{"status":"ok"}`))
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
