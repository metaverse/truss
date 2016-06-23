package main

import (
	"github.com/TuneLab/gob/protoc-gen-gokit-base/generate/pb"
	"github.com/TuneLab/gob/protoc-gen-gokit-base/generate/server"
	"golang.org/x/net/context"
)

type grpcBinding struct {
	server.CurrencyExchangeService
}

func (b grpcBinding) ExchangeRateGetRate(ctx context.Context, in *pb.ExchangeRateGetRateRequest) (*pb.ExchangeRateGetRateResponse, error) {
	ctx = context.WithValue(ctx, "transport", "grpc")
	ctx = context.WithValue(ctx, "request-method", "ExchangeRateGetRate")
	return b.CurrencyExchangeService.ExchangeRateGetRate(in)
}

func (b grpcBinding) ExchangeRateConvert(ctx context.Context, in *pb.ExchangeRateConvertRequest) (*pb.ExchangeRateConvertResponse, error) {
	ctx = context.WithValue(ctx, "transport", "grpc")
	ctx = context.WithValue(ctx, "request-method", "ExchangeRateConvert")
	return b.CurrencyExchangeService.ExchangeRateConvert(in)
}

func (b grpcBinding) Status(ctx context.Context, in *pb.StatusRequest) (*pb.StatusResponse, error) {
	ctx = context.WithValue(ctx, "transport", "grpc")
	ctx = context.WithValue(ctx, "request-method", "Status")
	return b.CurrencyExchangeService.Status(in)
}

func (b grpcBinding) Ping(ctx context.Context, in *pb.PingRequest) (*pb.PingResponse, error) {
	ctx = context.WithValue(ctx, "transport", "grpc")
	ctx = context.WithValue(ctx, "request-method", "Ping")
	return b.CurrencyExchangeService.Ping(in)
}
