# Secure Package Registry Dashboard

The goal of this component is 3-fold:

1. Allow users to easily search for and view information about packages in different ecosystems and their status in our
   private registry.
2. Provide documentation and instructions on how to use our private registry
3. Allow users to request the verification of packages.

We are pre-MVP, meaning that the focus is on the core features. Customer-facing features like authentication are
de-prioritized. We are only testing internally for now.

## Update .env

Since we are coordinating multiple urls, place the `home-ui` url here

```env
PUBLIC_DASHBOARD_BASE_URL=<BASE URL> # http://localhost:5174 usually
```
