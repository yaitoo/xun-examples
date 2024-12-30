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

//go:embed app/components
//go:embed app/public
//go:embed app/layouts
//go:embed app/pages
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
			s, err := c.Request().Cookie("session")
			if err != nil || s == nil || s.Value == "" {
				c.Redirect("/login?return=" + c.Request().URL.String())
				return htmx.ErrCancelled
			}

			c.Set("session", s.Value)
			return next(c)
		}
	})

	admin.Get("/{$}", func(c *htmx.Context) error {

		return c.View(User{
			Name: c.Get("session").(string),
		})
	})

	app.Post("/login", func(c *htmx.Context) error {

		it, err := htmx.BindForm[Login](c.Request())

		if err != nil {
			c.WriteStatus(http.StatusBadRequest)
			return htmx.ErrCancelled
		}

		if !it.Validate(c.AcceptLanguage()...) {
			c.WriteStatus(http.StatusBadRequest)
			return c.View(it)
		}

		if it.Data.Email != "htmx@yaitoo.cn" || it.Data.Password != "123" {
			c.WriteHtmxHeader(htmx.HxTrigger, htmx.HtmxHeader[string]{
				"showMessage": "Email or password is incorrect",
			})
			c.WriteStatus(http.StatusBadRequest)
			return c.View(it)
		}

		cookie := http.Cookie{
			Name:     "session",
			Value:    it.Data.Email,
			Path:     "/",
			MaxAge:   3600,
			HttpOnly: true,
			Secure:   true,
			SameSite: http.SameSiteLaxMode,
		}

		http.SetCookie(c.Writer(), &cookie)

		c.Redirect(c.GetCurrentUrl().Query().Get("return"))
		return nil
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

type Login struct {
	Email    string `form:"email" validate:"required,email"`
	Password string `form:"password" validate:"required"`
}

type User struct {
	ID   string
	Name string
}
