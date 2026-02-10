# API – Plataforma de Venda de Ingressos

API GraphQL em Go (gqlgen) com SQLite. Especificação funcional e técnica completa: **[docs/SPEC.md](../docs/SPEC.md)**.

## Requisitos

- Go 1.21+

## Executar

```bash
cd api
go build ./cmd/api
./api
# ou
go run ./cmd/api
```

Servidor em `http://localhost:8080`. Endpoint GraphQL: `POST http://localhost:8080/graphql`.

## Variáveis de ambiente

| Variável      | Descrição                    | Padrão              |
|---------------|------------------------------|---------------------|
| `PORT`        | Porta HTTP                   | `8080`              |
| `DB_PATH`     | Caminho do arquivo SQLite    | `./data/afterzin.db`|
| `JWT_SECRET`  | Chave para assinatura JWT    | (dev default)       |
| `PLAYGROUND`  | Habilitar GraphQL Playground | `false`             |
| `CORS_ORIGINS`| Origens CORS (uma por linha) | `http://localhost:5173` |

## Principais operações

- **Auth:** `register`, `login`
- **Catálogo:** `events`, `event`
- **Usuário:** `me`, `myTickets`, `myTicket`
- **Produtor:** `createEvent`, `createEventDate`, `createLot`, `createTicketType`, `publishEvent`
- **Checkout:** `checkoutPreview`, `checkoutPay`
- **Validação:** `validateTicket`

## Seeds

Para popular o banco com dados iniciais (usuários, eventos, lotes, ingressos):

```bash
go run ./cmd/seed
```

Senha dos usuários de seed: `123456`. Ver `internal/db/seeds/README.md` para detalhes.

## Estrutura

- `cmd/api` – servidor HTTP / GraphQL
- `cmd/seed` – comando para rodar seeds
- `internal/config` – configuração
- `internal/db` – SQLite e migrations
- `internal/graphql` – schema, resolvers e handlers
- `internal/auth` – JWT e bcrypt
- `internal/middleware` – CORS e auth
- `internal/repository` – acesso a dados
