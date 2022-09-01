#!/bin/bash
docker pull ghcr.io/middleware-labs/agent-host-go:dev
docker run -d \
--pid host \
--restart always \
-v /var/run/docker.sock:/var/run/docker.sock \
-e MW_API_KEY=$MW_API_KEY \
-e TARGET=$TARGET \
--network=host ghcr.io/middleware-labs/agent-host-go:dev api-server start
