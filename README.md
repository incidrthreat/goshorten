# GoShorten

This project spawned due to my curiousity in gRPC, golang, and building something practical to share.
___________________________
## What is GoShorten?
GoShorten is a self-hosted URL Shortener written in Golang.  It uses a gRPC server on the "backend" for API calls and stores data in on a Redis Database.
___________________________
## Getting Started

### Prerequisites
- [Docker](https://docs.docker.com/get-docker/)
- [Docker-Compose](https://docs.docker.com/compose/install/)

### How to Run GoShorten:
1. `git clone https://github.com/incidrthreat/goshorten.git`

2. `cd goshorten`

3. In the /backend directory, rename `config.json.example` to `config.json`
    1. Currently only supports Redis.

4. `docker-compose up` or `docker-compose up -d` 
    1. Redis "password" is on line 19 in `docker-compose.yml`
    2. Change as necessary

5. open your favorite browser to `localhost:8081`
__________________________
## Contributing

If you are interested in contributing to this project please send an email to `incidrthreat@hackmethod.com` or submit a PR with any changes you'd like to see.
___________________________
## Authors/Contributors

* *Initial* - [Incidrthreat](https://twitter.com/incidrthreat)
