# Seeds

Dados iniciais para popular o banco de dados em desenvolvimento.

## Conteúdo

- **Usuários** (senha de todos: `123456`)
  - `joao@email.com` – usuário comum
  - `maria@email.com` – usuário comum
  - `produtor@email.com` – usuário produtor (cria eventos)

- **Eventos publicados** (5 eventos)
  - Festival de Verão 2025 (festivais, destaque)
  - Show Anitta (shows, destaque)
  - Final Copa do Brasil 2025 (esportes)
  - Baile da Favorita (festas)
  - O Fantasma da Ópera (teatro)

- **Datas, lotes e tipos de ingresso** para cada evento

## Como executar

A partir da raiz do projeto `api`:

```bash
go run ./cmd/seed
```

Ou após build:

```bash
go build -o seed ./cmd/seed
./seed
```

O comando aplica as migrations (se ainda não foram) e em seguida **limpa** as tabelas de usuários, produtores, eventos, datas, lotes, tipos de ingresso, pedidos e tickets, e **insere** os dados de seed. Pode ser executado várias vezes (o banco volta ao estado inicial de seed).

## Variáveis de ambiente

- `DB_PATH` – caminho do arquivo SQLite (padrão: `./data/afterzin.db`)
