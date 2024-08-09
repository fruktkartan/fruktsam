
Build on push: [![](https://github.com/fruktkartan/fruktsam/actions/workflows/build-and-deploy.yml/badge.svg?branch=master&event=push)](https://github.com/fruktkartan/fruktsam/actions/workflows/build-and-deploy.yml) \
Manual deploy: [![](https://github.com/fruktkartan/fruktsam/actions/workflows/build-and-deploy.yml/badge.svg?branch=master&event=workflow_dispatch)](https://github.com/fruktkartan/fruktsam/actions/workflows/build-and-deploy.yml) \
Scheduled deploy: [![](https://github.com/fruktkartan/fruktsam/actions/workflows/build-and-deploy.yml/badge.svg?branch=master&event=schedule)](https://github.com/fruktkartan/fruktsam/actions/workflows/build-and-deploy.yml)

The build is deployed every night at 00:01 UTC (and on [manual workflow run](https://github.com/fruktkartan/fruktsam/actions/workflows/build-and-deploy.yml)).
The updated cache of reverse-geocoded addresses is commited back to the repo.

Needs `DATABASE_URL` environment variable, or in `.env`.

The following can be used to find out the production database URL (once you've managed
`login`, or `auth:login`?)

```
heroku pg:credentials:url --app fruktkartan-api
```

(`--app fruktkartan-api-dev` for the development database)

TODO: even when using the development database, generated links, API calls etc
still point at fruktkartan.se, fruktkartan-api.herokuapp.com, etc
