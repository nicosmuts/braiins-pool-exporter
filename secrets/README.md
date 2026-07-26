# Local secrets

This directory is for local development secrets used by Docker Compose.

To use a token file without placing the token in `.env`:

1. Create `secrets/braiins_pool_token`.
2. Put only the Braiins Pool API token in that file.
3. Set this value in `.env`:

   ```env
   BRAIINS_POOL_TOKEN_FILE=/run/secrets/braiins_pool_token
   ```

Files in this directory are ignored by Git except for this README and
`.gitkeep`. Do not commit live tokens, raw API responses, payout addresses,
transaction identifiers, or other private operational data.
