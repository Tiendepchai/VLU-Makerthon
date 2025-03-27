FROM golang:1.24.1-bookworm

WORKDIR /go/src/makerthon

COPY go.mod go.sum ./
RUN go mod download && go mod verify

COPY . .
RUN go install github.com/air-verse/air@latest
RUN go install github.com/a-h/templ/cmd/templ@latest
RUN curl -sLO https://github.com/tailwindlabs/tailwindcss/releases/latest/download/tailwindcss-linux-x64
RUN chmod +x tailwindcss-linux-x64

CMD [ "air" ]
