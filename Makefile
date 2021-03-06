.PHONY: clean

IMAGE_URL ?= 248174752766.dkr.ecr.us-west-1.amazonaws.com/autopfs

restart: .push
	ssh admin@mapbot.cernu.us sudo systemctl restart autopfs

push: .push
.push: .docker
	@ set -e; \
	docker push ${IMAGE_URL} && \
	touch .push

.docker: autopfs Dockerfile
	docker build --pull -t autopfs .
	docker tag autopfs ${IMAGE_URL}
	touch .docker

autopfs: ${shell find -name \*.go -o -name \*.js -o -name \*.css} go.mod
	go fmt github.com/pdbogen/autopfs/...
	go generate ./...
	go build -o autopfs github.com/pdbogen/autopfs/server

clean:
	rm -f .push .docker autopfs

tail:
	ssh admin@mapbot.cernu.us journalctl -u autopfs -f
nginx_tail:
	ssh admin@mapbot.cernu.us journalctl -u nginx -f
