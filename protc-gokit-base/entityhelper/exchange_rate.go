package entityhelper

import (
	"database/sql"
	"fmt"

	"github.com/hasAdamr/gokit-base/mv-shared/mysql"
	"github.com/hasAdamr/gokit-base/pb"
)

func (c *EntityHelper) FindExchangeRates(whereQstr string, args []interface{}, resetCache bool) (*pb.ExchangeRateEntities, error) {

	table := "exchange_rates"

	qStr := fmt.Sprintf(`SELECT
	                IFNULL(id, 0) AS id,
	                IFNULL(currency_code, 0) AS currency_code,
	                IFNULL(rate, 0) AS rate,
	                IFNULL(date, "") AS date,
	                CAST(IFNULL(created, "") AS char) AS created,
	                CAST(IFNULL(modified, "") AS char) AS modified
	            FROM %s`, table)

	if whereQstr != "" {
		qStr = qStr + " WHERE " + whereQstr
	}

	res := &pb.ExchangeRateEntities{
		Results: []*pb.ExchangeRateEntity{},
	}

	var mfn mysql.MapRow = func(rows *sql.Rows) (interface{}, error) {
		entry := &pb.ExchangeRateEntity{}
		err := rows.Scan(&entry.Id, &entry.CurrencyCode, &entry.Rate, &entry.Date, &entry.Created, &entry.Modified)
		return entry, err
	}

	dbClient, err := c.GetMysqlClient()
	if err != nil {
		return nil, err
	}
	defer dbClient.Close()

	entries, err := mysql.QueryRows(dbClient, qStr, args, mfn)

	ids := []interface{}{}
	for _, ientry := range entries {
		entry := ientry.(*pb.ExchangeRateEntity)
		res.Results = append(res.Results, entry)
		ids = append(ids, entry.Id)
	}

	return res, err
}
