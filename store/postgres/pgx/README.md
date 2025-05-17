# Postgres pgx plugin

This module implements a Postgres implementation of the micro store interface. 
It uses modern https://github.com/jackc/pgx driver to access Postgres.

## Implementation notes

### Concepts
Every database has they own connection pool. Due to the way connections are handled this means that all micro "databases" and "tables" can be stored under a single or several Postgres database as specified in the connection string (https://www.postgresql.org/docs/8.1/ddl-schemas.html). The mapping of micro to Postgres concepts is:
- micro database => Postgres schema
- micro table => Postgres table

### Expiry
Expiry is managed by an expiry column in the table. A record's expiry is specified in the column and when a record is read the expiry field is first checked, only returning the record if it's still valid otherwise it's deleted. A maintenance loop also periodically runs to delete any rows that have expired. 
