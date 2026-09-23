![Baton Logo](./docs/images/baton-logo.png)

# `baton-demo` [![Go Reference](https://pkg.go.dev/badge/github.com/conductorone/baton-demo.svg)](https://pkg.go.dev/github.com/conductorone/baton-demo) ![main ci](https://github.com/conductorone/baton-demo/actions/workflows/main.yaml/badge.svg)

`baton-demo` is an example connector built using the [Baton SDK](https://github.com/conductorone/baton-sdk). It uses hardcoded data to provide a simple example of how to build your own connector with Baton.

Check out [Baton](https://github.com/conductorone/baton) to learn more about the project in general.

# Getting Started
## brew
```
brew install conductorone/baton/baton conductorone/baton/baton-demo

baton-demo
baton resources
```

## docker
```
docker run --rm -v $(pwd):/out ghcr.io/conductorone/baton-demo:latest -f "/out/sync.c1z"
docker run --rm -v $(pwd):/out ghcr.io/conductorone/baton:latest -f "/out/sync.c1z" resources
```

## source
```
go install github.com/conductorone/baton/cmd/baton@main
go install github.com/conductorone/baton-demo/cmd/baton-demo@main

baton-demo 
baton resources
```

## Browse the live demo application

Run the built-in web viewer against the same database used by your connector:

```sh
go run ./cmd/baton-demo browse --db-file-name ./baton-demo.db
```

Open `http://127.0.0.1:8080`. Use `--port` to choose another port, or `--port 0` to
choose an available port. `BATON_DB_FILE_NAME` also sets the database path.

The viewer shows users, groups, roles, projects, scoped roles, non-human identities,
agents and credential metadata. Select a record to follow its direct relationships.
It refreshes every two seconds while visible, so provisioning performed by another
process appears without running a sync. The observed-changes panel tracks changes
seen during the current browser session; it is not a durable audit log. Short-lived
changes between refreshes may not be observed.

The database must already exist. Browsing uses a read-only connection, does not seed
or migrate it, and never exposes the passwords table. Missing tables in older demo
databases are identified in the UI. The viewer supports up to 10,000 rows per table
and reports an error instead of displaying a truncated inventory above that limit.
It binds only to localhost. Stop with Ctrl-C.

The proposed generic SDK connector browser is specified separately in
[the SDK browser spec](docs/sdk-connector-browser-spec.md).

## Seed demo users from a CSV

Pass `--users-csv` with a file that has an `email` column and either `display_name` or `first_name` and `last_name`. `employment_status` is optional; when provided, only `active` users are enabled. Every CSV column is emitted as a user profile attribute, and seeded users keep baton-demo's generated group and role assignments. Duplicate display names are distinguished with the user's email; duplicate emails are rejected. A ready-to-run example is included at `examples/users.csv`.

```sh
baton-demo --init-db --db-file-name /tmp/baton-demo-seeded.db \
  --users-csv ./examples/users.csv
```

# 

# Contributing, Support and Issues

We started Baton because we were tired of taking screenshots and manually building spreadsheets.  We welcome contributions, and ideas, no matter how small -- our goal is to make identity and permissions sprawl less painful for everyone.  If you have questions, problems, or ideas: Please open a Github Issue!

See [CONTRIBUTING.md](https://github.com/ConductorOne/baton/blob/main/CONTRIBUTING.md) for more details.

# `baton-demo` Command Line Usage

```
baton-demo

Usage:
  baton-demo [flags]
  baton-demo [command]

Available Commands:
  capabilities       Get connector capabilities
  completion         Generate the autocompletion script for the specified shell
  help               Help about any command

Flags:
      --client-id string       The client ID used to authenticate with ConductorOne ($BATON_CLIENT_ID)
      --client-secret string   The client secret used to authenticate with ConductorOne ($BATON_CLIENT_SECRET)
      --db-file string         A file to which the database will be written ($BATON_DB_FILE)
                               example: /path/to/dbfile.db ($BATON_DB_FILE)
  -f, --file string            The path to the c1z file to sync with ($BATON_FILE) (default "sync.c1z")
  -h, --help                   help for baton-demo
      --init-db                Whether to initialize the database ($BATON_INIT_DB)
                               example: true ($BATON_INIT_DB)
      --log-format string      The output format for logs: json, console ($BATON_LOG_FORMAT) (default "json")
      --log-level string       The log level: debug, info, warn, error ($BATON_LOG_LEVEL) (default "info")
  -p, --provisioning           This must be set in order for provisioning actions to be enabled ($BATON_PROVISIONING)
      --skip-full-sync         This must be set to skip a full sync ($BATON_SKIP_FULL_SYNC)
      --ticketing              This must be set to enable ticketing support ($BATON_TICKETING)
  -v, --version                version for baton-demo

Use "baton-demo [command] --help" for more information about a command.
```
