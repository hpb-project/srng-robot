package db

import (
	"github.com/astaxie/beego/logs"
	"sync"
	"time"

	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/errors"
	"github.com/syndtr/goleveldb/leveldb/filter"
	"github.com/syndtr/goleveldb/leveldb/opt"
	"github.com/syndtr/goleveldb/leveldb/util"
)

// KeyValueWriter wraps the Put method of a backing data store.
type KeyValueWriter interface {
	// Put inserts the given value into the key-value data store.
	Set(key interface{}, value []byte) error

	// Delete removes the key from the key-value data store.
	Delete(key interface{}) error
}

// Batch is a write-only database that commits changes to its host database
// when Write is called. A batch cannot be used concurrently.
type Batch interface {
	KeyValueWriter

	// ValueSize retrieves the amount of data queued up for writing.
	ValueSize() int

	// Write flushes any accumulated data to disk.
	Write() error

	// Reset resets the batch for reuse.
	Reset()
}

const (
	// degradationWarnInterval specifies how often warning should be printed if the
	// leveldb database cannot keep up with requested writes.
	degradationWarnInterval = time.Minute

	// minCache is the minimum amount of memory in megabytes to allocate to leveldb
	// read and write caching, split half and half.
	minCache = 16

	// minHandles is the minimum number of files handles to allocate to the open
	// database files.
	minHandles = 16

	// metricsGatheringInterval specifies the interval to retrieve leveldb database
	// compaction, io and pause stats to report to the user.
	metricsGatheringInterval = 3 * time.Second
)

// Database is a persistent key-value store. Apart from basic data storage
// functionality it also supports batch writes and iterating over the keyspace in
// binary-alphabetical order.
type LevelDB struct {
	fn string      // filename for reporting
	db *leveldb.DB // LevelDB instance

	quitLock sync.Mutex      // Mutex protecting the quit channel access
	quitChan chan chan error // Quit channel to stop the metrics collection before closing the database
}

func NewLevelDB(path string) *LevelDB {
	file, cache, handles := path, 1000, 1000
	db, err := New(file, cache, handles)
	if err != nil {
		logs.Error(" new level db failed ", " error ", err)
		return nil
	}
	return db
}

// New returns a wrapped LevelDB object. The namespace is the prefix that the
// metrics reporting should use for surfacing internal stats.
func New(file string, cache int, handles int) (*LevelDB, error) {
	return NewCustom(file, func(options *opt.Options) {
		// Ensure we have some minimal caching and file guarantees
		if cache < minCache {
			cache = minCache
		}
		if handles < minHandles {
			handles = minHandles
		}
		// Set default options
		options.OpenFilesCacheCapacity = handles
		options.BlockCacheCapacity = cache / 2 * opt.MiB
		options.WriteBuffer = cache / 4 * opt.MiB // Two of these are used internally
	})
}

// NewCustom returns a wrapped LevelDB object. The namespace is the prefix that the
// metrics reporting should use for surfacing internal stats.
// The customize function allows the caller to modify the leveldb options.
func NewCustom(file string, customize func(options *opt.Options)) (*LevelDB, error) {
	options := configureOptions(customize)
	usedCache := options.GetBlockCacheCapacity() + options.GetWriteBuffer()*2
	logCtx := []interface{}{"cache", usedCache, "handles", options.GetOpenFilesCacheCapacity()}
	if options.ReadOnly {
		logCtx = append(logCtx, "readonly", "true")
	}
	logs.Info("Allocated cache and file handles ", "logCtx ", logCtx)

	// Open the db and recover any potential corruptions
	db, err := leveldb.OpenFile(file, options)
	if _, corrupted := err.(*errors.ErrCorrupted); corrupted {
		db, err = leveldb.RecoverFile(file, nil)
	}
	if err != nil {
		return nil, err
	}
	// Assemble the wrapper with all the registered metrics
	ldb := &LevelDB{
		fn:       file,
		db:       db,
		quitChan: make(chan chan error),
	}

	return ldb, nil
}

// configureOptions sets some default options, then runs the provided setter.
func configureOptions(customizeFn func(*opt.Options)) *opt.Options {
	// Set default options
	options := &opt.Options{
		Filter:                 filter.NewBloomFilter(10),
		DisableSeeksCompaction: true,
	}
	// Allow caller to make custom modifications to the options
	if customizeFn != nil {
		customizeFn(options)
	}
	return options
}

// Close stops the metrics collection, flushes any pending data to disk and closes
// all io accesses to the underlying key-value store.
func (db *LevelDB) Close() error {
	db.quitLock.Lock()
	defer db.quitLock.Unlock()

	if db.quitChan != nil {
		errc := make(chan error)
		db.quitChan <- errc
		if err := <-errc; err != nil {
			logs.Error("Metrics collection failed", "err", err)
		}
		db.quitChan = nil
	}
	return db.db.Close()
}

// Has retrieves if a key is present in the key-value store.
func (db *LevelDB) Has(key interface{}) (bool, error) {
	k := key.([]byte)
	return db.db.Has(k, nil)
}

// Get retrieves the given key if it's present in the key-value store.
func (db *LevelDB) Get(key interface{}) ([]byte, bool) {
	k := key.([]byte)
	dat, err := db.db.Get(k, nil)
	if err != nil {
		return nil, false
	}
	return dat, true
}

// Put inserts the given value into the key-value store.
func (db *LevelDB) Set(key interface{}, value []byte) error {
	k := key.([]byte)
	return db.db.Put(k, value, nil)
}

// Delete removes the key from the key-value store.
func (db *LevelDB) Del(key interface{}) error {
	k := key.([]byte)
	return db.db.Delete(k, nil)
}

func (db *LevelDB) Iterator(prefix []byte, iterfunc func(key, value []byte)) {
	iter := db.db.NewIterator(util.BytesPrefix(prefix), nil)
	defer iter.Release()
	for iter.Next() {
		// Remember that the contents of the returned slice should not be modified, and
		// only valid until the next call to Next.
		key := iter.Key()
		value := iter.Value()
		if iterfunc != nil {
			iterfunc(key, value)
		}
	}
}

// NewBatch creates a write-only key-value store that buffers changes to its host
// database until a final write is called.
func (db *LevelDB) NewBatch() Batch {
	return &batch{
		db: db.db,
		b:  new(leveldb.Batch),
	}
}

// batch is a write-only leveldb batch that commits changes to its host database
// when Write is called. A batch cannot be used concurrently.
type batch struct {
	db   *leveldb.DB
	b    *leveldb.Batch
	size int
}

// Put inserts the given value into the batch for later committing.
func (b *batch) Set(key interface{}, value []byte) error {
	k := key.([]byte)
	b.b.Put(k, value)
	b.size += len(k) + len(value)
	return nil
}

// Delete inserts the a key removal into the batch for later committing.
func (b *batch) Delete(key interface{}) error {
	k := key.([]byte)
	b.b.Delete(k)
	b.size += len(k)
	return nil
}

// ValueSize retrieves the amount of data queued up for writing.
func (b *batch) ValueSize() int {
	return b.size
}

// Write flushes any accumulated data to disk.
func (b *batch) Write() error {
	return b.db.Write(b.b, nil)
}

// Reset resets the batch for reuse.
func (b *batch) Reset() {
	b.b.Reset()
	b.size = 0
}

// bytesPrefixRange returns key range that satisfy
// - the given prefix, and
// - the given seek position
func bytesPrefixRange(prefix, start []byte) *util.Range {
	r := util.BytesPrefix(prefix)
	r.Start = append(r.Start, start...)
	return r
}

