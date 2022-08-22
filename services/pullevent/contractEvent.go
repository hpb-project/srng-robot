package pullevent

import (
	"github.com/astaxie/beego/logs"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/hpb-project/srng-robot/contracts"
	"github.com/hpb-project/srng-robot/db"
	"strings"
	"time"
)
func friendlyCommit(commit contracts.Commit) string {
	s := fmt.Sprintf("author:%s, commit:%s, commitblock:%s, seed:%s, revealed:%v, consumer:%s, subsender:%s, subblock:%s",
	commit.Author.String(), hex.EncodeToString(commit.Commit[:]), commit.Block.Text(10), hex.EncodeToString(commit.Seed[:]),
	commit.Revealed, commit.Consumer.String(), commit.Subsender.String(), commit.SubBlock.Text(10))
	p := append([]byte(""), s...)
	return string(p)
}

func OracleContractHandler(vLog types.Log, pe *PullEvent, addr common.Address, history bool) error {
	logs.Info("handler oracle contract logs")
	filter, err := contracts.NewOracleFilterer(addr, pe.client)
	if err != nil {
		logs.Error("NewOracleFilter failed", "err", err)
		return err
	}
	{
		method := vLog.Topics[0]
		switch strings.ToLower(method.String()) {
		case EventSubscribe:
			sub, err := filter.ParseSubscribe(vLog)
			if err != nil {
				logs.Error("parse subscribe event failed", "err", err)
				return err
			}
			if sub.Commiter != pe.user {
				return nil
			}
			// go to reveal.
			logs.Info("got subscribe event","commit info", friendlyCommit(sub))
			if _, exist := db.GetSeedBySeedHash(pe.ldb, sub.Hash[:]); exist {
				// check unreveal
				if db.HashUnRevealSeed(pe.ldb, sub.Hash[:]) {
					pe.work.Reveal(sub.Hash[:])
				}
			}
			pe.work.Reveal(sub.Hash[:])


		case EventCommitHash:
			commit, err := filter.ParseCommitHash(vLog)
			if err != nil {
				logs.Error("parse commit event failed", "err", err)
				return err
			}
			if commit.Sender != pe.user {
				return nil
			}
			logs.Info("got commit event", "commit info", friendlyCommit(commit))
			// first check commit exist and unreveal.
			if _, exist := db.GetSeedBySeedHash(pe.ldb, commit.Hash[:]); exist {
				// check unreveal
				if db.HashUnRevealSeed(pe.ldb, commit.Hash[:]) {
					if history {
						pe.work.Reveal(commit.Hash[:])
					} else {
						go func(hash []byte) {
							time.Sleep(time.Second*10)
							pe.work.Reveal(hash)
						}(commit.Hash[:])
					}
				}
			}

		case EventRevealSeed:
			reveal, err := filter.ParseRevealSeed(vLog)
			if err != nil {
				logs.Error("parse reveal event failed", "err", err)
				return err
			}
			if reveal.Commiter != pe.user {
				return nil
			}
			logs.Info("got revealed event", "commit", friendlyCommit(reveal))
			// set commit reveal finished.
			db.DelUnRevealSeed(pe.ldb, reveal.Hash[:])
			db.SetSeedHashAndSeed(pe.ldb, reveal.Hash[:], reveal.Seed[:])


		case EventUnSubscribe, EventRandomConsumed:
			// ignore event.
		default:
			// do nothing.
		}
	}
	return nil
}
