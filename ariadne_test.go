package ariadne

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Decentr-net/decentr/app"
	"github.com/Decentr-net/decentr/x/pdv"
)

const nodeAddr = "http://zeus.testnet.decentr.xyz:26657"

var testErrHandler = func(t *testing.T, cancel func()) func(uint64, error) {
	return func(_ uint64, err error) {
		cancel()
		assert.NoError(t, err)
	}
}

func TestFetcher_FetchBlock(t *testing.T) {
	t.Parallel()

	f, err := New(nodeAddr, app.MakeCodec(), time.Second)
	require.NoError(t, err)

	b, err := f.FetchBlock(3009)
	require.NoError(t, err)
	require.EqualValues(t, 3009, b.Height)
	require.False(t, b.Time.IsZero())
	require.Len(t, b.Txs, 1)
	require.Len(t, b.Txs[0].GetMsgs(), 1)

	require.Equal(t, "pdv", b.Txs[0].GetMsgs()[0].Route())
	require.Equal(t, "distribute_rewards", b.Txs[0].GetMsgs()[0].Type())
	msg, ok := b.Txs[0].GetMsgs()[0].(pdv.MsgDistributeRewards)
	require.True(t, ok)

	require.EqualValues(t, uint64(0x60329c38), msg.Rewards[0].ID)
}

func TestFetcher_FetchBlock_Last(t *testing.T) {
	t.Parallel()

	f, err := New(nodeAddr, app.MakeCodec(), time.Second)
	require.NoError(t, err)

	b, err := f.FetchBlock(0)
	require.NoError(t, err)
	require.NotZero(t, b.Height)
	require.False(t, b.Time.IsZero())
	require.NotNil(t, b.Txs)
}

func TestFetcher_FetchBlocks(t *testing.T) {
	t.Parallel()

	f, err := New(nodeAddr, app.MakeCodec(), time.Second)
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())

	err = f.FetchBlocks(ctx, 1000, func(b Block) error {
		require.NotZero(t, b.Height)
		require.False(t, b.Time.IsZero())

		if b.Height == 1002 {
			cancel()
		}

		return nil
	}, WithErrHandler(testErrHandler(t, cancel)))

	require.Equal(t, context.Canceled, err)
}

func TestFetcher_FetchBlocks_Retrying(t *testing.T) {
	t.Parallel()

	f, err := New(nodeAddr, app.MakeCodec(), time.Second)
	require.NoError(t, err)

	count := 0
	var errTest = errors.New("test")

	ctx, cancel := context.WithCancel(context.Background())
	err = f.FetchBlocks(ctx, 1000, func(b Block) error {
		// we shouldn't fetch block again
		fi, _ := f.(fetcher)
		fi.c = nil

		require.NotZero(t, b.Height)
		require.False(t, b.Time.IsZero())

		if count == 3 {
			cancel()
		}

		count++
		return errTest
	},
		WithRetryInterval(time.Nanosecond),
		WithErrHandler(func(_ uint64, err error) {
			require.Equal(t, errTest, err)
		}),
	)

	require.Equal(t, context.Canceled, err)
}

func TestFetcher_FetchBlocks_SkipFailed(t *testing.T) {
	t.Parallel()

	f, err := New(nodeAddr, app.MakeCodec(), time.Second)
	require.NoError(t, err)

	var h uint64

	var errTest = errors.New("test")

	ctx, cancel := context.WithCancel(context.Background())
	err = f.FetchBlocks(ctx, 1000, func(b Block) error {
		require.NotZero(t, b.Height)
		require.False(t, b.Time.IsZero())

		require.NotEqual(t, h, b.Height)

		if b.Height == 1001 {
			cancel()
		}

		h = b.Height

		return errTest
	},
		WithSkipError(true),
		WithRetryInterval(time.Nanosecond),
		WithErrHandler(func(_ uint64, err error) {
			require.Equal(t, errTest, err)
		}),
	)

	require.Equal(t, context.Canceled, err)
}

func TestBlock_Messages(t *testing.T) {
	t.Parallel()

	f, err := New(nodeAddr, app.MakeCodec(), time.Second)
	require.NoError(t, err)

	b, err := f.FetchBlock(3009)
	require.NoError(t, err)

	require.Len(t, b.Messages(), 1)
}

func TestWithErrHandler(t *testing.T) {
	t.Parallel()

	f, err := New("wrong", app.MakeCodec(), time.Second)
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())

	require.Equal(t, context.Canceled, f.FetchBlocks(ctx, 1000,
		func(b Block) error { return nil },
		WithErrHandler(func(height uint64, err error) {
			require.EqualValues(t, 1000, height)
			require.Error(t, err)
			cancel()
		}),
	))
}
