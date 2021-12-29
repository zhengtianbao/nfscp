FROM alpine

WORKDIR /
COPY _output/bin/amd64/nfscp /

ENTRYPOINT ["/nfscp"]