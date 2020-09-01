
![Build on push](https://github.com/fruktkartan/fruktsam/workflows/Build%20(and%20deploy)/badge.svg?branch=master&event=push)
![Manual deploy](https://github.com/fruktkartan/fruktsam/workflows/Build%20(and%20deploy)/badge.svg?branch=master&event=workflow_dispatch)
![Scheduled deploy](https://github.com/fruktkartan/fruktsam/workflows/Build%20(and%20deploy)/badge.svg?branch=master&event=schedule)

Builds automatically upon push to `master`.

Every night at 00:01 (and on [manual workflow run](https://github.com/fruktkartan/fruktsam/actions?query=workflow%3A%22Build+%28and+deploy%29%22)),
the build is also deployed to https://fruktkartan.se/historik/. The cache of
reverse-geocoded addresses is also commited back to the repo.

Needs `FRUKTKARTAN_DATABASEURI` environment variable, or in `.env`.

TODO: could move the databaseuri deploy secret to organization level (same for fruktkartan-api)
