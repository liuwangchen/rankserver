package main

import (
	"context"
	"flag"

	"github.com/liuwangchen/rankserver/components"
	"github.com/liuwangchen/rankserver/config"
	"github.com/liuwangchen/rankserver/logic"
	"github.com/liuwangchen/rankserver/rpc"
	"github.com/liuwangchen/rankserver/service"
	"github.com/liuwangchen/toy/app"
	"github.com/liuwangchen/toy/pkg/singlethread"
	"github.com/liuwangchen/toy/third_party/redisx"
	"github.com/nats-io/nats.go"
)

func initMain() ([]app.Runner, error) {
	cfg := config.GetInstance()
	natsConn, err := nats.Connect(cfg.Common.NatsAddr)
	if err != nil {
		return nil, err
	}

	st := singlethread.NewST()

	redisClient, err := redisx.NewAsyncWithConfig(redisx.Config{
		Server: cfg.Common.RedisAddr,
	}, st.Async())
	if err != nil {
		return nil, err
	}
	components.RedisClient = redisClient

	err = logic.GetRankManagerInstance().Init(context.Background())
	if err != nil {
		return nil, err
	}

	// 初始化rpc
	runners, err := rpc.Init(rpc.Param{
		Conn:        natsConn,
		RankService: service.NewRankService(),
		Namespace:   cfg.Common.Namespace,
		As:          st.Async(),
		WebAddr:     cfg.Static.Web,
	})
	if err != nil {
		return nil, err
	}
	runners = append(runners, st)

	return runners, nil
}

func run() error {
	runners, err := initMain()
	if err != nil {
		return err
	}
	cfg := config.GetInstance()
	a := app.New(
		app.WithPProf(cfg.Common.PprofAddr),
		app.WithRunners(runners...),
	)

	return a.Run(context.Background())
}

func main() {
	// 初始化配置
	configPath := flag.String("f", "", "")
	flag.Parse()

	cfg := config.GetInstance()
	if err := cfg.Load(*configPath); err != nil {
		panic(err)
	}
	err := run()
	if err != nil {
		panic(err)
	}
}
