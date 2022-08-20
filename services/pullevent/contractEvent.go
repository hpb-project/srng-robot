package pullevent

import (
	"github.com/astaxie/beego/logs"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/hpb-project/srng-robot/contracts"
	"strings"
)

func OracleContractHandler(vLog types.Log, pe *PullEvent, addr common.Address) error {
	logs.Info("handler oracle contract logs")
	filter, err := contracts.NewOracleFilterer(addr, pe.client)
	if err != nil {
		logs.Error("NewOracleFilter failed", "err", err)
		return err
	}
	{
		method := vLog.Topics[0]
		logs.Info("got event ", vLog)
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
			logs.Info("commit has been subscribed", "commit", sub.Hash)

		case EventUnSubscribe:
			// ignore all unsubscribe event.

			//unsub, err := filter.ParseUnSubscribe(vLog)
			//if err != nil {
			//	logs.Error("parse unsubscribe event failed", "err", err)
			//	return err
			//}

		case EventCommitHash:
			commit, err := filter.ParseCommitHash(vLog)
			if err != nil {
				logs.Error("parse commit event failed", "err", err)
				return err
			}
			if commit.Sender != pe.user {
				return nil
			}
			// todo: add commit to monitor list, reveal after some block
			logs.Info("got event commit", commit)
		case EventRevealSeed:
			reveal, err := filter.ParseRevealSeed(vLog)
			if err != nil {
				logs.Error("parse reveal event failed", "err", err)
				return err
			}
			if reveal.Commiter != pe.user {
				return nil
			}
			// todo: set commit reveal finished.
			logs.Info("got event reveal", reveal)
		case EventRandomConsumed:
			// ignore all consumed event.
			//consume, err := filter.ParseRandomConsumed(vLog)
			//if err != nil {
			//	logs.Error("parse consume event failed", "err", err)
			//	return err
			//}
			//logs.Info("got event consume", consume)
		}
	}
	return nil
}
