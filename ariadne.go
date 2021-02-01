// Package ariadne is a library for fetching blocks from cosmos based blockchain node.
package ariadne

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth/types"
	rpcclient "github.com/tendermint/tendermint/rpc/client"
	rpchttp "github.com/tendermint/tendermint/rpc/client/http"
	jsonrpcclient "github.com/tendermint/tendermint/rpc/jsonrpc/client"
)

// ErrTooHighBlockRequested returned when blockchain's height is less than requested.
var ErrTooHighBlockRequested = errors.New("too high block requested")

// Block presents transactions and height.
// If you need have more information open new issue on github or DIY and send pull request.
type Block struct {
	Height uint64
	Txs    []sdk.Tx
}

//go:generate mockgen -destination=./mock/ariadne_mock.go -package=mock -source=ariadne.go

// Fetcher interface for fetching.
type Fetcher interface {
	// FetchBlocks starts fetching routine and returns a channel for result.
	// As FetchBlocks can't make multiple requests to node the channel doesn't have a buffer.
	FetchBlocks(ctx context.Context, from uint64, opts ...FetchBlocksOption) <-chan Block
	// FetchBlock fetches block from blockchain.
	// If height is zero then the highest block will be requested.
	FetchBlock(height uint64) (*Block, error)
}

type fetcher struct {
	c rpcclient.Client
	d sdk.TxDecoder
}

// New returns new instance of fetcher.
func New(node string, cdc *codec.Codec, timeout time.Duration) (Fetcher, error) {
	httpClient, err := jsonrpcclient.DefaultHTTPClient(node)
	if err != nil {
		return nil, err
	}
	httpClient.Timeout = timeout

	client, err := rpchttp.NewWithClient(node, "/websocket", httpClient)

	if err != nil {
		return nil, err
	}

	return fetcher{
		c: client,
		d: types.DefaultTxDecoder(cdc),
	}, nil
}

// FetchBlocks starts fetching routine and returns a channel for result.
// As FetchBlocks can't make multiple requests to node the channel doesn't have a buffer.
func (f fetcher) FetchBlocks(ctx context.Context, from uint64, opts ...FetchBlocksOption) <-chan Block {
	cfg := defaultFetchBlockOptions
	for _, v := range opts {
		v(&cfg)
	}

	height := uint64(1)
	if from > 0 {
		height = from
	}

	ch := make(chan Block)
	go func() {
		for {
			select {
			case <-ctx.Done():
				close(ch)
				return
			default:
				b, err := f.FetchBlock(height)

				if err != nil {
					if errors.Is(err, ErrTooHighBlockRequested) {
						time.Sleep(cfg.retryLastBlockInterval)
						continue
					}

					cfg.errHandler(height, fmt.Errorf("failed to get block: %w", err))
					time.Sleep(cfg.retryInterval)
					continue
				}

				height++
				ch <- *b
			}
		}
	}()

	return ch
}

// FetchBlock fetches block from blockchain.
// If height is zero then the highest block will be requested.
func (f fetcher) FetchBlock(height uint64) (*Block, error) {
	var h *int64
	if height > 0 {
		h = new(int64)
		*h = int64(height)
	}

	res, err := f.c.Block(h)
	if err != nil {
		if strings.Contains(err.Error(), "must be less than or equal") {
			return nil, ErrTooHighBlockRequested
		}
		return nil, err
	}

	txs := make([]sdk.Tx, len(res.Block.Txs))
	for i, v := range res.Block.Txs {
		tx, err := f.d(v)
		if err != nil {
			return nil, fmt.Errorf("failed to decode tx: %w", err)
		}
		txs[i] = tx
	}

	return &Block{
		Height: uint64(res.Block.Height),
		Txs:    txs,
	}, nil
}

// Messages returns all messages in all transactions.
func (b Block) Messages() []sdk.Msg {
	msgs := make([]sdk.Msg, 0, len(b.Txs))
	for _, tx := range b.Txs {
		msgs = append(msgs, tx.GetMsgs()...)
	}

	return msgs
}

// FilterMessages returns filtered slice with messages defined in types slices
// If types slice is empty no one message will be returned.
func FilterMessages(msgs []sdk.Msg, types ...reflect.Type) []sdk.Msg {
	out := make([]sdk.Msg, 0, len(msgs))

	for _, msg := range msgs {
		msgT := reflect.TypeOf(msg)

		for _, t := range types {
			if msgT == t {
				out = append(out, msg)
				break
			}
		}
	}

	return out
}
