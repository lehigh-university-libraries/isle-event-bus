# riq

riq reads an Islandora event from ActiveMQ and processes the event from either a microservice running locally or from a service in the cloud.

🚧 Full documentation is available at https://libops.github.io/riq/

## Purpose

This service was created to address three points of concern in the ISLE stack:

1. Be able to leverage Islandora microservices from a local machine without needing to deploy the microservices locally
2. Support event types currently not able to be processed by [islandora/alpaca](https://github.com/islandora/alpaca)
3. Create a liveness probe // healthcheck on the service

## Attribution

This is a rewrite of [islandora/alpaca](https://github.com/Islandora/alpaca), adding the ability to directly upload the source file to a microservice to enable unlocking shared microservice deployments.
