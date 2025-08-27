package app


import (
       "fmt"
       "os"
       "os/signal"
       "sync"
       "syscall"
       "time"

       "github.com/yourusername/task-herald/internal/config"
       "github.com/yourusername/task-herald/internal/taskwarrior"
       "github.com/yourusername/task-herald/internal/web"
)

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

       // Set up polling
       taskCh := make(chan []taskwarrior.Task)
       stopCh := make(chan struct{})
       go taskwarrior.Poller(cfg.PollInterval, taskCh, stopCh)

       // Update tasks on poll
       go func() {
              for t := range taskCh {
                     mu.Lock()
                     tasks = t
                     mu.Unlock()
                     fmt.Printf("Polled %d tasks with notification_date set\n", len(t))
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
