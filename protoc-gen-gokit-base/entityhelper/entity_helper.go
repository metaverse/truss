package entityhelper

import (
	stdlog "log"
	"os"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/levels"
	"github.com/hasAdamr/gokit-base/mv-shared/mysql"
)

type EntityHelper struct {
	*mysql.Client
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

func (c *EntityHelper) GetMysqlClient() (*mysql.Client, error) {
	if c.Client == nil {
		client, err := mysql.NewClient()
		if err != nil {
			return nil, err
		}
		c.Client = client
	}
	return c.Client, nil
}
