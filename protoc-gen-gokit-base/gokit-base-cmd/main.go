package main

import (
	"fmt"
	"math/rand"
	"net"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/hasAdamr/gokit-base/controller"
	"github.com/hasAdamr/gokit-base/pb"
	"github.com/hasAdamr/gokit-base/server"

	"google.golang.org/grpc"
)

const (
	SERVICE_ORG  = "labdev"
	SERVICE_NAME = "currency_exchange"
)

func main() {

	// Hook up controller
	ctrl := &controller.Controller{}

	// Business domain
	var svc server.CurrencyExchangeService
	{
		svc = pureCurrencyExchangeService{ctrl}
	}

	// Mechanical stuff
	rand.Seed(time.Now().UnixNano())
	errc := make(chan error)

	go func() {
		errc <- interrupt()
	}()

	// Transport: gRPC
	if eval := os.Getenv("GRPC_PORT"); eval != "" {
		grpcPrt, _ := strconv.Atoi(eval)

		go func() {
			grpcAddr := fmt.Sprintf(":%d", grpcPrt)

			gopts := []grpc.ServerOption{}
			grpcServer := grpc.NewServer(gopts...) // uses its own, internal context

			ln, err := net.Listen("tcp", grpcAddr)
			if err != nil {
				errc <- err
				return
			}

			pb.RegisterCurrencyExchangeServiceServer(grpcServer, grpcBinding{svc})
			errc <- grpcServer.Serve(ln)
		}()
	}

}

func interrupt() error {
	c := make(chan os.Signal)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
	return fmt.Errorf("%s", <-c)
}
