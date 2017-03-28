package middlewares

import (
	pb "github.com/TuneLab/go-truss/cmd/_integration-tests/middlewares/middlewarestest-service"
)

func WrapService(in pb.MiddlewaresTestServer) pb.MiddlewaresTestServer {
	return in
}
