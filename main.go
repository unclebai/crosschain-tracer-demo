package main

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/binance-chain/go-sdk/client/rpc"
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
	crossPackages, err := e.GetCrossChainPackages(286545068)
	if err != nil {
		panic(err)
	}
	bz, _ := json.MarshalIndent(crossPackages, "\t", "\t")
	fmt.Println(string(bz))
}

// The `content` of CrossChainPackage is now defined as interface.
// Actually its actually type is different according to different type
func (e *Executor) GetCrossChainPackages(height int64) ([]sdkMsg.CrossChainPackage, error) {
	block, err := e.rpcClient.Block(&height)
	if err != nil {
		return nil, err
	}
	blockResults, err := e.rpcClient.BlockResults(&height)
	if err != nil {
		return nil, err
	}
	var crossPackages []sdkMsg.CrossChainPackage
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
					cp, err := sdkMsg.ParseClaimPayload(msg.Payload)
					if err != nil {
						return nil, err
					}
					crossPackages = append(crossPackages, cp...)
				}
			}
		}
	}
	return crossPackages, nil
}

// This function is used to pase 32bytes symbol to string
func BytesToSymbol(symbolBytes [32]byte) string {
	tokenSymbolBytes := make([]byte, 32)
	copy(tokenSymbolBytes[:], symbolBytes[:])
	return string(bytes.Trim(tokenSymbolBytes, "\x00"))
}
