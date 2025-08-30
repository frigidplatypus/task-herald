package web

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
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
       var req map[string]interface{}
       if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
	       http.Error(w, "Invalid JSON", http.StatusBadRequest)
	       return
       }
       desc, ok := req["Description"].(string)
       if !ok || desc == "" {
	       http.Error(w, "Description is required", http.StatusBadRequest)
	       return
       }
       args := []string{"add", desc}
       // Handle tags (array or string)
       if tags, ok := req["Tags"]; ok {
	       switch t := tags.(type) {
	       case []interface{}:
		       for _, tag := range t {
			       if tagStr, ok := tag.(string); ok && tagStr != "" {
				       args = append(args, "+"+tagStr)
			       }
		       }
	       case string:
		       if t != "" {
			       args = append(args, "+"+t)
		       }
	       }
       }
       // Handle annotations (array or string)
       var annotations []string
       if ann, ok := req["Annotations"]; ok {
	       switch a := ann.(type) {
	       case []interface{}:
		       for _, v := range a {
			       if s, ok := v.(string); ok && s != "" {
				       annotations = append(annotations, s)
			       }
		       }
	       case string:
		       if a != "" {
			       annotations = append(annotations, a)
		       }
	       }
       }
       // Add all other fields as key:value (skip Description, Tags, Annotations)
       for k, v := range req {
	       if k == "Description" || k == "Tags" || k == "Annotations" {
		       continue
	       }
	       // Convert value to string
	       var val string
	       switch vv := v.(type) {
	       case string:
		       val = vv
	       case float64:
		       val = fmt.Sprintf("%v", vv)
	       case bool:
		       val = fmt.Sprintf("%v", vv)
	       default:
		       continue
	       }
	       if val != "" {
		       args = append(args, fmt.Sprintf("%s:%s", toSnakeCase(k), val))
	       }
       }
       cmd := exec.Command("task", args...)
       output, err := cmd.CombinedOutput()
       if err != nil {
	       log.Printf("[api] Failed to create task: %s (args: %v)", string(output), args)
	       http.Error(w, "Failed to create task: "+string(output), http.StatusInternalServerError)
	       return
       }
       log.Printf("[api] Created new task via API: args=%v output=%s", args, string(output))

       // If there are annotations, get the new task's ID and add them
       if len(annotations) > 0 {
	       // Try to extract the new task's ID from output (e.g., "Created task 42.")
	       id := extractTaskID(string(output))
	       if id != "" {
		       for _, ann := range annotations {
			       annCmd := exec.Command("task", id, "annotate", ann)
			       annOut, annErr := annCmd.CombinedOutput()
			       if annErr != nil {
				       log.Printf("[api] Failed to annotate task %s: %s", id, string(annOut))
			       } else {
				       log.Printf("[api] Annotated task %s: %s", id, ann)
			       }
		       }
	       } else {
		       log.Printf("[api] Could not extract task ID to annotate")
	       }
       }

       w.Header().Set("Content-Type", "application/json")
       w.WriteHeader(http.StatusCreated)
       w.Write([]byte(`{"status":"ok"}`))
}


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

// toSnakeCase converts CamelCase or PascalCase to snake_case
func toSnakeCase(s string) string {
       var out []rune
       for i, r := range s {
	       if i > 0 && r >= 'A' && r <= 'Z' {
		       out = append(out, '_')
	       }
	       out = append(out, r)
       }
       return strings.ToLower(string(out))
}

// extractTaskID tries to extract the task ID from taskwarrior output
func extractTaskID(output string) string {
       re := regexp.MustCompile(`Created task (\d+)`)
       match := re.FindStringSubmatch(output)
       if len(match) > 1 {
	       return match[1]
       }
       return ""
}

