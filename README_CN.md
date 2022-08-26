## srng-robot
`srng-robot` 是为 [HRG](https://rc.hpb.io) 的提交者提供的自动化提交和验证工具.

## 编译
1. 安装 golang 1.17+ 
2. 编译 `robot`
```
# git clone https://github.com/hpb-project/srng-robot
# cd srng-robot
# go build ./cmd/robot
```

## 部署
* 在 `conf/app.conf` 配置文件中填入HPB账号的私钥
* 确保使用的账号至少存有10个HPB, 30 个HRG.
* 执行 start.sh 脚本运行程序