package server

import (
	"github.com/hasAdamr/gokit-base/pb"
)

type CurrencyExchangeService interface {
	ExchangeRateGetRate(req *pb.ExchangeRateGetRateRequest) (*pb.ExchangeRateGetRateResponse, error)
	ExchangeRateConvert(req *pb.ExchangeRateConvertRequest) (*pb.ExchangeRateConvertResponse, error)
	Status(req *pb.StatusRequest) (*pb.StatusResponse, error)
	Ping(req *pb.PingRequest) (*pb.PingResponse, error)
}
