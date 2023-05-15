package main

import (
	"fmt"
	"log"
	"net/mail"

	"github.com/gosimple/slug"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/mailer"
)

func main() {
	app := pocketbase.New()

	app.OnRecordAfterCreateRequest("posts").Add(func(e *core.RecordCreateEvent) error {
		subscriptions, err := app.Dao().FindRecordsByExpr("subscriptions", dbx.HashExp{"unsubscribed": false})

		if title, ok := e.Record.Get("title").(string); ok {
			postSlug := fmt.Sprintf("%s-%s", slug.Make(title), e.Record.Id)
			e.Record.Set("slug", postSlug)
			if err := app.Dao().SaveRecord(e.Record); err != nil {
				return err
			}

			for _, subscription := range subscriptions {
				message := &mailer.Message{
					From: mail.Address{
						Address: app.Settings().Meta.SenderAddress,
						Name:    app.Settings().Meta.SenderName,
					},
					To:      []mail.Address{{Address: subscription.Email()}},
					Subject: "New post published",
					HTML: fmt.Sprintf(`
					Checkout our latest post on <a href="https://elykp.com/%s">Elykp.com</a>
					`, postSlug),
				}

				app.NewMailClient().Send(message)
			}
		} else {
			return nil
		}

		return err
	})

	app.OnRecordViewRequest("posts").Add(func(e *core.RecordViewEvent) error {
		e.Record.Set("viewCounts", e.Record.GetInt("viewCounts")+1)
		if err := app.Dao().SaveRecord(e.Record); err != nil {
			return err
		}
		return nil
	})

	if err := app.Start(); err != nil {
		log.Fatal(err)
	}

}
