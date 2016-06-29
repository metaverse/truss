package main

import (
	"{{.AbsoluteRelativeImportPath}}pb"
	"{{.AbsoluteRelativeImportPath}}server"
	"golang.org/x/net/context"
)

type grpcBinding struct {
	server.{{.Service.GetName}}
}

{{range $i := .Service.Methods}}
func (b grpcBinding) {{$i.GetName}}(ctx context.Context, in *pb.{{$i.RequestType.GetName}}) (*pb.{{$i.ResponseType.GetName}}, error) {
	ctx = context.WithValue(ctx, "transport", "grpc")
	ctx = context.WithValue(ctx, "request-method", "{{$i.RequestType.GetName}}")
	return b.{{.Service.GetName}}.{{$i.GetName}}(in)
}
{{end}}


// ORGINAL FOR COMPARISON
// Note that on the context.WithValue for request-method that "Request" is stripped off the string, that may need to be done

//func (b grpcBinding) ExchangeRateGetRate(ctx context.Context, in *pb.ExchangeRateGetRateRequest) (*pb.ExchangeRateGetRateResponse, error) {
	//ctx = context.WithValue(ctx, "transport", "grpc")
	//ctx = context.WithValue(ctx, "request-method", "ExchangeRateGetRate")
	//return b.CurrencyExchangeService.ExchangeRateGetRate(in)
//}

//func (b grpcBinding) ExchangeRateConvert(ctx context.Context, in *pb.ExchangeRateConvertRequest) (*pb.ExchangeRateConvertResponse, error) {
	//ctx = context.WithValue(ctx, "transport", "grpc")
	//ctx = context.WithValue(ctx, "request-method", "ExchangeRateConvert")
	//return b.CurrencyExchangeService.ExchangeRateConvert(in)
//}

//func (b grpcBinding) Status(ctx context.Context, in *pb.StatusRequest) (*pb.StatusResponse, error) {
	//ctx = context.WithValue(ctx, "transport", "grpc")
	//ctx = context.WithValue(ctx, "request-method", "Status")
	//return b.CurrencyExchangeService.Status(in)
//}

//func (b grpcBinding) Ping(ctx context.Context, in *pb.PingRequest) (*pb.PingResponse, error) {
	//ctx = context.WithValue(ctx, "transport", "grpc")
	//ctx = context.WithValue(ctx, "request-method", "Ping")
	//return b.CurrencyExchangeService.Ping(in)
//}
