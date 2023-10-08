package rpc

// ----------

import (
	"github.com/liuwangchen/apis/apipb"
	"github.com/liuwangchen/rankserver/config"
	"github.com/liuwangchen/toy/app"
	"github.com/liuwangchen/toy/pkg/async"
	"github.com/liuwangchen/toy/transport/middleware/recovery"
	"github.com/liuwangchen/toy/transport/rpc/httprpc"
	"github.com/liuwangchen/toy/transport/rpc/natsrpc"
	"github.com/nats-io/nats.go"
)

type Param struct {
	Conn        *nats.Conn
	RankService apipb.RankServerNatsService
	Namespace   string
	As          async.IAsync
	WebAddr     string
}

// Init 注册消息入口
func Init(param Param) ([]app.Runner, error) {
	natsServerConn, err := natsrpc.NewServerConn(
		natsrpc.WithConn(param.Conn),
		natsrpc.WithNamespace(param.Namespace),
		natsrpc.WithConnMiddleware(recovery.Recovery()),
	)
	if err != nil {
		return nil, err
	}

	httpServerConn, err := httprpc.NewServerConn(
		httprpc.WithAddress(param.WebAddr),
		httprpc.WithConnMiddleware(recovery.Recovery()),
	)
	if err != nil {
		return nil, err
	}

	// 以rankType排行榜类型来注册service
	rankTypeRange := config.GetInstance().Rank.Dynamic.TypeRange
	for i := rankTypeRange[0]; i <= rankTypeRange[1]; i++ {
		err = apipb.RegisterAsyncRankServerNatsServer(natsServerConn, param.As, param.RankService, natsrpc.WithServiceTopic(i))
		if err != nil {
			return nil, err
		}
	}

	apipb.RegisterAsyncRankServerHTTPServer(httpServerConn, param.As, param.RankService)

	return []app.Runner{natsServerConn, httpServerConn}, nil
}
