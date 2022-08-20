package product

import (
	"context"
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
)

type ProductService struct {
	ctx context.Context
	ldb *db.LevelDB
	oracleContract *contracts.Oracle

	user common.Address
	transopt *bind.TransactOpts
	callopt  *bind.CallOpts

	revealTask chan common.Hash
}

func NewProductService(config config.Config, ldb *db.LevelDB)  (*ProductService,error) {
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

	product := &ProductService{
		ctx:ctx,
		oracleContract: oracle,
		ldb: ldb,
		user: keyAddr,
		transopt: transopt,
		callopt: callopt,
		revealTask: make(chan common.Hash, 10),
	}
	return product, nil
}

func (s ProductService) DoReveal(commit common.Hash) {
	s.revealTask <- commit
}

func (s ProductService) DoCommit() {
	r := append(s.user.Bytes(),utils.CryptoRandom()...)
	seed := sha3.Sum256(r)
	seedHash,err := s.oracleContract.GetHash(s.callopt, seed)
	if err != nil {
		beego.Error("get seed hash failed", "err", err)
		return
	}
	// todo: store seed and seedhash to leveldb.

	//
	tx,err := s.oracleContract.Commit(s.transopt, seedHash)
	if err != nil {
		logs.Error("commit seed hash failed", "err", err)
		return
	}
	// todo: store seed hash and txhash.
	tx.Hash()
}

func (s ProductService) Run() {
	// 1. loop get block header.
	// 2. find commit to reveal.
	// 3. make new commit.
	for {

	}
}
