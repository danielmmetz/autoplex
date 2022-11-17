FROM ubuntu:kinetic

RUN apt-get update \
	&& apt-get install -y --no-install-recommends \
        ca-certificates \
        git \
        golang \
        unrar \
	&& rm -rf /var/lib/apt/lists/* \
    && :

WORKDIR /autoplex
COPY go.mod go.sum /autoplex/
RUN go mod download
COPY . /autoplex
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o autoplex

ENTRYPOINT ["./autoplex"]
