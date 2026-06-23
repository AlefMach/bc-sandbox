# bc-sandbox

Aplicacao Buffalo com Postgres, Kafka e Cassandra para desenvolvimento local.

## Requisitos

- Go
- Node.js e Yarn
- Docker e Docker Compose

As versoes esperadas pelo projeto estao em `.tool-versions`.

## Instalacao das ferramentas

Instale o Buffalo CLI e o plugin do Pop:

```console
go install github.com/gobuffalo/cli/cmd/buffalo@latest
go install github.com/gobuffalo/buffalo-pop/v3@latest
```

Garanta que o binario instalado pelo Go esteja no `PATH`:

```console
export PATH="$PATH:$(go env GOPATH)/bin"
```

Confira se o Buffalo esta disponivel:

```console
buffalo version
```

## Configuracao

Crie o arquivo `.env` a partir do exemplo:

```console
cp .env-example .env
```

As credenciais padrao ja apontam para os servicos locais do Docker Compose.

Instale as dependencias do projeto:

```console
make deps
```

## Subindo a infraestrutura

Suba Postgres, Kafka e Cassandra:

```console
make infra-up
```

Para subir tambem as ferramentas opcionais, como Kafka UI:

```console
make infra-up-tools
```

## Banco de dados

Crie os bancos configurados no `database.yml`:

```console
make db-create
```

Rode as migrations:

```console
make db-migrate
```

Opcionalmente, rode a seed:

```console
make db-seed
```

## Rodando a aplicacao

Inicie o servidor de desenvolvimento do Buffalo:

```console
make dev
```

A aplicacao ficara disponivel em:

```text
http://127.0.0.1:3000
```

## Comandos uteis

```console
make help
make format
make test
make infra-ps
make infra-logs
make infra-down
```
