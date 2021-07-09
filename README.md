# TERRA WORKER

This repository contains a worker part dedicated for cosmos transactions.

# Compatibility
Works with `columbus-3` and `columbus-4` chains
## Worker
Stateless worker is responsible for connecting with the chain, getting information, converting it to a common format and sending it back to manager.
Worker can be connected with multiple managers but should always answer only to the one that sent request.

## API
Implementation of bare requests for network.

### Client
Worker's business logic wiring of messages to client's functions.


## Installation
This system can be put together in many different ways.
This readme will describe only the simplest one worker, one manager with embedded scheduler approach.

### Compile
To compile sources you need to have go 1.14.1+ installed.

```bash
    make build
```

### Running
Worker also need some basic config:

```bash
    MANAGERS=0.0.0.0:8085
    TERRA_RPC_ADDR=https://terra-4.address
    DATAHUB_KEY=1QAZXSW23EDCvfr45TGB
    CHAIN_ID=columbus-4
```

Where
    - `TERRA_RPC_ADDR` is a http address to node's RPC endpoint
    - `MANAGERS` a comma-separated list of manager ip:port addresses that worker will connect to. In this case only one

After running both binaries worker should successfully register itself to the manager.

If you wanna connect with manager running on docker instance add `HOSTNAME=host.docker.internal` (this is for OSX and Windows). For linux add your docker gateway address taken from ifconfig (it probably be the one from interface called docker0).


## Developing Locally

First, you will need to set up a few dependencies:

1. [Install Go](https://golang.org/doc/install)
2. A running [indexer-manager](https://github.com/figment-networks/indexer-manager) instance
3. A running datastore API instance (configured with `STORE_HTTP_ENDPOINTS`).

Then, run the worker with some environment config:

```
CHAIN_ID=columbus-4 \
STORE_HTTP_ENDPOINTS=http://127.0.0.1:8986/input/jsonrpc \
TERRA_RPC_ADDR=https://columbus-4--rpc--archive.datahub.figment.io \
TERRA_LCD_ADDR=https://columbus-4--lcd--archive.datahub.figment.io \
DATAHUB_KEY={YOUR KEY HERE} \
go run ./cmd/terra-worker
```
## Transaction Types
List of currently supporter transaction types in terra-worker are (listed by modules):
- bank:
    `multisend` , `send`
- crisis:
    `verify_invariant`
- distribution:
    `withdraw_validator_commission` , `set_withdraw_address` , `withdraw_delegator_reward` , `fund_community_pool`
- evidence:
    `submit_evidence`
- gov:
    `deposit` , `vote` , `submit_proposal`
- market:
    `swap` , `swapsend`
- msgauth:
    `grant_authorization` , `revoke_authorization` , `exec_delegated`
- oracle:
    `exchangeratevote` , `exchangerateprevote` , `delegatefeeder` , `aggregateexchangerateprevote` , `aggregateexchangeratevote`
- slashing:
    `unjail`
- staking:
    `begin_unbonding` , `edit_validator` , `create_validator` , `delegate` , `begin_redelegate`
- wasm:
    `execute_contract`, `store_code`, `update_contract_owner` , `instantiate_contract` , `migrate_contract`
- internal:
    `error`
