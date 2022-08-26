## srng-robot
`srng-robot` is a automation tools for [HRG](https://rc.hpb.io) Commiter to create commit and do reveal.

## build
1. install golang atleast 1.17 version. 
2. build `robot`
```
# git clone https://github.com/hpb-project/srng-robot
# cd srng-robot
# go build ./cmd/robot
```

## deploy
* set hpb account private key in `conf/app.conf`
* prepare atleast 10 HPB and 30 HRG in hpb account. 
* exec `./start.sh` 