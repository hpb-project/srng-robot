package db


const (
	prefixSeedHashAndSeed = "kss"
	prefixSeedHashAndTx   = "kst"
	prefixSeedHashAndCommit = "ksm"
)

func keySeedHashAndSeed(hash []byte) []byte {
	return append([]byte(prefixSeedHashAndSeed), hash...)
}

func keySeedHashAndTx(hash []byte) []byte {
	return append([]byte(prefixSeedHashAndTx), hash...)
}

func keySeedHashAndCommit(hash []byte) []byte {
	return append([]byte(prefixSeedHashAndCommit), hash...)
}


func SetSeedHashAndSeed(ldb *LevelDB, hash []byte, seed []byte) error {
	return ldb.Set(keySeedHashAndSeed(hash), seed)
}

func GetSeedBySeedHash(ldb *LevelDB, hash []byte) ([]byte,bool) {
	return ldb.Get(keySeedHashAndSeed(hash))
}

func SetSeedHashAndTx(ldb *LevelDB, hash []byte, tx []byte) error {
	return ldb.Set(keySeedHashAndTx(hash), tx)
}

func GetTxBySeedHash(ldb *LevelDB, hash []byte) ([]byte,bool) {
	return ldb.Get(keySeedHashAndTx(hash))
}

func SetSeedHashAndCommit(ldb *LevelDB, hash []byte, commit []byte) error {
	return ldb.Set(keySeedHashAndCommit(hash), commit)
}

func GetTxBySeedCommit(ldb *LevelDB, hash []byte) ([]byte,bool) {
	return ldb.Get(keySeedHashAndCommit(hash))
}