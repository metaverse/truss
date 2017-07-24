package handlers

import (
	pb "github.com/TuneLab/truss/cmd/_integration-tests/middlewares/middlewarestest-service"
)

func WrapService(in pb.MiddlewaresTestServer) pb.MiddlewaresTestServer {
	return in
}
