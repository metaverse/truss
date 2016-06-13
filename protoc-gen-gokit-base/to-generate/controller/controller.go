package controller

import (
	stdlog "log"
	"os"

	"github.com/TuneLab/gob/protoc-gen-gokit-base/to-generate/entityhelper"
	"github.com/TuneLab/gob/protoc-gen-gokit-base/to-generate/pb"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/levels"
)

type Controller struct {
	EntityHelper *entityhelper.EntityHelper
}

var (
	logger levels.Levels
)

func init() {
	klogger := log.NewJSONLogger(os.Stdout)
	logger = levels.New(klogger)
	stdlog.SetFlags(0)                              // flags are handled by Go kit's logger
	stdlog.SetOutput(log.NewStdlibAdapter(klogger)) // redirect anything using stdlib log to us
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
