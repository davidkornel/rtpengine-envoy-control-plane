FROM golang:1.13.6-stretch

WORKDIR /rtpengine-envoy-control-plane

# Fetch and preserve module dependencies
ENV GOPROXY=https://proxy.golang.org
COPY go.mod ./
RUN go mod download

# COPY project into WORKDIR
RUN mkdir controlplane
COPY build/controlplane.sh ./
COPY controlplane ./controlplane
COPY Makefile ./

#Install dependencies & install/build binary
RUN go get ./controlplane
RUN go install -i ./controlplane

#CMD exec /bin/bash -c "trap : TERM INT; sleep infinity & wait"

#start server
#IF you want to start the management server by yourself (i.e kubernetes deployment container CMD ARGS)
#Delete line below
#CMD make controlplane

