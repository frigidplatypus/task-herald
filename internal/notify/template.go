package notify

import (
	"bytes"
	"text/template"
	"time"
)

type TaskInfo struct {
	ID               string
	UUID             string
	Description      string
	Tags             []string
	Due              *time.Time
	NotificationDate *time.Time
	Project          string
	Priority         string
}

const DefaultMessage = `🔔 Task Reminder: {{.Description}}
🆔 ID: {{.ID}}
📁 Project: {{.Project}}
🏷️ Tags: {{range .Tags}}{{.}} {{end}}
⏰ Due: {{if .Due}}{{.Due.Format "2006-01-02 15:04"}}{{else}}N/A{{end}}
📅 Notification: {{if .NotificationDate}}{{.NotificationDate.Format "2006-01-02 15:04"}}{{else}}N/A{{end}}`

func RenderMessage(task TaskInfo, tmpl string) (string, error) {
	if tmpl == "" {
		tmpl = DefaultMessage
	}
	t, err := template.New("msg").Parse(tmpl)
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	err = t.Execute(&buf, task)
	if err != nil {
		return "", err
	}
	return buf.String(), nil
}
