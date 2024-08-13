package state

import (
	"context"
	"fmt"
	"github.com/danboykis/ishkur/config"
	"github.com/danboykis/ishkur/db"
	"github.com/danboykis/ishkur/routes"
	"net/http"
)

type States struct {
	Db         db.Db
	Config     *config.Config
	Version    *config.Version
	HttpServer *http.Server
}

func (ss *States) InitConfig() error {
	if ss.Config != nil {
		return nil
	}
	conf := config.Config{
		Port:    8080,
		Host:    "",
		LogPath: "/tmp/ishkur.log",
		Redis: config.Redis{
			Host:     "localhost",
			Port:     10010,
			Password: "Ky3ZR7v01DfuAnm4IdzHbI52bxI2PDs65v9ozY0zLSX19Bz56W",
		},
	}
	ss.Config = &conf
	return nil
}

func (ss *States) InitDb() error {
	if ss.Db != nil {
		return nil
	}
	dbConn, dbErr := db.NewRedis(ss.Config.Redis)
	if dbErr != nil {
		return fmt.Errorf("cannot start DB: %w", dbErr)
	}
	ss.Db = dbConn
	return nil
}

func (ss *States) InitHttpServer() error {
	if ss.HttpServer != nil {
		return nil
	}
	ss.HttpServer = routes.SetupHttpServer(ss.Config, *ss.Version, ss.Db)
	return nil
}

func (ss *States) StopDb(ctx context.Context) error {
	if ss.Db != nil {
		defer func() { ss.Db = nil }()
		return ss.Db.Close(ctx)
	}
	return nil
}

func (ss *States) StopHttpServer(ctx context.Context) error {
	if ss.HttpServer != nil {
		defer func() { ss.HttpServer = nil }()
		return ss.HttpServer.Shutdown(ctx)
	}
	return nil
}
