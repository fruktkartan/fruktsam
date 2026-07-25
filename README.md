
Needs `DATABASE_URL` environment variable, or in `.env`.

Note that if you run this without a `reversecache` file (with some
data) in the destination directory (default `./dist`, see flag `-d`),
fruktsam will do an OSM nominatim-reverse web API call for each and
every tree in the database, which might become rate limited. Also, all
needed image files that do not already exist in the destination's
`images` directory will need to be downloaded.

The following can be used to find out the production database URL (once you've managed
`login`, or `auth:login`?)

```
heroku pg:credentials:url --app fruktkartan-fullstack
```

(`--app fruktkartan-fullstack-dev` for the development database)

TODO: even when using the development database, generated links, API calls etc
still point at fruktkartan.se
