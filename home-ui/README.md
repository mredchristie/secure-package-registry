# Home UI

## Update .env

Since we are coordinating multiple urls, place the `home-ui` url here

```env
PUBLIC_HOME_BASE_URL=<BASE URL> # http://localhost:5173 usually
```

Set your better auth secret key in `.env`:

```env
BETTER_AUTH_SECRET=<Secret Key>
```

run `openssl rand -base64 32` to generate a random secret key