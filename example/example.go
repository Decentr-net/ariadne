package main

import (
	"context"
	"fmt"
	"time"

	"github.com/Decentr-net/ariadne"
)

func main() {
	nodeAddr := "zeus.testnet.decentr.xyz:9090"

	fetcher, err := ariadne.New(context.Background(), nodeAddr, time.Minute)
	if err != nil {
		panic(err)
	}

	b, err := fetcher.FetchBlock(context.Background(), 0) // Fetch one block(the highest block)
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
