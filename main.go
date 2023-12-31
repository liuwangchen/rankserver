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
	"github.com/liuwangchen/toy/registry"
	"github.com/liuwangchen/toy/third_party/redisx"
	"github.com/nats-io/nats.go"
	clientv3 "go.etcd.io/etcd/client/v3"
)

var (
	serviceName = "rank"
	hash        = "43b543"
	serviceId   = "1"
	namespace   = "test"
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
		WebAddr:     cfg.Rank.Static.Web,
	})
	if err != nil {
		return nil, err
	}
	runners = append(runners, st)

	return runners, nil
}

func run() error {
	etcdClient, err := clientv3.NewFromURL(config.GetInstance().Common.EtcdAddr)
	if err != nil {
		return err
	}
	runners, err := initMain()
	if err != nil {
		return err
	}
	cfg := config.GetInstance()
	a := app.New(
		app.WithPProf(cfg.Common.PprofAddr),
		app.WithID(serviceId),
		app.WithName(serviceName),
		app.WithVersion(hash),
		app.WithRegistrar(registry.NewEtcdRegistry(etcdClient, registry.Namespace(namespace))),
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
