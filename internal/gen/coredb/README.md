# CoreDB

## How to update the database schema

To update the database schema, you need to make a new migration file, which can be found in `infra/migrations`. You must
make a "up" and "down" file, to indicate creating and removing the new changes. The "up" file should contain the new
schema changes, and the "down" file should contain the reverse of those changes.

The naming convention for the migrations files is `<migration number>_<migration_name>.<up/down>.sql>` the migration
number should be incremented for each new migration, and should be 6 digits long ie: `000001`, `000002`, etc. The
migration name should be a short description of the changes being made, and should be in snake case.

When you have created the new migration files, update the sqlc `schema.sql` file to reflect the new schema changes. This
file is used by sqlc to generate the new database queries, and should be kept up to date with the latest schema changes.
(Just pasting the contents of the "up" file into the end of `schema.sql` should be sufficient).

Run `sqlc generate` to generate the new database models and such.
