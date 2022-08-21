package models

import (
	"encoding/json"
	"github.com/ethereum/go-ethereum/common"
	"math/big"
)

// Commit is an auto generated low-level Go binding around an user-defined struct.
type Commit struct {
	Author        common.Address `json:"author"`
	Commit        [32]byte       `json:"commit"`
	Block         *big.Int       `json:"block"`
	Seed          [32]byte       `json:"seed"`
	Revealed      bool           `json:"revealed"`
	VerifiedBlock *big.Int       `json:"verifiedblock"`
	Consumer      common.Address `json:"consumer"`
	Subsender     common.Address `json:"subuser"`
	SubBlock      *big.Int       `json:"subblock"`
	Substatus     uint8          `json:"substatus"`
}

func (c Commit) Bytes() []byte {
	d, _ := json.Marshal(c)
	return d
}

func CommitFromBytes(data []byte) *Commit {
	var c = &Commit{}
	json.Unmarshal(data, c)
	return c
}

type CommitHash []byte
type CommitHashList []CommitHash

func CommitListFromBytes(data []byte) CommitHashList {
	var list = []CommitHash{}
	json.Unmarshal(data, &list)
	return list
}
