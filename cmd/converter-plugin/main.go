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
