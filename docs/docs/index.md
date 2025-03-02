# riq

riq reads the Islandora Queue and processes the event.

## Basic overview

This service reads an Islandora event from ActiveMQ and processes the event from either a microservice running locally or from a service in the cloud.

## Purpose

This service was created to address two points of concern in the ISLE stack:

1. Be able to leverage Islandora microservices from a local machine without needing to deploy the microservices locally
2. Support event types currently not able to be processed by [islandora/alpaca](https://github.com/islandora/alpaca)
