package controller

import (
	"github.com/TuneLab/gob/protoc-gen-gokit-base/to-generate/entityhelper"
	"github.com/TuneLab/gob/protoc-gen-gokit-base/to-generate/pb"
)

type Controller struct {
	EntityHelper *entityhelper.EntityHelper
}

func (c *Controller) GetEntityHelper() *entityhelper.EntityHelper {
	if c.EntityHelper == nil {
		c.EntityHelper = &entityhelper.EntityHelper{}
	}

	return c.EntityHelper
}

func (c *Controller) Status(req *pb.StatusRequest) (*pb.StatusResponse, error) {
	return &pb.StatusResponse{
		Status: pb.ServiceStatus_OK,
	}, nil
}

func (c *Controller) Ping(req *pb.PingRequest) (*pb.PingResponse, error) {
	return &pb.PingResponse{
		Status: pb.ServiceStatus_OK,
	}, nil
}
