### Test web server

Use this to do some simple HTTP calls.  A service is provided in [docker-compose](../docker-compose.yml) called `us-web` that will build the `nginx` image and load a small test harness index.html. It needs an API host, but it doesn't explicitly require one - it just gloms on to whatever you point it at. Sample usage would be something like:

```bash
docker-compose [--build [--remove-orphans]] [-d] us-web
```

Since any API host exposes the same interface, it doesn't matter which database you use. The current defaults assume a a host named `servicetester` on port `3000`. These values should really come from the command line.

All the resources for the test harness, except jQuery, are in the `index.html` file, so it can serve as a reference for ajax calls and wonky redirect handling.

Could possibly add more interfaces to the server - all the services are exposed in the config already, just not trying to make a project out of a table editor/index lister. Would be neat though. Also, curiously not secure.
