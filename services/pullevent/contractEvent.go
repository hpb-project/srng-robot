package pullevent

import (
	"encoding/hex"
	"github.com/astaxie/beego/logs"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/hpb-project/srng-robot/contracts"
	"github.com/hpb-project/srng-robot/db"
	"strings"
)

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
			logs.Info("got subscribe event","commithash", hex.EncodeToString(sub.Hash[:]), "consumer is", sub.Consumer)
			if _, exist := db.GetSeedBySeedHash(pe.ldb, sub.Hash[:]); exist {
				// check unreveal
				if db.HasUnRevealSeed(pe.ldb, sub.Hash[:]) {
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
			logs.Info("got new commit event", "commit hash", hex.EncodeToString(commit.Hash[:]), "block", commit.Block)
			// first check commit exist and unreveal.

		case EventRevealSeed:
			reveal, err := filter.ParseRevealSeed(vLog)
			if err != nil {
				logs.Error("parse reveal event failed", "err", err)
				return err
			}
			if reveal.Commiter != pe.user {
				return nil
			}
			logs.Info("got revealed event", "commit", hex.EncodeToString(reveal.Hash[:]))
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
