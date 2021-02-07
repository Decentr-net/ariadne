package main

import (
	"context"
	"fmt"
	"time"

	"github.com/Decentr-net/ariadne"

	decentr "github.com/Decentr-net/decentr/app"
)

func main() {
	nodeAddr := "http://zeus.testnet.decentr.xyz:26657"
	cdc := decentr.MakeCodec()

	fetcher, err := ariadne.New(nodeAddr, cdc, time.Minute)
	if err != nil {
		panic(err)
	}

	b, err := fetcher.FetchBlock(0) // Fetch one block(the highest block)
	if err != nil {
		panic(err)
	}

	fmt.Printf("messages from block %d: \n%+v\n\n", b.Height, b.Messages())

	fmt.Println("start fetching blocks")
	fmt.Println(fetcher.FetchBlocks(context.Background(), b.Height, func(b ariadne.Block) error {
		fmt.Printf("got new block %d. there are %d messages\n",
			b.Height,
			len(b.Messages()),
		)
		return nil
	},
		ariadne.WithErrHandler(func(h uint64, err error) {
			fmt.Printf("got an error on height %d: %s\n", h, err.Error())
		}),
		ariadne.WithRetryInterval(time.Second*2),
		ariadne.WithRetryLastBlockInterval(time.Second*5),
	).Error())
}
