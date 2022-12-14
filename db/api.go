package db

const (
	prefixSeedHashAndSeed   = "kss"
	prefixSeedHashAndTx     = "kst"
	prefixSeedHashAndCommit = "ksm"
	prefixUnrevealedSeed    = "kunreveal"
	prefixRevealedSeed      = "krevealed"
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

func keySeedHashUnReveal(hash []byte) []byte {
	return append([]byte(prefixUnrevealedSeed), hash...)
}

func keySeedHashRevealed(hash []byte) []byte {
	return append([]byte(prefixRevealedSeed), hash...)
}

func SetSeedHashAndSeed(ldb *LevelDB, hash []byte, seed []byte) error {
	return ldb.Set(keySeedHashAndSeed(hash), seed)
}

func GetSeedBySeedHash(ldb *LevelDB, hash []byte) ([]byte, bool) {
	return ldb.Get(keySeedHashAndSeed(hash))
}

func SetSeedHashAndTx(ldb *LevelDB, hash []byte, tx []byte) error {
	return ldb.Set(keySeedHashAndTx(hash), tx)
}

func GetTxBySeedHash(ldb *LevelDB, hash []byte) ([]byte, bool) {
	return ldb.Get(keySeedHashAndTx(hash))
}

func SetSeedHashAndCommit(ldb *LevelDB, hash []byte, commit []byte) error {
	return ldb.Set(keySeedHashAndCommit(hash), commit)
}

func GetTxBySeedCommit(ldb *LevelDB, hash []byte) ([]byte, bool) {
	return ldb.Get(keySeedHashAndCommit(hash))
}

func SetRevealedSeed(ldb *LevelDB, hash []byte) error {
	return ldb.Set(keySeedHashRevealed(hash), hash)
}

func HasRevealedSeed(ldb *LevelDB, hash []byte) bool {
	_, exist := ldb.Get(keySeedHashRevealed(hash))
	return exist
}

func SetUnRevealSeed(ldb *LevelDB, hash []byte) error {
	return ldb.Set(keySeedHashUnReveal(hash), hash)
}

func HasUnRevealSeed(ldb *LevelDB, hash []byte) bool {
	find, _ := ldb.Has(keySeedHashUnReveal(hash))
	return find
}

func DelUnRevealSeed(ldb *LevelDB, hash []byte) {
	ldb.Del(keySeedHashUnReveal(hash))
}

func GetAllUnReveald(ldb *LevelDB) [][]byte {
	seedhash := make([][]byte, 0, 1000)
	ldb.Iterator([]byte(prefixUnrevealedSeed), func(k, v []byte) {
		p := make([]byte, len(v))
		copy(p[:],v[:])
		seedhash = append(seedhash, p)
	})
	return seedhash
}
