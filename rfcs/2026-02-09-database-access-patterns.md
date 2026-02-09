+++
title = "Database Access Patterns"
author = "cheongyx@cardiff.ac.uk"
date = "2026-02-09"
+++

## Approaches Evaluated

### Kysely in Sveltekit direct database access

Reads can happen from anywhere since for package metadata is immutable and we don't have to worry about consistency. The
question is simply what is more maintainable and reusable.

Since we are managing migrations from Golang, we cannot use a traditional ORM. We found [Kysely](https://kysely.dev/) to
be a good middle ground similar to `sqlc` in terms of allowing type-safe access to the database with signatures mapping
nearly identically to SQL.

The advantages of having direct data access from Sveltekit is it allows for a full-stack environment where everything is
statically typed. Using Form Actions, it allows us to have type-safe bindings between the front and back end similar to
React Server Components.

However, Form Actions makes caching on the HTTP level impossible and creates complexities when the data needs to be
accessed elsewhere. For example, if an internal service requires searching for packages or accessing package versions,
we will need duplicate code.

If we were to do an API endpoint rather than Form Actions integrated with Sveltekit, while still using Typescript, we
lose the advantages of type-safe bindings while also tying ourselves into an inefficient language.

### An API endpoint in core-svc

That is already where we are handling our database and migrations. It makes sense that we reuse the existing
infrastructure and generated `sqlc` type-safe bindings for the API.

The disadvantage of having our types in Go is that we will have to somehow define the same types in Typescript for type
safety.

One solution to this problem is generating OpenAPI specifications from the Go structs using `kin-openapi`. There are
then multiple tools available for converting specifications to type definitions in any language.

## Decision

I ultimately decided on writing the API in Go for the consistency. We can have all database operations in the same
service, minimizing the chance of race conditions and need to repeated middleware to handle security and other aspects.
