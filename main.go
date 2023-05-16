package main

import (
	"fmt"
	"log"
	"net/http"
	"net/mail"
	"time"

	"elykp.com/server/cache"
	"github.com/gosimple/slug"
	"github.com/labstack/echo/v5"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/mailer"
)

func getIp(c echo.Context) string {
	if ip := c.Request().Header.Get("X-Forwarded-For"); ip != "" {
		return ip
	}
	return c.RealIP()
}

func main() {
	app := pocketbase.New()

	cacheMgr := cache.NewLocalCache(time.Second)

	app.OnBeforeServe().Add(func(e *core.ServeEvent) error {
		e.Router.POST("/api/posts/:id/view-counts", func(c echo.Context) error {
			currPostId := c.PathParam("id")
			record, err := e.App.Dao().FindRecordById("posts", currPostId)
			if err != nil {
				return c.String(http.StatusNotFound, "Record not found")
			}
			ip := getIp(c)
			cacheKey := fmt.Sprintf("view-counts:%v-%v", ip, currPostId)
			if _, err := cacheMgr.Get(cacheKey); err != nil {
				cacheMgr.Set(cacheKey, 1, time.Now().Add(time.Hour*2).Unix())
				record.Set("viewCounts", record.GetInt("viewCounts")+1)
				if err := e.App.Dao().SaveRecord(record); err != nil {
					return c.String(http.StatusBadRequest, err.Error())
				}
			}

			return c.NoContent(http.StatusNoContent)
		})

		return nil
	})

	app.OnRecordAfterCreateRequest("posts").Add(func(e *core.RecordCreateEvent) error {
		subscriptions, err := app.Dao().FindRecordsByExpr("subscriptions", dbx.HashExp{"unsubscribed": false})

		if title, ok := e.Record.Get("title").(string); ok {
			postSlug := fmt.Sprintf("%s-%s", slug.Make(title), e.Record.Id)
			e.Record.Set("slug", postSlug)
			if err := app.Dao().SaveRecord(e.Record); err != nil {
				return err
			}

			go func() {
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
			}()

		} else {
			return nil
		}

		return err
	})

	if err := app.Start(); err != nil {
		cacheMgr.StopCleanup()
		log.Fatal(err)
	}

}
