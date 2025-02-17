package main

import (
	"embed"
	"flag"
	"io/fs"
	"log"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/yaitoo/xun"
	"github.com/yaitoo/xun/ext/form"
	"github.com/yaitoo/xun/ext/htmx"
)

//go:embed app/components
//go:embed app/public
//go:embed app/layouts
//go:embed app/pages
//go:embed app/text
var fsys embed.FS

func main() {
	var dev bool
	flag.BoolVar(&dev, "dev", false, "it is development environment")

	flag.Parse()

	var opts []xun.Option
	if dev {
		// use local filesystem in development, and watch files to reload automatically
		opts = []xun.Option{xun.WithFsys(os.DirFS("./app")), xun.WithWatch()}
	} else {
		// use embed resources in production environment
		views, _ := fs.Sub(fsys, "app")
		opts = []xun.Option{xun.WithFsys(views)}
	}

	opts = append(opts, xun.WithInterceptor(htmx.New()))
	app := xun.New(opts...)

	app.Use(loggingRequest, loadSession)

	app.Get("/{$}", func(c *xun.Context) error {
		return c.View(map[string]string{
			"Name": "go-xun",
		})
	})

	app.Get("/user/{id}", func(c *xun.Context) error {
		id := c.Request.PathValue("id")
		user := getUserById(id)
		return c.View(user)
	})

	app.Get("/sitemap.xml", func(c *xun.Context) error {
		return c.View(struct {
			LastMod time.Time
		}{
			LastMod: time.Now(),
		}, "text/sitemap.xml")
	})

	admin := app.Group("/admin")

	admin.Use(loggingRequest, loadSession,
		func(next xun.HandleFunc) xun.HandleFunc {
			return func(c *xun.Context) error {
				_, ok := c.Get("Session").(string)
				if !ok {
					c.Redirect("/login?return=" + c.Request.URL.Path)
					return xun.ErrCancelled
				}

				return next(c)
			}
		})

	admin.Get("/{$}", func(c *xun.Context) error {
		s, _ := c.Get("Session").(string)
		return c.View(User{
			Name: s,
		})
	})

	app.Post("/login", func(c *xun.Context) error {

		it, err := form.BindForm[Login](c.Request)

		if err != nil {
			c.WriteStatus(http.StatusBadRequest)
			return xun.ErrCancelled
		}

		if !it.Validate(c.AcceptLanguage()...) {
			c.WriteStatus(http.StatusBadRequest)
			return c.View(it)
		}

		if it.Data.Email != "xun@yaitoo.cn" || it.Data.Password != "123" {
			htmx.WriteHeader(c, htmx.HxTrigger, htmx.HxHeader[string]{
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

		http.SetCookie(c.Response, &cookie)

		ref, _ := url.Parse(c.RequestReferer())

		ret := ref.Query().Get("return")

		if ret == "" {
			ret = "/admin"
		}

		c.Redirect(ret)
		return nil
	})

	app.Start()
	defer app.Close()

	if dev {
		slog.Default().Info("xun-admin is running in development")
	} else {
		slog.Default().Info("xun-admin is running in production")
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

func loggingRequest(next xun.HandleFunc) xun.HandleFunc {
	return func(c *xun.Context) error {
		n := time.Now()
		defer func() {
			duration := time.Since(n)

			log.Println(c.Routing.Pattern, duration)
		}()
		return next(c)
	}
}

func loadSession(next xun.HandleFunc) xun.HandleFunc {
	return func(c *xun.Context) error {
		s, err := c.Request.Cookie("session")
		if err == nil {
			c.Set("Session", s.Value)
		}

		return next(c)
	}
}

type Login struct {
	Email    string `form:"email" validate:"required,email"`
	Password string `form:"password" validate:"required"`
}

type User struct {
	ID   string
	Name string
}
