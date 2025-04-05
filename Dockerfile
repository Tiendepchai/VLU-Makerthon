FROM golang:1.24.1-bookworm

WORKDIR /go/src/makerthon

COPY go.mod go.sum ./
RUN go mod download && go mod verify

COPY . .
RUN curl -sLO https://github.com/tailwindlabs/tailwindcss/releases/latest/download/tailwindcss-linux-x64
RUN chmod +x tailwindcss-linux-x64

CMD [ "go", "tool", "air" ]
