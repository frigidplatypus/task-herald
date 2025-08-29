package web

import (
	"html/template"
	"log"
	"net/http"
	"os/exec"
	"path/filepath"

	"task-herald/internal/taskwarrior"
)

type Server struct {
	tmpl     *template.Template
	GetTasks func() []taskwarrior.Task
}

func NewServer(getTasks func() []taskwarrior.Task) *Server {
	tmpl := template.Must(template.ParseFiles(
		filepath.Join("web", "templates", "layout.html"),
		filepath.Join("web", "templates", "index.html"),
		filepath.Join("web", "templates", "tasks.html"),
	))
	return &Server{tmpl: tmpl, GetTasks: getTasks}
}

func (s *Server) Serve(addr string) error {
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("web/static"))))
	http.HandleFunc("/", s.handleIndex)
	http.HandleFunc("/api/tasks", s.handleTasks)
	http.HandleFunc("/api/set-notification-date", s.handleSetNotificationDate)
	log.Printf("Web UI listening on %s", addr)
	return http.ListenAndServe(addr, nil)
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
