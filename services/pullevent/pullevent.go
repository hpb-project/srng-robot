package pullevent

import (
	"context"
	"github.com/astaxie/beego/logs"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/hpb-project/srng-robot/config"
	"github.com/hpb-project/srng-robot/db"
	"github.com/hpb-project/srng-robot/utils"
	"github.com/prometheus/common/log"
	"math/big"
	"time"
)

const (
	LastSyncBlockKey = "lastSyncBlock"
)

var (
	bigOne     = big.NewInt(1)
	bigTen     = big.NewInt(10)
	bighundred = big.NewInt(100)
	bigK       = big.NewInt(1000)
)

type Worker interface {
	NewCommit() error
	Reveal(commit []byte) error
}

type logHandler func(log types.Log, pe *PullEvent, addr common.Address, history bool) error

type PullEvent struct {
	ctx             context.Context
	client          *ethclient.Client
	lastBlock       *big.Int
	ldb             *db.LevelDB
	oracle          common.Address
	user            common.Address
	contractHandler logHandler
	work 			Worker
}

func NewPullEvent(config config.Config, ldb *db.LevelDB, w Worker) *PullEvent {
	lastBlock := big.NewInt(0)
	value, exist := ldb.Get([]byte(LastSyncBlockKey))
	if exist {
		lastBlock.SetBytes(value)
	}
	client, err := ethclient.Dial(config.NodeRPC)
	if err != nil {
		logs.Error("pull event create client failed", "err", err)
		return nil
	}
	pe := &PullEvent{
		ctx:             context.Background(),
		lastBlock:       lastBlock,
		oracle:          common.HexToAddress(config.Oracle),
		user:            utils.PrivkToAddress(config.PrivKey),
		contractHandler: OracleContractHandler,
		client:          client,
		ldb: ldb,
		work: w,
	}
	logs.Info("create pull evnet succeed")
	return pe
}

func (p *PullEvent) GetLogs() {
	query := ethereum.FilterQuery{}
	query.FromBlock = p.lastBlock
	query.ToBlock = new(big.Int).Add(p.lastBlock, big.NewInt(1))
	query.Addresses = []common.Address{p.oracle}

	if p.lastBlock.Int64() == 0 {
		txhash := "0x0cce1507429f709fa77d8e59c795c5e67aa6e7f601f70ad249f97b38f9c681c0" // product contract
		receipt,_ := p.client.TransactionReceipt(p.ctx, common.HexToHash(txhash))
		p.lastBlock = receipt.BlockNumber
	}
	for {
		query.FromBlock = p.lastBlock

		log.Info("start fileter start at ", p.lastBlock.Text(10))
		history := false
		height, err := p.client.BlockNumber(p.ctx)
		if height <= p.lastBlock.Uint64() {
			time.Sleep(time.Second)
			continue
		} else if (height - 1000) >= p.lastBlock.Uint64() {
			query.ToBlock = new(big.Int).Add(p.lastBlock, bigK)
			history = true
		} else if (height - 100) >= p.lastBlock.Uint64() {
			query.ToBlock = new(big.Int).Add(p.lastBlock, bighundred)
			history = true
		} else if (height - 10) >= p.lastBlock.Uint64() {
			query.ToBlock = new(big.Int).Add(p.lastBlock, bigTen)
		} else {
			query.ToBlock = new(big.Int).Add(p.lastBlock, bigOne)
		}

		allLogs, err := p.client.FilterLogs(p.ctx, query)
		if err != nil {
			log.Error("filter logs failed", err)
			continue
		}
		if len(allLogs) > 0 {
			for _, vlog := range allLogs {
				if p.contractHandler != nil {
					p.contractHandler(vlog, p, vlog.Address, history)
				}
			}
		}
		p.ldb.Set([]byte(LastSyncBlockKey), p.lastBlock.Bytes())
		p.lastBlock = new(big.Int).Add(query.ToBlock, bigOne)
	}
}
