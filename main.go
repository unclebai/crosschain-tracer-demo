package main

import (
	"encoding/json"
	"fmt"

	"github.com/binance-chain/go-sdk/client/rpc"
	"github.com/binance-chain/go-sdk/common/rlp"
	"github.com/binance-chain/go-sdk/common/types"
	sdkMsg "github.com/binance-chain/go-sdk/types/msg"
	"github.com/binance-chain/go-sdk/types/tx"
)

const (
	TxEventTypeClaim = "claim"
)

func NewExecutor() *Executor {
	c := rpc.NewRPCClient("tcp://dataseed1.defibit.io:80", types.ProdNetwork)
	return &Executor{
		rpcClient: c,
	}
}

type Executor struct {
	rpcClient *rpc.HTTP
}

func main() {
	e := NewExecutor()
	err := e.GetTransferInPackages(286545068)
	if err != nil {
		panic(err)
	}
}

func (e *Executor) GetTransferInPackages(height int64) error {
	block, err := e.rpcClient.Block(&height)
	if err != nil {
		return err
	}
	blockResults, err := e.rpcClient.BlockResults(&height)
	if err != nil {
		return err
	}
	for idx, t := range block.Block.Data.Txs {
		txResult := blockResults.Results.DeliverTx[idx]
		if txResult.Code != 0 {
			continue
		}

		stdTx, err := rpc.ParseTx(tx.Cdc, t)
		if err != nil {
			fmt.Errorf("parse tx error, err=%s", err.Error())
			continue
		}

		msgs := stdTx.GetMsgs()
		for _, msg := range msgs {
			switch msg := msg.(type) {
			case sdkMsg.ClaimMsg:
				events := blockResults.Results.DeliverTx[idx].Events
				var isExecutedClaim bool
				for _, event := range events {
					// Only the last OracleClaim transaction get such an event
					if event.Type == TxEventTypeClaim {
						isExecutedClaim = true
						break
					}
				}
				if isExecutedClaim {
					err = parseInterestedClaimPayload(msg.Payload)
					if err != nil {
						panic(err)
					}
				}

			}
		}
	}
	return nil
}

func parseInterestedClaimPayload(payload []byte) error {
	packages := sdkMsg.Packages{}
	err := rlp.DecodeBytes(payload, &packages)
	if err != nil {
		return err
	}
	for _, pack := range packages {
		// The channel for transfer fund from BSC to BC
		// Please refer to https://github.com/bnb-chain/go-sdk/blob/7f0fb6a81fb64565e6c8676c7b335d4ef6e9e177/types/msg/msg-oracle.go#L119
		// to fetch more interested packages.
		if pack.ChannelId != 3 {
			continue
		}
		var unpackage sdkMsg.TransferInSynPackage
		err = rlp.DecodeBytes(pack.Payload[sdkMsg.PackageHeaderLength:], &unpackage)
		if err != nil {
			return err
		}
		bz, _ := json.MarshalIndent(unpackage, "\t", "\t")
		fmt.Println(string(bz))
	}
	return nil
}
