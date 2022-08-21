package monitor

import (
	"context"
	"encoding/hex"
	"errors"
	"github.com/astaxie/beego"
	"github.com/astaxie/beego/logs"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/hpb-project/srng-robot/config"
	"github.com/hpb-project/srng-robot/contracts"
	"github.com/hpb-project/srng-robot/db"
	"github.com/hpb-project/srng-robot/utils"
	"golang.org/x/crypto/sha3"
	"math/big"
	"time"
)

type MonitorService struct {
	ctx context.Context
	ldb *db.LevelDB
	client *ethclient.Client
	conf config.Config
	oracleContract *contracts.Oracle

	user common.Address
	transopt *bind.TransactOpts
	callopt  *bind.CallOpts

	revealTask chan []byte
}

func NewMonitorService(config config.Config, ldb *db.LevelDB)  (*MonitorService,error) {
	ctx := context.Background()
	client, err := ethclient.Dial(config.NodeRPC)
	if err != nil {
		return nil, err
	}
	oracleAddr := common.HexToAddress(config.Oracle)
	oracle, err := contracts.NewOracle(oracleAddr, client)
	if err != nil {
		logs.Error("create oracle contract failed", "err", err)
		return nil, err
	}

	key,err := crypto.HexToECDSA(config.PrivKey)
	if err != nil {
		logs.Error("invalid private key")
		return nil, err
	}

	keyAddr := utils.PrivkToAddress(config.PrivKey)
	signer := types.LatestSignerForChainID(big.NewInt(int64(config.ChainId)))

	transopt := &bind.TransactOpts{
		From: keyAddr,
		Signer: func(address common.Address, tx *types.Transaction) (*types.Transaction, error) {
			if address != keyAddr {
				return nil, errors.New("not authorized account")
			}
			signature, err := crypto.Sign(signer.Hash(tx).Bytes(), key)
			if err != nil {
				return nil, err
			}
			return tx.WithSignature(signer, signature)
		},

	}
	transopt.GasPrice,_ = new(big.Int).SetString("5000000000", 10)
	transopt.GasLimit = 1000000

	callopt := &bind.CallOpts{
		Pending:     false,
		From:        keyAddr,
		BlockNumber: nil,
		Context:     ctx,
	}

	product := &MonitorService{
		ctx:ctx,
		oracleContract: oracle,
		ldb: ldb,
		user: keyAddr,
		conf: config,
		transopt: transopt,
		callopt: callopt,
		client:client,
		revealTask: make(chan []byte, 1000),
	}
	logs.Info("create monitor succeed")
	product.approvetoken(big.NewInt(10000000000))
	logs.Info("token approve finished")
	return product, nil
}

func (s MonitorService)approvetoken(amount *big.Int) error {
	var unit,_ = new(big.Int).SetString("1000000000000000000", 10)
	token,err := contracts.NewToken(common.HexToAddress(s.conf.Token), s.client)
	if err != nil {
		logs.Error("create token contracts failed", "err", err)
		return err
	}
	tx,err := token.Approve(s.transopt, common.HexToAddress(s.conf.Oracle), new(big.Int).Mul(amount, unit))
	if err != nil {
		logs.Error("approve token failed", "err",err)
		return err
	}
	receipt := s.waittx(tx)
	if receipt != nil && receipt.Status == 1 {
		logs.Info("approve token succeed")
		return nil
	} else {
		logs.Info("approve token failed")
		return errors.New("approve token failed")
	}
}

func (s MonitorService) waittx(tx *types.Transaction) *types.Receipt {
	ticker := time.NewTicker(time.Second*2)
	timeout := time.NewTimer(time.Second*30)
	logs.Debug("wait tx", "hash", tx.Hash())
	defer ticker.Stop()
	defer timeout.Stop()
	for {
		select {
		case <-ticker.C:
			r, err := s.client.TransactionReceipt(s.ctx, tx.Hash())
			if err != nil || r == nil {
				continue
			}
			return r

		case <-timeout.C:
			return nil
		}
	}
}

func (s MonitorService) doReveal(commit []byte) bool {
	var hash [32]byte
	var seed [32]byte

	value,exist := db.GetSeedBySeedHash(s.ldb, commit)
	if !exist {
		logs.Error("can't doreveal because not found seed", "commit", hex.EncodeToString(commit))
		return false
	}
	copy(hash[:], commit[:])
	copy(seed[:], value[:])
	tx,err := s.oracleContract.Reveal(s.transopt, hash, seed)
	if err != nil {
		logs.Error("tx reveal failed", "err", err)
		return false
	}
	logs.Info("do reveal", "hash", hex.EncodeToString(hash[:]))
	receipt := s.waittx(tx)
	if receipt != nil && receipt.Status == 1 {
		// successful
		return true
	} else {
		return false
	}
}

func (s MonitorService) DoCommit() error {
	r := append(s.user.Bytes(),utils.CryptoRandom()...)
	seed := sha3.Sum256(r)
	seedHash,err := s.oracleContract.GetHash(s.callopt, seed)
	if err != nil {
		beego.Error("get seed hash failed", "err", err)
		return err
	}
	db.SetSeedHashAndSeed(s.ldb, seedHash[:], seed[:])

	tx,err := s.oracleContract.Commit(s.transopt, seedHash)
	if err != nil {
		logs.Error("commit seed hash failed", "err", err)
		return err
	}
	logs.Info("do commit", "hash", hex.EncodeToString(seedHash[:]))
	receipt := s.waittx(tx)
	if receipt != nil && receipt.Status == 1 {
		db.SetUnRevealSeed(s.ldb, seedHash[:])
		return nil
	} else {
		return errors.New("commit failed")
	}
}

func (s MonitorService) DoReveal(commit []byte) {
	s.revealTask <- commit
}

func (s MonitorService) Run() {
	var uncommitmap = make(map[common.Hash]bool)
	// load unrevealed commit list from contract.
	uncommited, err := s.oracleContract.GetUserUnverifiedList(s.callopt, s.user)
	if err != nil {
		//
	}
	for _, cml := range uncommited {
		h := common.Hash{}
		h.SetBytes(cml.Commit[:])
		uncommitmap[h] = true
	}
	// load unreveal seed from db and merge with contract.
	unrevealed := db.GetAllUnReveald(s.ldb)
	for _, seedhash := range unrevealed {
		h := common.Hash{}
		h.SetBytes(seedhash[:])
		if _,exist := uncommitmap[h]; exist {
			if s.doReveal(seedhash) {
				db.DelUnRevealSeed(s.ldb, seedhash)
			}
		} else {
			db.DelUnRevealSeed(s.ldb, seedhash)
		}
	}
	tm := time.NewTicker(time.Second * 10)
	defer tm.Stop()

	for {
		select {
		case <- tm.C:
			if len(s.revealTask) < 10 {
				s.DoCommit()
			}
		case commit,ok := <-s.revealTask:
			if !ok {
				return
			}
			succeed := s.doReveal(commit)
			if succeed {
				db.DelUnRevealSeed(s.ldb, commit)
			}
		}

	}
}

