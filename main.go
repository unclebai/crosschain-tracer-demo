package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	types2 "github.com/cosmos/cosmos-sdk/x/oracle/types"

	"github.com/bnb-chain/go-sdk/client/rpc"
	"github.com/bnb-chain/go-sdk/common/types"
	sdkMsg "github.com/bnb-chain/go-sdk/types/msg"
	"github.com/bnb-chain/go-sdk/types/tx"
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
	crossPackages, err := e.GetCrossChainPackages(158104713)
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
			case types2.ClaimMsg:
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
					// TODO: in most cases, the cross chain package will be handled successfully.
					// In some other case, the cross chain app has to deal with exception, like refund token if the
					// receiver is not allowed to receive cross chain transfer. Give the most concerned cross chain transfer in
					// packages, there are two kind of tags:
					// 1. transferInSuccess_{symbol}_{receive addr}: {amount}; means cross chain transfer in package is successfully handled;
					// 2. transferInRefund_{symbol}_{receive addr}: {amount}; means cross chain transfer in package failed to be handled, have to refund the token to BSC.
					// Reference code: https://github.com/bnb-chain/node/blob/4d97f955e9e7ac369d2cbb33181763239d6cdf42/plugins/bridge/cross_app.go#L477
					//for _, tag := range blockResults.Results.DeliverTx[idx].Tags {
					//	// Filter your interested tags
					//}
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
