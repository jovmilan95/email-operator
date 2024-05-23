package thirdparty

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/mailersend/mailersend-go"
	"github.com/mailgun/mailgun-go/v4"
)

type MailClient struct {
	Provider  string
	ApiToken  string
	Recipient string
	Subject   string
	From      string
	Text      string
	Domain    string
}

type Result struct {
	MessageID      string
	DeliveryStatus string
}

const (
	MessageCheckTimeout  = 60 * time.Second
	MessageCheckInterval = 5 * time.Second
)

func (e *MailClient) SendEmail() (Result, error) {
	switch e.Provider {
	case "mailersend":
		return e.sendViaMailerSend()
	case "mailgun":
		return e.sendViaMailgun()
	default:
		return Result{}, errors.New("unsupported email provider")
	}
}

func (e *MailClient) sendViaMailerSend() (Result, error) {
	ctx := context.Background()
	ms := mailersend.NewMailersend(e.ApiToken)
	message := ms.Email.NewMessage()
	message.SetFrom(mailersend.From{
		Name:  e.From,
		Email: e.From,
	})
	message.SetRecipients([]mailersend.Recipient{
		{
			Name:  e.Recipient,
			Email: e.Recipient,
		},
	})
	message.SetSubject(e.Subject)
	message.SetText(e.Text)

	res, err := ms.Email.Send(ctx, message)
	if err != nil {
		return Result{}, err
	}
	messageID := res.Header.Get("X-Message-Id")

	// Initialize timer
	timer := time.NewTimer(MessageCheckTimeout)
	defer timer.Stop()

	for {
		select {
		case <-timer.C:
			// Timeout reached, return an error
			return Result{}, errors.New("timed out waiting for message delivery")
		default:
			// Check message delivery status
			messageResponse, _, err := ms.Message.Get(context.Background(), messageID)
			if err != nil {
				return Result{}, err
			}
			if len(messageResponse.Data.Emails) > 0 && messageResponse.Data.Emails[0].Status == "delivered" {
				// Message delivered, return success
				return Result{MessageID: messageID, DeliveryStatus: "Delivered"}, nil
			}
			// Wait for a short duration before checking again
			time.Sleep(MessageCheckInterval)
		}
	}
}

func (e *MailClient) sendViaMailgun() (Result, error) {
	mg := mailgun.NewMailgun(e.Domain, e.ApiToken)

	sender := e.From
	subject := e.Subject
	body := e.Text
	recipient := e.Recipient

	message := mg.NewMessage(sender, subject, body, recipient)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	// Send the message with a 10 second timeout
	_, messageID, err := mg.Send(ctx, message)

	messageID = strings.Trim(messageID, "<>")

	if err != nil {
		return Result{}, err
	}

	// Initialize timer
	timer := time.NewTimer(MessageCheckTimeout)
	defer timer.Stop()

	for {
		select {
		case <-timer.C:
			// Timeout reached, return an error
			return Result{}, errors.New("timed out waiting for message delivery")
		default:

			// Check message delivery status
			it := mg.ListEvents(&mailgun.ListEventOptions{
				Limit:  100,
				Filter: map[string]string{"message-id": messageID, "event": "delivered"},
			})

			var events []mailgun.Event
			noError := it.First(ctx, &events)
			if noError && len(events) > 0 {
				// Message delivered, return success
				return Result{MessageID: messageID, DeliveryStatus: "Delivered"}, nil
			}
			// Wait for a short duration before checking again
			time.Sleep(MessageCheckInterval)
		}
	}
}
