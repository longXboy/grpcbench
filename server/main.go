package main

import (
	"flag"
	"log"
	"net"
	"net/http"
	_ "net/http/pprof"
	"sync/atomic"
	"time"

	"bench/proto"
	"go-common/net/rpc/warden"
	xtime "go-common/time"

	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

var reqNum uint64

type Hello struct{}

func (t *Hello) Say(ctx context.Context, args *proto.BenchmarkMessage) (reply *proto.BenchmarkMessage, err error) {
	s := "OK"
	var i int32 = 100
	args.Field1 = s
	args.Field2 = i
	atomic.AddUint64(&reqNum, 1)
	return args, nil
}

var host = flag.String("s", "0.0.0.0:8972", "listened ip and port")
var isWarden = flag.Bool("w", true, "is warden or grpc client")

func main() {
	go func() {
		log.Println("run http at :6060")
		log.Println(http.ListenAndServe("0.0.0.0:6060", nil))
	}()

	flag.Parse()

	go stat()
	if *isWarden {
		runWarden()
	} else {
		runGrpc()
	}
}

func runGrpc() {
	log.Println("run grpc")
	lis, err := net.Listen("tcp", *host)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	s := grpc.NewServer(grpc.InitialWindowSize(65535*1000), grpc.InitialConnWindowSize(65535*10000))

	proto.RegisterHelloServer(s, &Hello{})
	s.Serve(lis)
}

func runWarden() {
	log.Println("run warden")
	s := warden.NewServer(
		&warden.ServerConfig{Timeout: xtime.Duration(time.Second * 3)},
		grpc.InitialWindowSize(65535*1000),
		grpc.InitialConnWindowSize(65535*10000),
	)
	proto.RegisterHelloServer(s.Server(), &Hello{})
	s.Run(*host)
}

func stat() {
	ticker := time.NewTicker(time.Second * 5)
	defer ticker.Stop()
	var last uint64
	lastTs := uint64(time.Now().UnixNano())
	for {
		<-ticker.C
		now := atomic.LoadUint64(&reqNum)
		nowTs := uint64(time.Now().UnixNano())
		qps := (now - last) * 1e6 / ((nowTs - lastTs) / 1e3)
		last = now
		lastTs = nowTs
		log.Println("qps:", qps)
	}
}
