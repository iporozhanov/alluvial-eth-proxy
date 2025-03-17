package eth_client

import (
	"context"
	"errors"
	"eth-proxy/internal/eth_client/mocks"
	"github.com/ethereum/go-ethereum/common"
	"github.com/go-playground/assert/v2"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
	"math/big"
	"testing"
)

func TestNodeProxy_AddClient(t *testing.T) {
	nodeProxy := NewNodeProxy(zap.S())

	client := mocks.NewEthClient(t)

	nodeProxy.AddClient("https://mainnet.infura.io/v3/ef391c6c612f48f88cae26bc256487be", client)
	assert.Equal(t, nodeProxy.clients["https://mainnet.infura.io/v3/ef391c6c612f48f88cae26bc256487be"], client)
}

func TestNodeProxy_SetUnhealthy(t *testing.T) {
	nodeProxy := NewNodeProxy(zap.S())
	client := mocks.NewEthClient(t)
	nodeProxy.AddClient("https://mainnet.infura.io/v3/ef391c6c612f48f88cae26bc256487be", client)
	assert.Equal(t, nodeProxy.clients["https://mainnet.infura.io/v3/ef391c6c612f48f88cae26bc256487be"], client)

	nodeProxy.setUnhealthy("https://mainnet.infura.io/v3/ef391c6c612f48f88cae26bc256487be", client)
	assert.Equal(t, nodeProxy.unhealthy["https://mainnet.infura.io/v3/ef391c6c612f48f88cae26bc256487be"], client)
	assert.Equal(t, nodeProxy.clients["https://mainnet.infura.io/v3/ef391c6c612f48f88cae26bc256487be"], nil)
}

func TestNodeProxy_SetHealthy(t *testing.T) {
	nodeProxy := NewNodeProxy(zap.S())
	client := mocks.NewEthClient(t)
	client.EXPECT().BlockNumber(context.Background()).Return(uint64(0), nil)
	nodeProxy.unhealthy["https://mainnet.infura.io/v3/ef391c6c612f48f88cae26bc256487be"] = client

	nodeProxy.checkUnhealthy()
}

func TestNodeProxy_BalanceAt(t *testing.T) {

	tests := []struct {
		name    string
		balance *big.Int
		err     error
		clients []struct {
			address string
			balance *big.Int
			err     error
		}
	}{
		{
			"1 client",
			big.NewInt(10),
			nil,
			[]struct {
				address string
				balance *big.Int
				err     error
			}{
				{
					address: "https://mainnet.infura.io/v3/ef391c6c612f48f88cae26bc256487be",
					balance: big.NewInt(10),
					err:     nil,
				},
			},
		},
		{
			"2 clients",
			big.NewInt(10),
			nil,
			[]struct {
				address string
				balance *big.Int
				err     error
			}{
				{
					address: "https://mainnet.infura.io/v3/ef391c6c612f48f88cae26bc256487be",
					balance: big.NewInt(10),
					err:     nil,
				},
				{
					address: "https://mainnet.infura.io/v3/ef391c6c612f48f88cae26bc256",
					balance: nil,
					err:     errors.New("test error"),
				},
			},
		},
		{
			"1 unhealthy client",
			nil,
			errors.New("no healthy client found"),
			[]struct {
				address string
				balance *big.Int
				err     error
			}{
				{
					address: "https://mainnet.infura.io/v3/ef391c6c612f48f88cae26bc256",
					balance: nil,
					err:     errors.New("test error"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nodeProxy := NewNodeProxy(zap.S())

			for _, client := range tt.clients {
				c := mocks.NewEthClient(t)
				c.EXPECT().
					BalanceAt(context.Background(), common.HexToAddress("0x1"), mock.Anything).
					Return(client.balance, client.err).Maybe()
				nodeProxy.AddClient(client.address, c)
			}
			balance, err := nodeProxy.BalanceAt(context.Background(), common.HexToAddress("0x1"), nil)
			assert.Equal(t, balance, tt.balance)
			assert.Equal(t, tt.err, err)
		})
	}
}

func TestNodeProxy_FailedRequestMovesClientToUnhealthy(t *testing.T) {
	nodeProxy := NewNodeProxy(zap.S())
	client := mocks.NewEthClient(t)
	client.EXPECT().BalanceAt(context.Background(), common.HexToAddress("0x1"), mock.Anything).Return(nil, errors.New("test error"))
	nodeProxy.AddClient("fail.com", client)
	_, err := nodeProxy.BalanceAt(context.Background(), common.HexToAddress("0x1"), nil)
	assert.Equal(t, err, errors.New("no healthy client found"))

	assert.Equal(t, len(nodeProxy.clients), 0)
	assert.Equal(t, len(nodeProxy.unhealthy), 1)

	assert.Equal(t, nodeProxy.unhealthy["fail.com"], client)
}
