# Secure Package Registry (SPR)

A package registry enforcing supply chain security across multiple package ecosystems (NPM, Go Modules).

## MVP Goals

- Fully bootstrap SPR with our own security, of the highest relevant level. All dependencies used both in NPM and Go Modules should be included in the registry. This includes behaviour analysis and reproducible building of packages from source.
- Have a clean, intuitive UI to interact with the system, with proper authentication for organisations and developers to pull their allowed packages.
- Have a well polished marketing website to promote the product and its features to both technical and non-technical interested parties such as investors and managers.

## How to run

Due to the number of individual services, it is not recommended and very complex to run the full application without podman.

install `podman` [https://podman.io](https://podman.io)

1. `git clone` the repository
2. `podman compose build; podman compose up -d`
3. Access `localhost:7001` to be directed to the landing page.

If you want to access the other services, see `rfcs/2026-03-26-port-allocation-scheme.md` for the port numbers.

Edit `.env` to change information and secrets around, see `.env.example` for what `.env` should look like.

To run individual any individual services, run `podman compose run <name>`. The current services we have are:

```text
home-ui         # Landing/Marketing Website
dashboard-ui    # Functionality/Developer-side Website
spr             # Core Services (Reproducible Builds, Behaviour Analysis, Workflows etc.)
gitea           # Gitea (https://github.com/go-gitea/gitea), this is what we use as the package registry (see rfcs/2026-02-02-OSS-registry-options.md)
gitea_db        # Gitea database
core_db         # Core, PostgreSQL, database
valkey          # Valkey (Key-Value) Database
minio           # Minio (Object-storage) Database
minio-init      # Minio initialisation
rabbitmq        # RabbitMQ service (Inter-Service communication)
```
