package main

import (
	"io"

	"github.com/figment-networks/terra-worker/api"
	"go.uber.org/zap"
)

var cli *api.Client

func init() {
	cli = api.NewClient("", "", nil, nil, 0)
}

func DecodeFee(logger *zap.Logger, reader io.Reader) []map[string]interface{} {
	return cli.GetFromRaw(logger, reader)
}

func DecodeEvents(logger *zap.Logger, txReader, txLogReader io.Reader) ([]interface{}, error) {
	events, err := cli.GetEventsFromRaw(logger, txReader, txLogReader)
	if err != nil {
		return nil, err
	}

	slice := []interface{}{}
	for _, e := range events {
		slice = append(slice, e)
	}
	return slice, nil
}
