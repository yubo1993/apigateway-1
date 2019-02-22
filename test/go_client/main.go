package main

import (
	"flag"
	"fmt"
	"git.henghajiang.com/backend/api_gateway_v2/sdk/golang"
	"git.henghajiang.com/backend/golang_utils/log"
	"github.com/coreos/etcd/clientv3"
	"github.com/gin-gonic/gin"
	_ "github.com/jinzhu/gorm/dialects/mysql"
	"runtime"
	"time"
)

var (
	logger   = log.New()
	flagPort = flag.Int("port", 7789, "server listening port")
)

func init() {
	flag.Parse()
	node := golang.NewNode("127.0.0.1", *flagPort, golang.NewHealthCheck(
		"/check",
		10,
		5,
		3,
		true,
	))
	logger.Infof("flagPort： %d", *flagPort)
	logger.Infof("node port: %d", node.Port)
	svr := golang.NewService("test", node)
	gw := golang.NewApiGatewayRegistrant(
		ConnectToEtcd(),
		node,
		svr,
		[]*golang.Router{
			golang.NewRouter("test1", "/front/$1", "/test/$1", svr),
			golang.NewRouter("test2", "/api/v1/test/$1", "/test/$1", svr),
			golang.NewRouter("test3", "/rd", "/redirect", svr),
		},
	)
	if err := gw.Register(); err != nil {
		logger.Exception(err)
	}
}

func ConnectToEtcd() *clientv3.Client {
	cli, _ := clientv3.New(
		clientv3.Config{
			Endpoints:            []string{"127.0.0.1:2379"},
			AutoSyncInterval:     time.Duration(0) * time.Second,
			DialTimeout:          time.Duration(3) * time.Second,
			DialKeepAliveTime:    time.Duration(30) * time.Second,
			DialKeepAliveTimeout: time.Duration(5) * time.Second,
			Username:             "",
			Password:             "",
		},
	)
	return cli
}

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())

	r := gin.New()
	r.Use(gin.Logger())

	r.POST("/test/:test_id", func(c *gin.Context) {
		c.JSON(200, map[string]interface{}{"result": "success", "test_id": *flagPort})
	})
	r.GET("/check", func(c *gin.Context) {
		c.JSON(200, map[string]string{"result": "success"})
	})
	r.POST("/redirect", func(c *gin.Context) {
		c.Redirect(308, "http://127.0.0.1/test/redirect")
	})

	if err := r.Run(fmt.Sprintf(":%d", *flagPort)); err != nil {
		logger.Exception(err)
	}
}
