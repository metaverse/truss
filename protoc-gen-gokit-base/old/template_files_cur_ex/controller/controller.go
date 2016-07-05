package controller

import (
	"fmt"
	"strings"

	"{{.AbsoluteRelativeImportPath}}entityhelper"
	"{{.AbsoluteRelativeImportPath}}pb"
)

type Controller struct {
	EntityHelper *entityhelper.EntityHelper
}

var lastLoaded string

type OpenExchangeResult struct {
	Base      string             `json:"base"`
	Timestamp int64              `json:"timestamp"`
	Rates     map[string]float32 `json:"rates"`
}

func (c *Controller) GetEntityHelper() *entityhelper.EntityHelper {
	if c.EntityHelper == nil {
		c.EntityHelper = &entityhelper.EntityHelper{}
	}

	return c.EntityHelper
}

func FormatCurrencyCode(str string) string {
	return strings.ToUpper(str)
}

func (c *Controller) ExchangeRateGetRate(req *pb.ExchangeRateGetRateRequest) (*pb.ExchangeRateGetRateResponse, error) {
	req.FromCurrency = FormatCurrencyCode(req.FromCurrency)
	req.ToCurrency = FormatCurrencyCode(req.ToCurrency)

	whereStr := `currency_code IN (?, ?)`
	args := []interface{}{
		req.FromCurrency,
		req.ToCurrency,
	}

	if req.Date != "" {
		whereStr = whereStr + " AND date = ? AND deleted IS NULL"
		args = append(args, req.Date)
	} else {
		whereStr = whereStr + " AND deleted IS NULL ORDER BY date DESC LIMIT 4"
	}

	rates, err := c.GetEntityHelper().FindExchangeRates(whereStr, args, false)
	if err != nil {
		return nil, fmt.Errorf("ExchangeRateGetRate - %v", err)
	}

	var fromRate float32
	for _, r := range rates.Results {
		if r.CurrencyCode == req.FromCurrency {
			fromRate = float32(r.Rate)
			break
		}
	}

	var toRate float32
	for _, r := range rates.Results {
		if r.CurrencyCode == req.ToCurrency {
			toRate = float32(r.Rate)
			break
		}
	}

	if fromRate == 0.0 && toRate == 0.0 {
		return nil, fmt.Errorf("ExchangeRateGetRate - Unable to load rate for from %s and to %s", req.FromCurrency, req.ToCurrency)
	} else if fromRate == 0.0 {
		return nil, fmt.Errorf("ExchangeRateGetRate - Unable to load rate for from %s", req.FromCurrency)
	} else if toRate == 0.0 {
		return nil, fmt.Errorf("ExchangeRateGetRate - Unable to load rate for to %s", req.ToCurrency)
	}

	toRateInt := int(toRate * 1000000)
	fromRateInt := int(fromRate * 1000000)

	return &pb.ExchangeRateGetRateResponse{
		FromCurrency: req.FromCurrency,
		ToCurrency:   req.ToCurrency,
		Date:         req.Date,
		Rate:         float32((float32(toRateInt) / float32(fromRateInt))),
	}, nil
}

func (c *Controller) ExchangeRateConvert(req *pb.ExchangeRateConvertRequest) (*pb.ExchangeRateConvertResponse, error) {
	req.FromCurrency = FormatCurrencyCode(req.FromCurrency)
	req.ToCurrency = FormatCurrencyCode(req.ToCurrency)

	// if not rate specific load the rate
	if req.Rate == 0.0 {
		res, err := c.ExchangeRateGetRate(&pb.ExchangeRateGetRateRequest{FromCurrency: req.FromCurrency, ToCurrency: req.ToCurrency, Date: req.Date})
		if err != nil {
			return nil, err
		}
		req.Rate = float32(res.Rate)
	}

	return &pb.ExchangeRateConvertResponse{
		FromCurrency: req.FromCurrency,
		ToCurrency:   req.ToCurrency,
		Date:         req.Date,
		Rate:         req.Rate,
		Amount:       req.Amount,
		Result:       float32(req.Amount / req.Rate),
	}, nil
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
