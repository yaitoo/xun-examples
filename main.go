package main

import (
	"embed"
	"flag"
	"io/fs"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/yaitoo/htmx"
)

//go:embed app
var fsys embed.FS

func main() {
	var dev bool
	flag.BoolVar(&dev, "dev", false, "it is development environment")

	flag.Parse()

	var opts []htmx.Option
	if dev {
		// use local filesystem in development, and watch files to reload automatically
		opts = []htmx.Option{htmx.WithFsys(os.DirFS("./app")), htmx.WithWatch()}
	} else {
		// use embed resources in production environment
		views, _ := fs.Sub(fsys, "app")
		opts = []htmx.Option{htmx.WithFsys(views)}
	}
	app := htmx.New(opts...)

	app.Use(func(next htmx.HandleFunc) htmx.HandleFunc {
		return func(c *htmx.Context) error {
			n := time.Now()
			defer func() {
				duration := time.Since(n)

				log.Println(c.Routing.Pattern, duration)
			}()
			return next(c)
		}
	})

	app.Get("/{$}", func(c *htmx.Context) error {
		return c.View(map[string]string{
			"Name": "go-htmx",
		})
	})

	app.Get("/user/{id}", func(c *htmx.Context) error {
		id := c.Request().PathValue("id")
		user := getUserById(id)
		return c.View(user)
	})

	admin := app.Group("/admin")

	admin.Use(func(next htmx.HandleFunc) htmx.HandleFunc {
		return func(c *htmx.Context) error {
			token := c.Request().Header.Get("X-Token")
			if !checkToken(token) {
				c.WriteStatus(http.StatusUnauthorized)
				return htmx.ErrCancelled
			}
			return next(c)
		}
	})

	app.Start()
	defer app.Close()

	if dev {
		log.Println("htmx-admin is running in development")
	} else {
		log.Println("htmx-admin is running in production")
	}

	err := http.ListenAndServe(":80", http.DefaultServeMux)
	if err != nil {
		panic(err)
	}
}

func getUserById(id string) User {
	return User{
		ID:   id,
		Name: "Yaitoo",
	}
}

func checkToken(token string) bool {
	return true
}

type User struct {
	ID   string
	Name string
}
