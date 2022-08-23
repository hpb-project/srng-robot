package monitor

import (
	"context"
	"crypto/ecdsa"
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
	"sync"
	"time"
)

type MonitorService struct {
	ctx context.Context
	ldb *db.LevelDB
	client *ethclient.Client
	signer types.Signer
	privk *ecdsa.PrivateKey
	conf config.Config
	oracleContract *contracts.Oracle
	muxnonce sync.Mutex
	nonce uint64

	user common.Address
	callopt  *bind.CallOpts

	waitmux sync.Mutex
	waittoreveal [][]byte

	revealTask chan []byte
}
const (
	MAX_UNVERIFY_BLOCK = 400 // todo: change to read from config contract.
)

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

	callopt := &bind.CallOpts{
		Pending:     false,
		From:        keyAddr,
		BlockNumber: nil,
		Context:     ctx,
	}

	nonce,err := client.NonceAt(ctx, keyAddr, nil)
	if err != nil {
		logs.Error("can't get user nonce", "err", err)
	}

	product := &MonitorService{
		ctx:ctx,
		oracleContract: oracle,
		ldb: ldb,
		user: keyAddr,
		conf: config,
		callopt: callopt,
		client:client,
		signer: signer,
		privk: key,
		nonce: nonce,
		waittoreveal: make([][]byte,0),
		revealTask: make(chan []byte, 1000),
	}
	logs.Info("create monitor succeed")
	product.approvetoken(big.NewInt(10000000000))
	logs.Info("token approve finished")
	return product, nil
}

func (s MonitorService)getnonce() uint64 {
	s.muxnonce.Lock()
	defer s.muxnonce.Unlock()
	var result uint64
	chain,_ := s.client.NonceAt(s.ctx, s.user, nil)
	if chain > s.nonce {
		result = chain
		s.nonce = chain + 1
	} else {
		result = s.nonce
		s.nonce += 1
	}
	return result
}

func (s MonitorService)getTransopt() *bind.TransactOpts {
	transopt := &bind.TransactOpts{
		From: s.user,
		Signer: func(address common.Address, tx *types.Transaction) (*types.Transaction, error) {
			if address != s.user {
				return nil, errors.New("not authorized account")
			}
			signature, err := crypto.Sign(s.signer.Hash(tx).Bytes(), s.privk)
			if err != nil {
				return nil, err
			}
			return tx.WithSignature(s.signer, signature)
		},
		Nonce: new(big.Int).SetUint64(s.getnonce()),
	}
	transopt.GasPrice,_ = new(big.Int).SetString("5000000000", 10)
	transopt.GasLimit = 1000000

	return transopt
}

func (s MonitorService)approvetoken(amount *big.Int) error {
	var unit,_ = new(big.Int).SetString("1000000000000000000", 10)
	token,err := contracts.NewToken(common.HexToAddress(s.conf.Token), s.client)
	if err != nil {
		logs.Error("create token contracts failed", "err", err)
		return err
	}
	tx,err := token.Approve(s.getTransopt(), common.HexToAddress(s.conf.Oracle), new(big.Int).Mul(amount, unit))
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

func (s MonitorService) doReveal(commit []byte, needcheck bool) bool {
	var hash [32]byte
	var seed [32]byte

	value,exist := db.GetSeedBySeedHash(s.ldb, commit)
	if !exist {
		logs.Error("can't doreveal because not found seed", "commit", hex.EncodeToString(commit))
		return true
	}
	copy(hash[:], commit[:])
	copy(seed[:], value[:])

	tx,err := s.oracleContract.Reveal(s.getTransopt(), hash, seed)
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

func (s MonitorService) AddToRevealAgain(commit []byte) {
	s.waitmux.Lock()
	defer s.waitmux.Unlock()

	s.waittoreveal = append(s.waittoreveal, commit)
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

	tx,err := s.oracleContract.Commit(s.getTransopt(), seedHash)
	if err != nil {
		logs.Error("commit seed hash failed", "err", err)
		return err
	}
	logs.Info("do commit", "hash", hex.EncodeToString(seedHash[:]))
	receipt := s.waittx(tx)
	if receipt == nil || receipt.Status == 1 {
		// wait timeout or commit succeed
		db.SetUnRevealSeed(s.ldb, seedHash[:])
	}
	return nil
}

func (s MonitorService) DoReveal(commit []byte) {
	s.revealTask <- commit
}

func (s MonitorService) MergeRecord(waittoreveal [][]byte) [][]byte {
	var needtoreveal = make([][]byte,0)
	var needtorevealmap = make(map[common.Hash]bool)

	var uncommitmap = make(map[common.Hash]contracts.Commit)
	curblock,_ := s.client.BlockNumber(s.ctx)

	// load unrevealed commit list from contract.
	uncommited, err := s.oracleContract.GetUserUnverifiedList(s.callopt, s.user)
	if err != nil {
		//
	}
	for _, cml := range uncommited {
		h := common.Hash{}
		h.SetBytes(cml.Commit[:])
		uncommitmap[h] = cml
	}

	for _, seedhash := range waittoreveal {
		h := common.Hash{}
		h.SetBytes(seedhash[:])
		if _,exist := needtorevealmap[h]; exist {
			continue
		}

		if info,exist := uncommitmap[h]; exist {
			if (info.Block.Int64() + MAX_UNVERIFY_BLOCK) <= int64(curblock) {
				// timeout
			} else {
				needtorevealmap[h] = true
				needtoreveal = append(needtoreveal, h.Bytes())
			}
		} else {
			// wait commit can find in contract.
		}
	}
	return needtoreveal
}

func (s MonitorService) Run() {
	needreveal := s.MergeRecord(db.GetAllUnReveald(s.ldb))
	for _, r := range needreveal {
		s.doReveal(r, false)
	}

	committicker := time.NewTicker(time.Second * 15)
	defer committicker.Stop()

	revealticker := time.NewTicker(time.Second * 20)
	defer revealticker.Stop()

	go func() {
		for {
			select {
			case commit,ok := <-s.revealTask:
				if !ok {
					return
				}
				succeed := s.doReveal(commit,false)
				if succeed {
					db.DelUnRevealSeed(s.ldb, commit)
				} else {
					s.AddToRevealAgain(commit)
				}
			}
		}
	}()

	for {
		select {
		case <- committicker.C:
			if len(s.revealTask) < 10 {
				s.DoCommit()
			}

		case <- revealticker.C:
			unrevealed := db.GetAllUnReveald(s.ldb)
			s.waitmux.Lock()
			unrevealed = append(unrevealed,s.waittoreveal...)
			s.waittoreveal = make([][]byte,0)
			s.waitmux.Unlock()
			needreveal := s.MergeRecord(unrevealed)
			for _, r := range needreveal {
				s.DoReveal(r)
			}
		}
	}
}

