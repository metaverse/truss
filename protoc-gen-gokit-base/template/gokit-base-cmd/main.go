package main

import (
	"fmt"
	stdlog "log"
	"math/rand"
	"net"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/levels"

	"github.com/TuneLab/gob/protoc-gen-gokit-base/to-generate/controller"
	"github.com/TuneLab/gob/protoc-gen-gokit-base/to-generate/pb"
	"github.com/TuneLab/gob/protoc-gen-gokit-base/to-generate/server"

	"google.golang.org/grpc"
)

func main() {

	// Set up logging for errors and other details
	var logger levels.Levels
	{
		// Log to Stdout
		klogger := log.NewJSONLogger(os.Stdout)
		logger = levels.New(klogger)
		// Take normal logs and put them into logger
		stdlog.SetFlags(0)                              // flags are handled by Go kit's logger
		stdlog.SetOutput(log.NewStdlibAdapter(klogger)) // redirect anything using stdlib log to us
	}

	// All fatal errors go on this channel
	errc := make(chan error)

	// Take system interupts and pass them to the errc channel to be logged (ex. Ctrl-C)
	go func() {
		errc <- interrupt()
	}()

	// Take logs out of the errc channel and log them crit
	defer logger.Crit().Log("fatal", <-errc)

	// Note that math/rand is seeded by rand.Seed(1) unless changed at some point in execution
	rand.Seed(time.Now().UnixNano())

	// Hook up controller
	ctrl := &controller.Controller{}

	// Business domain
	var svc server.CurrencyExchangeService
	{
		svc = pureCurrencyExchangeService{ctrl}
	}

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
