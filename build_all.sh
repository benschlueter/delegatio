#!/bin/bash

docker build -f container/challenges/testing/Dockerfile.archlinux -t ghcr.io/benschlueter/delegatio/archimage:0.1 .
docker push ghcr.io/benschlueter/delegatio/archimage:0.1


docker build -f Dockerfile.ssh -t ghcr.io/benschlueter/delegatio/ssh:0.1 .
docker push ghcr.io/benschlueter/delegatio/ssh:0.1

docker build -f Dockerfile.grader -t ghcr.io/benschlueter/delegatio/grader:0.1 .
docker push ghcr.io/benschlueter/delegatio/grader:0.1
