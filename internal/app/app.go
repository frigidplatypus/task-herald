package app


import (
       "fmt"
       "os"
       "os/signal"
       "os/exec"
       "regexp"
       "strings"
       "sync"
       "syscall"
       "time"

       "github.com/yourusername/task-herald/internal/config"
       "github.com/yourusername/task-herald/internal/taskwarrior"
       "github.com/yourusername/task-herald/internal/web"
       "github.com/nicholas-fedor/shoutrrr"
)

// contains checks if a slice contains a string
func contains(slice []string, s string) bool {
       for _, v := range slice {
              if v == s {
                     return true
              }
       }
       return false
}

func Run() error {
       fmt.Println("Taskwarrior Notifications service starting...")

       cfg, err := config.LoadConfig("config.yaml")
       if err != nil {
              return fmt.Errorf("failed to load config: %w", err)
       }
       config.Set(cfg)

       // Shared state for polled tasks
       var (
              mu    sync.RWMutex
              tasks []taskwarrior.Task
       )

       // Set up Shoutrrr senders if configured
         // Initialize Shoutrrr senders
         var shoutrrrSenders []shoutrrr.Sender
         for _, url := range cfg.Shoutrrr.URLs {
                 sender, err := shoutrrr.CreateSender(url)
                 if err != nil {
                        fmt.Printf("invalid shoutrrr URL '%s': %v\n", url, err)
                        continue
                 }
                 shoutrrrSenders = append(shoutrrrSenders, sender)
         }
      // Set up polling
       taskCh := make(chan []taskwarrior.Task)
       stopCh := make(chan struct{})
       go taskwarrior.Poller(cfg.PollInterval, taskCh, stopCh)

       // Update tasks on poll
       go func() {
                       for t := range taskCh {
                                    // Process inline "+tag" directives: add tag and clean description
                                    for i, task := range t {
                                             re := regexp.MustCompile(`\+([A-Za-z0-9_-]+)`)  // match literal '+tag'
                                           matches := re.FindAllStringSubmatch(task.Description, -1)
                                           if len(matches) > 0 {
                                                  // Add each tag
                                                  for _, m := range matches {
                                                         tag := m[1]
                                                         exec.Command("task", task.UUID, "modify", "+"+tag).Run()
                                                         // locally update tags
                                                         if !contains(t[i].Tags, tag) {
                                                                t[i].Tags = append(t[i].Tags, tag)
                                                         }
                                                  }
                                                  // Clean description
                                                  newDesc := re.ReplaceAllString(task.Description, "")
                                                  newDesc = strings.TrimSpace(newDesc)
                                                  exec.Command("task", task.UUID, "modify", "description:"+newDesc).Run()
                                                  t[i].Description = newDesc
                                           }
                                    }
                                    mu.Lock()
                     tasks = t
                     mu.Unlock()
                   fmt.Printf("Polled %d tasks with notification_date set\n", len(t))
                               // Send notifications for tasks with notification_date
                               for _, task := range t {
                                      if task.NotificationDate == "" {
                                             continue
                                      }
                                      // Determine priority emoji
                                      var priEmoji string
                                      switch task.Priority {
                                      case "H": priEmoji = "🔥"
                                      case "M": priEmoji = "⚠️"
                                      case "L": priEmoji = "ℹ️"
                                      default:  priEmoji = "•"
                                      }
                                      // Append due date if set
                                      duePart := ""
                                      if task.Due != "" {
                                             duePart = fmt.Sprintf(" 🗓 Due: %s", task.Due)
                                      }
                                      // Build message: priority emoji, project, description, due date
                                      text := fmt.Sprintf("%s %s – %s%s", priEmoji, task.Project, task.Description, duePart)
                                      for _, sender := range shoutrrrSenders {
                                             go func(s shoutrrr.Sender, title, body string) {
                                                    msg := shoutrrr.Message{ Title: title, Text: body }
                                                    if err := s.Send(msg); err != nil {
                                                           fmt.Printf("failed to send notification: %v\n", err)
                                                    }
                                             }(sender, task.Description, text)
                                      }
                               }
              }
       }()

       // Start web server
       srv := web.NewServer(func() []taskwarrior.Task {
              mu.RLock()
              defer mu.RUnlock()
              return tasks
       })
       go srv.Serve(cfg.Web.Listen)

       // Handle graceful shutdown
       sigCh := make(chan os.Signal, 1)
       signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
       <-sigCh
       close(stopCh)
       time.Sleep(1 * time.Second) // Give poller time to exit
       return nil
}
