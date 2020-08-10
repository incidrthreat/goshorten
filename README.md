# GoShorten

This project spawned due to my curiousity in gRPC, golang, and building something practical to share.
___________________________
## What is GoShorten?
GoShorten is a self-hosted URL Shortener written in Golang.  It uses a gRPC server on the "backend" for API calls and stores data in a Redis Database.  The current Time-To-Live for each URL/Code is setable via the webgui.  Options for 5 min, 24 hrs, and 48 hrs are available.
___________________________
## Getting Started

### Prerequisites
- [Docker](https://docs.docker.com/get-docker/)
- [Docker-Compose](https://docs.docker.com/compose/install/)

### How to Run GoShorten:
1. `git clone https://github.com/incidrthreat/goshorten.git`

2. `cd goshorten`

3. Execute `./generate-tls-certs.sh` to generate the TLS certs.

4. In the /backend directory, rename `config.json.example` to `config.json`
    1. Currently only supports Redis.

5. `docker-compose up` or `docker-compose up -d` 
    1. Redis "password" is on line 19 in `docker-compose.yml`
    2. Change password as necessary

6. open your favorite browser to `localhost:8081`

___________________________
## Screenshots
#### Home Page
![Home Page](/screenshots/homepageV1.0.2.png)
#### Successful Code creation
![Success!](/screenshots/successfulcodeV1.0.2.png)
#### Invalid Code retreival
![Invalid](/screenshots/invalidcodeV1.0.2.png)
__________________________
## Contributing

If you are interested in contributing to this project please send an email to `incidrthreat@hackmethod.com` or submit a PR with any changes you'd like to see.  If you run into issues please submit an issues "ticket" [here](https://github.com/incidrthreat/goshorten/issues).
___________________________
## Authors/Contributors

* *Initial* - [Incidrthreat](https://twitter.com/incidrthreat)