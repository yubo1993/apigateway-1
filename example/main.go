package main

import (
	"flag"
	"fmt"
	"git.henghajiang.com/backend/api_gateway_v2/sdk/golang"
	"github.com/coreos/etcd/clientv3"
	"github.com/gin-gonic/gin"
	"github.com/hhjpin/goutils/logger"
	_ "github.com/jinzhu/gorm/dialects/mysql"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"
)

var (
	flagPort = flag.Int("port", 7789, "server listening port")
)

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

func exit(gw *golang.ApiGatewayRegistrant) {
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)
	<-signalChan
	logger.Infof("service exiting ...")
	if err := gw.Unregister(); err != nil {
		logger.Error(err)
	}
	os.Exit(0)
}

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	flag.Parse()
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

	node := golang.NewNode("127.0.0.1", *flagPort, golang.NewHealthCheck("/check", 10, 5, 3, true))
	svr := golang.NewService("test", node)
	gw := golang.NewApiGatewayRegistrant(
		ConnectToEtcd(),
		node,
		svr,
		[]*golang.Router{
			golang.NewRouter("test1", "POST", "/front/$1", "/test/$1", svr),
			golang.NewRouter("test2", "GET", "/api/v1/test/$1", "/test/$1", svr),
			golang.NewRouter("test3", "POST", "/rd", "/redirect", svr),
		},
	)

	if err := gw.Register(); err != nil {
		logger.Error(err)
		return
	}

	go exit(&gw)
	if err := r.Run(fmt.Sprintf(":%d", *flagPort)); err != nil {
		logger.Error(err)
	}
}
