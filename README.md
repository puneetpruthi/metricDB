# Metric Data Store Service

This project aims at creating a real time Metric Data Store Service that is modeled as a 12-factor application (https://12factor.net).

The application accepts 2 gRPC client calls:
- Set Metric for a specific UID object
- Get Metric for a specified UID spread over N-points in the given time interval

To connect to this service, the docker compose file defines this service be exposed at port 6000 on the host.

The project also provides a skeleton for the client code that was used for testing the functionality of the application.

## Deployment

To deploy these projects, you must have:
* git
* docker
pre-installed on your system.

After you have the pre-requisites installed on your system, clone the github repository and fire up docker-compose in the repository directory

## Technologies Used

Language Used: golang (version 1.9.1)
Database Used: redis
 
A golang wrapper API library was used to access the database to help in accessing the database as a time series database instead of a simple key-value store

redis was chosen as a database for the following reasons:
- has a simple golang wrapper to use as time series DB
- wrapper provides in-built concurrency model to improve parallel processing
- cluster client flexibility helps in scaling up to 1000s of application deployment
- easy pluggability to docker compose ecosystem
- online research showed high score for redis performance
- ease of use (familiarity)

For vendoring dependencies for the project, an open-source tool (vndr) was used. (Source https://github.com/LK4D4/vndr).
Reasons for choosing this tool:
- Familiarty while working on docker
- Simplicity of the tool for such projects
- light weight and explicit processing of project dependency

## Author(s)

* **Puneet Pruthi** - *Initial work*

## References

* https://github.com/donnpebe/go-redis-timeseries
* https://github.com/grpc/grpc-go
* https://12factor.net/
* https://grpc.io/
