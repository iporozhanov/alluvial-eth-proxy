package eth_client

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"go.uber.org/zap"
	"math"
	"math/big"
	"sync"
	"time"
)

func NewEthClient(url string) (*ethclient.Client, error) {
	client, err := ethclient.Dial(url)
	if err != nil {
		return nil, err
	}

	return client, nil
}

type EthClient interface {
	ethereum.BlockNumberReader
	BalanceAt(ctx context.Context, account common.Address, blockNumber *big.Int) (*big.Int, error)
	Close()
}

type HealthCheckErrors []string

func (h HealthCheckErrors) Error() string {
	b, _ := json.Marshal(h)
	return string(b)
}

const maxBlockDiff = 3

type NodeProxy struct {
	clients     map[string]EthClient
	unhealthy   map[string]EthClient
	mu          sync.RWMutex
	stop        chan bool
	l           *zap.SugaredLogger
	clientCount int
}

func NewNodeProxy(logger *zap.SugaredLogger) *NodeProxy {
	np := &NodeProxy{
		clients:   make(map[string]EthClient),
		unhealthy: make(map[string]EthClient),
		l:         logger,
	}

	return np
}

func (np *NodeProxy) Start() {

	np.l.Infof("started with %d clients", len(np.clients))

	timer := time.NewTicker(time.Second * 10)
	go func() {
		for {
			select {
			case <-timer.C:
				np.checkUnhealthy()
			case <-np.stop:
				timer.Stop()
				return
			}
		}
	}()
}

func (np *NodeProxy) Close() {
	np.stop <- true
	for _, client := range np.clients {
		client.Close()
	}
	for _, client := range np.unhealthy {
		client.Close()
	}

}

func (np *NodeProxy) BalanceAt(ctx context.Context, account common.Address, blockNumber *big.Int) (*big.Int, error) {
	c, index := np.getClient()
	if c == nil {
		return nil, errors.New("no healthy client found")
	}

	balance, err := c.BalanceAt(ctx, account, blockNumber)
	if err != nil && !errors.Is(err, context.Canceled) {
		np.setUnhealthy(index, c)

		return np.BalanceAt(ctx, account, blockNumber)
	}

	return balance, nil
}

func (np *NodeProxy) BlockNumber(ctx context.Context) (uint64, error) {
	np.mu.RLock()
	blockNumbers := make(map[string]uint64)
	unhealthy := make(map[string]EthClient)
	var lowest, highest uint64
	errStr := make(HealthCheckErrors, 0)
	for i, client := range np.clients {
		block, err := client.BlockNumber(ctx)
		if err != nil && !errors.Is(err, context.Canceled) {
			unhealthy[i] = client
			errStr = append(errStr, err.Error())
			continue
		}

		if lowest < block {
			lowest = block
		}

		if block > highest {
			highest = block
		}

		np.l.Infof("healthy client %s at height %d", i, block)
		blockNumbers[i] = block
	}

	if math.Abs(float64(highest-lowest)) > maxBlockDiff {
		for i, block := range blockNumbers {
			if block == lowest {
				errStr = append(errStr, fmt.Sprintf("client %s is unhealthy due to block difference", i))
				unhealthy[i] = np.clients[i]
			}
		}
	}

	defer np.mu.RUnlock()
	for i, c := range unhealthy {
		np.setUnhealthy(i, c)
	}

	for i, _ := range np.unhealthy {
		errStr = append(errStr, fmt.Sprintf("client %s is unhealthy", i))
	}

	if len(errStr) > 0 {
		return highest, fmt.Errorf(errStr.Error())
	}

	return highest, nil

}

func (np *NodeProxy) AddClient(url string, client EthClient) {
	np.mu.Lock()
	defer np.mu.Unlock()

	np.clients[url] = client
	np.clientCount++
	np.l.Infof("added new node with url: %s", url)
}

func (np *NodeProxy) setUnhealthy(index string, client EthClient) {
	np.mu.Lock()
	defer np.mu.Unlock()

	np.l.Warnf("unhealthy client %s", index)
	np.unhealthy[index] = client
	delete(np.clients, index)
}

func (np *NodeProxy) checkUnhealthy() {
	np.mu.Lock()
	defer np.mu.Unlock()
	if len(np.unhealthy) == 0 {
		return
	}

	unsetUnhealthy := make([]string, 0, len(np.unhealthy))

	for i, client := range np.unhealthy {
		_, err := client.BlockNumber(context.Background())
		if err != nil {
			np.clients[i] = client
			unsetUnhealthy = append(unsetUnhealthy, i)
		}
	}

	for _, i := range unsetUnhealthy {
		np.l.Warnf("client %s moved to healthy status", i)
		delete(np.unhealthy, i)
	}
}

func (np *NodeProxy) getClient() (EthClient, string) {
	np.mu.RLock()
	defer np.mu.RUnlock()

	if len(np.clients) == 0 {
		return nil, ""
	}

	for i, client := range np.clients {
		return client, i
	}
	return nil, ""
}
