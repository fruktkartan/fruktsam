
Build on push: ![](https://github.com/fruktkartan/fruktsam/workflows/Build%20(and%20deploy)/badge.svg?branch=master&event=push)  
Manual deploy: ![](https://github.com/fruktkartan/fruktsam/workflows/Build%20(and%20deploy)/badge.svg?branch=master&event=workflow_dispatch)  
Scheduled deploy: ![](https://github.com/fruktkartan/fruktsam/workflows/Build%20(and%20deploy)/badge.svg?branch=master&event=schedule)  

The build is deployed every night at 00:01 UTC (and on [manual workflow run](https://github.com/fruktkartan/fruktsam/actions?query=workflow%3A%22Build+%28and+deploy%29%22)).
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
