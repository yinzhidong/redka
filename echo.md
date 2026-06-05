

```sh

export GOPROXY='https://goproxy.cn ,direct'

make setup build           


cd build && ./redka -h 0.0.0.0 -p 6379 data.db


redis-cli -h localhost -p 6379


https://redis.io/docs/latest/commands/hello/

localhost:6379> hello 3
1# "server" => "redis"
2# "version" => "7.3.0"
3# "proto" => (integer) 3
4# "id" => (integer) 0
5# "mode" => "standalone"
6# "role" => "master"
7# "modules" => (empty array)



https://github.com/yinzhidong/redka/blob/main/backend/bin/redka


https://api.gitproxy.dev/github.com/yinzhidong/redka/raw/refs/heads/main/backend/bin/redka


curl -o ./redka https://api.gitproxy.dev/github.com/yinzhidong/redka/raw/refs/heads/main/backend/bin/redka


```



## 学习

```sh

git remote rm origin

git remote add origin https://github.com/yinzhidong/redka.git


https://github.com/orgs/tursodatabase/repositories

```