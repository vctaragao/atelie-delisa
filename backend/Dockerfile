# Imagem de produção: compila o binário e o copia para uma imagem mínima.
FROM golang:1.26-alpine AS build

RUN apk add --no-cache git
WORKDIR /src
COPY . .

# go mod tidy resolve as dependências na hora do build; o projeto é
# distribuído sem go.sum para não exigir Go instalado na sua máquina.
RUN go mod tidy

# CGO_ENABLED=0 produz um binário estático — modernc.org/sqlite é Go puro,
# então não há dependência da libsqlite3 do sistema.
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/server ./cmd/server

FROM alpine:3.21

RUN adduser -D -u 10001 atelie
COPY --from=build /out/server /usr/local/bin/server

# O volume de dados é montado aqui; o dono precisa ser o usuário do processo.
RUN mkdir -p /data && chown atelie:atelie /data
USER atelie

EXPOSE 8080
ENV DB_PATH=/data/atelie.db ADDR=:8080

ENTRYPOINT ["/usr/local/bin/server"]
