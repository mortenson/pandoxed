# Build webserver
FROM golang:1.23-bookworm as base-golang

WORKDIR /build

COPY main.go .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o pandoxed ./main.go

# Create container in pandoc compatible environment
FROM pandoc/latex:3.5-ubuntu

COPY --from=base-golang /build/pandoxed /pandoxed
COPY flag.txt /flags/unguessable-350616a813ba44a5abed3adc1c61de08/flag.txt
RUN chmod 755 /flags/unguessable-350616a813ba44a5abed3adc1c61de08

RUN useradd -ms /bin/bash pandoxeduser

USER pandoxeduser

WORKDIR /home/pandoxeduser

ENTRYPOINT ["/pandoxed"]
