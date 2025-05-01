FROM golang:1.24.1-bookworm

WORKDIR /go/src/makerthon

COPY go.mod go.sum ./
RUN go mod download && go mod verify

# Thêm các dependencies OAuth2
RUN go get golang.org/x/oauth2
RUN go get golang.org/x/oauth2/github
RUN go get golang.org/x/oauth2/google
RUN go mod tidy

COPY . .
RUN curl -sLO https://github.com/tailwindlabs/tailwindcss/releases/latest/download/tailwindcss-linux-x64
RUN chmod +x tailwindcss-linux-x64

CMD [ "go", "tool", "air" ]
