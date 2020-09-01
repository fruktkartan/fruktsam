
Builds automatically upon push to `master`.

Every night at 00:01 (and on [manual workflow run](https://github.com/fruktkartan/fruktsam/actions?query=workflow%3A%22Build+%28and+deploy%29%22)),
the build is also deployed to https://fruktkartan.se/historik/. The cache of
reverse-geocoded addresses is also commited back to the repo.

Needs FRUKTKARTAN_DATABASEURI environment variable, or in `.env`.
