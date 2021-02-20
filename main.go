package main

import (
	"context"
	"fmt"
	"github.com/ngerakines/tavern/model"
	"github.com/ngerakines/tavern/server"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/gin-contrib/gzip"
	ginzap "github.com/gin-contrib/zap"
	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
	_ "github.com/lib/pq"
	"github.com/oklog/run"
	"github.com/urfave/cli"
	"go.uber.org/zap"
)

func main() {
	app := cli.NewApp()
	app.Name = "tavern"
	app.Usage = "A minimal activity pub server."
	app.Copyright = "(c) 2019 Nick Gerakines"

	app.Commands = []cli.Command{
		{
			Name:   "server",
			Usage:  "Run the web server",
			Action: Server,
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:   "database",
					EnvVar: "DATABASE",
				},
				cli.StringFlag{
					Name:   "users",
					EnvVar: "USERs",
				},
				cli.StringFlag{
					Name:   "domain",
					EnvVar: "DOMAIN",
				},
				cli.StringFlag{
					Name:   "listen",
					EnvVar: "LISTEN",
					Value:  "0.0.0.0",
				},
				cli.StringFlag{
					Name:   "port",
					EnvVar: "PORT",
					Value:  "3000",
				},
				cli.BoolFlag{
					Name: "init, i",
				},
			},
		},
	}

	sort.Sort(cli.FlagsByName(app.Flags))
	sort.Sort(cli.CommandsByName(app.Commands))

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}

func Server(cliCtx *cli.Context) error {
	logger, err := zap.NewDevelopment()
	if err != nil {
		return err
	}

	logger.Info("Starting", zap.String("GOOS", runtime.GOOS))

	db, err := getDB(cliCtx)
	if err != nil {
		return err
	}
	defer func() {
		if cerr := db.Close(); cerr != nil {
			logger.Error("error closing database connection", zap.Error(cerr))
		}
	}()

	domain := cliCtx.String("domain")

	if err = automigrate(cliCtx, db, logger); err != nil {
		return err
	}

	r := gin.New()

	r.Use(ginzap.Ginzap(logger, time.RFC3339, true))
	r.Use(gzip.Gzip(gzip.DefaultCompression))

	r.GET("/health", func(i *gin.Context) {
		i.Data(200, "text/plain", []byte("OK"))
	})

	wkh := server.WebKnownHandler{
		Domain: domain,
		Logger: logger,
		DB:     db,
	}

	ah := server.ActorHandler{
		Domain: domain,
		Logger: logger,
		DB:     db,
	}

	r.GET("/.well-known/webfinger", wkh.WebFinger)

	usersRouter := r.Group("/users")
	{
		usersRouter.Use(server.MatchContentTypeMiddleware)
		usersRouter.Use(server.UserExistsMiddleware(logger, db, domain))
		usersRouter.GET("/:user", ah.ActorHandler)
		usersRouter.GET("/:user/followers", ah.FollowersHandler)
		usersRouter.GET("/:user/following", ah.FollowingHandler)

		usersRouter.GET("/:user/outbox", ah.OutboxHandler)
		usersRouter.POST("/:user/outbox", ah.OutboxSubmitHandler)
	}

	var g run.Group

	addr := net.JoinHostPort(cliCtx.String("listen"), cliCtx.String("port"))

	srv := &http.Server{
		Addr:    addr,
		Handler: r,
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	g.Add(func() error {
		logger.Info("starting http service", zap.String("addr", srv.Addr))
		return srv.ListenAndServe()
	}, func(error) {
		logger.Info("stopping http service")
		if err := srv.Shutdown(ctx); err != nil {
			logger.Error("error shutting http service down", zap.Error(err))
		}
	})

	quit := make(chan os.Signal)
	signal.Notify(quit, os.Interrupt)
	g.Add(func() error {
		logger.Info("starting signal listener")
		<-quit
		return nil
	}, func(error) {
		logger.Info("stopping signal listener")
		close(quit)
	})

	if err := g.Run(); err != nil {
		logger.Error("error caught", zap.Error(err))
	}
	return nil
}

func getDB(cliCtx *cli.Context) (*gorm.DB, error) {
	dbURI := cliCtx.String("database")
	if dbURI == "" {
		return nil, fmt.Errorf("invalid database configuration")
	}
	db, err := gorm.Open("postgres", dbURI)
	if err != nil {
		return nil, err
	}

	if err := db.DB().Ping(); err != nil {
		return nil, err
	}

	return db.Debug(), nil
}

func automigrate(cliCtx *cli.Context, db *gorm.DB, logger *zap.Logger) error {
	if cliCtx.Bool("init") {
		if err := db.Exec(`CREATE EXTENSION IF NOT EXISTS "uuid-ossp";`).Error; err != nil {
			return err
		}
		if err := db.Exec(`CREATE EXTENSION IF NOT EXISTS citext;`).Error; err != nil {
			return err
		}
		if err := db.AutoMigrate(&model.Actor{},
			&model.Activity{},
			&model.Graph{},
			&model.ActorActivity{},
			&model.Object{},
		).
			Error; err != nil {
			return err
		}

		users := strings.Split(cliCtx.String("users"), ";")
		domain := cliCtx.String("domain")
		for _, name := range users {
			actor, err := model.CreateActor(db, name, domain)
			if err != nil {
				return err
			}
			logger.Debug("actor initialized", zap.String("name", actor.Name), zap.String("id", actor.ID.String()))

			for _, name2 := range users {
				if name != name2 {
					graph, err := model.CreateGraphRel(db, string(model.NewActorID(name, domain)), string(model.NewActorID(name2, domain)))
					if err != nil {
						return err
					}
					logger.Debug("graph initialized", zap.String("from", graph.Follower), zap.String("to", graph.Actor))
				}
			}
		}
	}
	return nil
}
