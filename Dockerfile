FROM fedora:37

RUN dnf update -y \
    && dnf install -y \
        git \
        golang \
        unrar \
    && dnf clean all \
    && :

WORKDIR /autoplex
COPY go.mod go.sum /autoplex/
RUN go mod download
COPY . /autoplex
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o autoplex

ENTRYPOINT ["./autoplex"]
