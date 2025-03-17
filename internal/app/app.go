package app

import (
	"context"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"go.uber.org/zap"
	"math/big"
)

type App struct {
	l         *zap.SugaredLogger
	ethClient EthClient
}

func New(l *zap.SugaredLogger, ethClient EthClient) *App {
	return &App{
		l:         l,
		ethClient: ethClient,
	}
}

type EthClient interface {
	BalanceAt(ctx context.Context, account common.Address, blockNumber *big.Int) (*big.Int, error)
	ethereum.BlockNumberReader
}

func (a App) GetBalance(ctx context.Context, address string) (string, error) {
	b, err := a.ethClient.BalanceAt(ctx, common.HexToAddress(address), nil)
	if err != nil {
		return "", err
	}

	return b.String(), nil
}

func (a App) HealthCheck(ctx context.Context) error {
	_, err := a.ethClient.BlockNumber(ctx)
	if err != nil {
		return err
	}

	return nil
}
