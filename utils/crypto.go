package utils

import (
	"crypto/ecdsa"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

func PrivkToAddress(privk string) common.Address {
	privateKey, err := crypto.HexToECDSA(privk)
	if err != nil {
		return common.Address{}
	}
	publicKey := privateKey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		return common.Address{}
	}

	return crypto.PubkeyToAddress(*publicKeyECDSA)
}

func CryptoRandom() []byte {
	pk, _ := crypto.GenerateKey()
	return pk.D.Bytes()
}
