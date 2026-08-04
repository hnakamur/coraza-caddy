ftw-all:
  rm -f build/*.log
  go run mage.go ftw 2>&1 | tee ftw.log

ftw case:
  rm -f build/*.log
  FTW_INCLUDE={{case}} FTW_DEBUG=1 go run mage.go ftw 2>&1 | tee {{case}}.log

build-docker-compose:
  (cd ftw; docker compose build --no-cache --pull --build-arg CRS_VERSION=v4.25.0)

build-caddy:
  go run mage.go buildCaddyLinux

build-ftw:
  go build -C ../../coreruleset/go-ftw -trimpath -tags netgo,osusergo -o "${PWD}/ftw/go-ftw"
