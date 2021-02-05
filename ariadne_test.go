package ariadne

import (
	"context"
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

	b, err := f.FetchBlock(1000)
	require.NoError(t, err)
	require.EqualValues(t, 1000, b.Height)
	require.False(t, b.Time.IsZero())
	require.Len(t, b.Txs, 1)
	require.Len(t, b.Txs[0].GetMsgs(), 1)

	require.Equal(t, "pdv", b.Txs[0].GetMsgs()[0].Route())
	require.Equal(t, "create_pdv", b.Txs[0].GetMsgs()[0].Type())
	msg, ok := b.Txs[0].GetMsgs()[0].(pdv.MsgCreatePDV)
	require.True(t, ok)

	require.EqualValues(t, 1610732375, msg.Timestamp)
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
	count := 0

	for b := range f.FetchBlocks(ctx, 1000, WithErrHandler(testErrHandler(t, cancel))) {
		require.NotZero(t, b.Height)
		require.False(t, b.Time.IsZero())
		count++

		if count == 2 {
			cancel() // finish test
		}
	}
}

func TestBlock_Messages(t *testing.T) {
	t.Parallel()

	f, err := New(nodeAddr, app.MakeCodec(), time.Second)
	require.NoError(t, err)

	b, err := f.FetchBlock(100000)
	require.NoError(t, err)

	require.Len(t, b.Messages(), 8)
}
func TestWithErrHandler(t *testing.T) {
	t.Parallel()

	f, err := New("wrong", app.MakeCodec(), time.Second)
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())

	for range f.FetchBlocks(ctx, 1000,
		WithErrHandler(func(height uint64, err error) {
			require.EqualValues(t, 1000, height)
			require.Error(t, err)
			cancel()
		}),
	) {
	}
}
