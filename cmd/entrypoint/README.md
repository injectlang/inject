# entrypoint

Used with the config-container method of deployment.

This program is used in your code container to act as the entrypoint.  When the container starts, 
`entrypoint` will contact `http://localhost:5309/sh` (or whatever URL you specify in `CONFIG_CONTAINER_URL`).

`injectord` will respond with key/value pairs that are compatible with bash, and could be used like
`eval $(curl http://localhost:5309/sh)`.  `entrypoint` will take those key/value pairs and inject them into
a new environment (as in env vars), then fork/exec your daemon.