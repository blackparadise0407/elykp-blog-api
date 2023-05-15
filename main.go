package main

import (
	"fmt"
	"log"
	"net/mail"

	"github.com/gosimple/slug"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/models"
	"github.com/pocketbase/pocketbase/tools/mailer"

	"elykp.com/server/pkg/session"
	_ "elykp.com/server/pkg/session/providers/memory"
)

func setViewCount(app *pocketbase.PocketBase, r *models.Record) error {
	currViewCounts, ok := r.Get("viewCounts").(int)
	if ok {
		r.Set("viewCounts", currViewCounts+1)
	} else {
		r.Set("viewCounts", 1)
	}
	if err := app.Dao().SaveRecord(r); err != nil {
		return err
	}
	return nil
}

func main() {
	app := pocketbase.New()
	var globalSessions *session.Manager

	if sessions, err := session.NewManager("memory", "session", 3600); err != nil {
		log.Fatal(err.Error())
	} else {
		globalSessions = sessions
	}
	go globalSessions.GC()

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
		var ip string
		if ip = e.HttpContext.Request().Header.Get("X-Forwarded-For"); ip == "" {
			ip = e.HttpContext.Request().RemoteAddr
		}
		currRecordId := e.Record.Id
		sess := globalSessions.SessionStart(e.HttpContext.Response(), e.HttpContext.Request())
		ids, ok := sess.Get(ip).([]string)
		if ok {
			for _, id := range ids {
				if id == currRecordId {
					break
				} else {
					sess.Set(ip, append(ids, currRecordId))
					setViewCount(app, e.Record)
				}
			}
		} else {
			sess.Set(ip, []string{currRecordId})
			setViewCount(app, e.Record)
		}

		return nil
	})

	if err := app.Start(); err != nil {
		log.Fatal(err)
	}

}
