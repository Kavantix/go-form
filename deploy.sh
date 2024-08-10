#! /bin/bash
#
docker build -f prod.Dockerfile . --platform="linux/amd64" -t europe-west3-docker.pkg.dev/eighth-gamma-414620/docker/go-form:v0.0.1 -t go-form:v0.0.1

docker push europe-west3-docker.pkg.dev/eighth-gamma-414620/docker/go-form:v0.0.1

gcloud run deploy go-form \
	--image=europe-west3-docker.pkg.dev/eighth-gamma-414620/docker/go-form:v0.0.1 \
	--region=europe-west3 \
	--project=eighth-gamma-414620
