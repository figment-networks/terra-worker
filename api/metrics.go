package api

import "github.com/figment-networks/indexing-engine/metrics"

var (
	conversionDuration = metrics.MustNewHistogramWithTags(metrics.HistogramOptions{
		Namespace: "indexerworker",
		Subsystem: "api",
		Name:      "conversion_duration",
		Desc:      "Duration how long it takes to convert from raw to format",
		Tags:      []string{"type"},
	})

	rawRequestHTTPDuration = metrics.MustNewHistogramWithTags(metrics.HistogramOptions{
		Namespace: "indexerworker",
		Subsystem: "api",
		Name:      "request_http",
		Desc:      "Duration how long it takes to take data from chain",
		Tags:      []string{"endpoint", "status"},
	})

	rawRequestGRPCDuration = metrics.MustNewHistogramWithTags(metrics.HistogramOptions{
		Namespace: "indexerworker",
		Subsystem: "api",
		Name:      "request_grpc",
		Desc:      "Duration how long it takes to take data from chain",
		Tags:      []string{"endpoint", "status"},
	})

	numberOfItems = metrics.MustNewCounterWithTags(metrics.Options{
		Namespace: "indexerworker",
		Subsystem: "api",
		Name:      "tx_num",
		Desc:      "Number of all transactions returned from one request",
		Tags:      []string{"type"},
	})

	numberOfItemsBlock = metrics.MustNewCounterWithTags(metrics.Options{
		Namespace: "indexerworker",
		Subsystem: "api",
		Name:      "block_tx_num",
		Desc:      "Number of all transactions returned from one request",
		Tags:      []string{"type"},
	})

	unknownTransactions = metrics.MustNewCounterWithTags(metrics.Options{
		Namespace: "indexerworker",
		Subsystem: "api",
		Name:      "tx_unknown",
		Desc:      "Number of unknown transactions",
		Tags:      []string{"type"},
	})

	brokenTransactions = metrics.MustNewCounterWithTags(metrics.Options{
		Namespace: "indexerworker",
		Subsystem: "api",
		Name:      "tx_broken",
		Desc:      "Number of broken transactions",
		Tags:      []string{"type"},
	})

	numberOfItemsTransactions     *metrics.GroupCounter
	numberOfItemsInBlock          *metrics.GroupCounter
	transactionConversionDuration *metrics.GroupObserver
)

/*
var (
	conversionDuration = metrics.MustNewHistogramWithTags(metrics.HistogramOptions{
		Namespace: "indexers",
		Subsystem: "worker_api_terra",
		Name:      "conversion_duration",
		Desc:      "Duration how long it takes to convert from raw to format",
		Tags:      []string{"type"},
	})

	rawRequestDuration = metrics.MustNewHistogramWithTags(metrics.HistogramOptions{
		Namespace: "indexers",
		Subsystem: "worker_api_terra",
		Name:      "request_duration",
		Desc:      "Duration how long it takes to take data from terra",
		Tags:      []string{"endpoint", "status"},
	})

	numberOfItems = metrics.MustNewHistogramWithTags(metrics.HistogramOptions{
		Namespace: "indexers",
		Subsystem: "worker_api_terra",
		Name:      "number_of_items",
		Desc:      "Number of all transactions returned from one request",
		Tags:      []string{"type"},
	})

	blockCacheEfficiency = metrics.MustNewCounterWithTags(metrics.Options{
		Namespace: "indexers",
		Subsystem: "worker_api_terra",
		Name:      "block_cache_efficiency",
		Desc:      "How Efficient the shared block cache is",
		Tags:      []string{"cache"},
	})

	blockCacheEfficiencyHit       *metrics.GroupCounter
	blockCacheEfficiencyMissed    *metrics.GroupCounter
	numberOfItemsTransactions     *metrics.GroupObserver
	transactionConversionDuration *metrics.GroupObserver
	convertionDurationObserver    *metrics.GroupObserver
)
*/
