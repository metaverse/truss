package main

import (
	"github.com/hasAdamr/gokit-base/controller"
	"github.com/hasAdamr/gokit-base/pb"
)

type pureCurrencyExchangeService struct {
	*controller.Controller
}

func (p pureCurrencyExchangeService) ExchangeRateGetRate(req *pb.ExchangeRateGetRateRequest) (*pb.ExchangeRateGetRateResponse, error) {
	res, err := p.Controller.ExchangeRateGetRate(req)
	if res == nil {
		res = &pb.ExchangeRateGetRateResponse{}
	}
	return res, err
}

func (p pureCurrencyExchangeService) ExchangeRateConvert(req *pb.ExchangeRateConvertRequest) (*pb.ExchangeRateConvertResponse, error) {
	res, err := p.Controller.ExchangeRateConvert(req)
	if res == nil {
		res = &pb.ExchangeRateConvertResponse{}
	}
	return res, err
}

func (p pureCurrencyExchangeService) Status(req *pb.StatusRequest) (*pb.StatusResponse, error) {
	res, err := p.Controller.Status(req)
	if res == nil {
		res = &pb.StatusResponse{
			Status: pb.ServiceStatus_FAIL,
		}
	}
	return res, err
}

func (p pureCurrencyExchangeService) Ping(req *pb.PingRequest) (*pb.PingResponse, error) {
	res, err := p.Controller.Ping(req)
	if res == nil {
		res = &pb.PingResponse{}
	}
	return res, err
}
