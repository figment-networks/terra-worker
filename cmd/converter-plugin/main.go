package main

import (
	"fmt"
	"io"

	"github.com/figment-networks/indexer-manager/structs"
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

func DecodeEvents(logger *zap.Logger, txReader, txLogReader io.Reader) structs.TransactionEvents {
	fmt.Println("[DecodeEvents] called")
	return cli.GetEventsFromRaw(logger, txReader, txLogReader)
}
