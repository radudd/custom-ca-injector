FROM golang:alpine as build

# set labels for metadata
LABEL maintainer="Radu Domnu<radu.domnu@gmail.com>" \
  name="custom-ca-injector" \
  description="A Kubernetes mutating webhook server that implements custom ca injection" 

# set environment variables
ENV GO111MODULE=on \
  CGO_ENABLED=0

RUN apk add git make openssl

# Test and build 
WORKDIR /build
COPY . /build
RUN make test
RUN make build

FROM scratch

VOLUME /ssl
WORKDIR /app

COPY --from=build /build/custom-ca-injector .

CMD ["/app/custom-ca-injector"]

USER 1001