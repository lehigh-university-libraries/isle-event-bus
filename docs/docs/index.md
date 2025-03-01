# riq

riq reads the Islandora Queue and processes the event.

## Basic overview

This service reads an Islandora event from either ActiveMQ or an HTTP endpoint and processes the event from either a microservice running locally or from a service in the cloud

## Purpose

This service was created to address three points of concern in the ISLE stack:

1. Be able to leverage Islandora microservices from a local machine without needing to deploy the microservices locally
2. Provide a mechanism to replay missed events
3. Support event types currently not able to be processed by [islandora/alpaca](https://github.com/islandora/alpaca)
