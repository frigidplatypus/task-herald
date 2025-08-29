package notify

import (
	"context"
	"log"

	shoutrrr "github.com/containrrr/shoutrrr"
)

// Notifier wraps shoutrrr for sending notifications
// url: shoutrrr service URL (e.g. ntfy, discord, etc)
// logger: optional logger for errors

type Notifier struct {
	url    string
	logger *log.Logger
}

func NewNotifier(url string, logger *log.Logger) *Notifier {
	return &Notifier{url: url, logger: logger}
}

func (n *Notifier) Send(ctx context.Context, message string) error {
	sender, err := shoutrrr.CreateSender(n.url)
	if err != nil {
		if n.logger != nil {
			n.logger.Printf("[notify] failed to create shoutrrr sender: %v", err)
		}
		return err
	}

       errs := sender.Send(message, nil)
       if len(errs) > 0 && errs[0] != nil {
	       if n.logger != nil {
		       n.logger.Printf("[notify] failed to send notification: %v", errs)
	       }
	       return errs[0]
       }
       return nil
}
