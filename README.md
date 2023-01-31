## Zero

This directory is the Core Service Backend for Zero Fintech. It is deployed on Railway as an EC2 instance.
This backend is the central gatewaty for all requests from the frontend. And the central location to the Core Mongo DB database 
The backend contains the plaid API server and the core server functionality

### Installing

0. Install extra packages: 
    ```go install github.com/cosmtrek/air@latest```
    ```go install github.com/swaggo/swag/cmd/swag@latest```
1. Clone the repo
2. Create your own .env file
3. ```make dev```
4. view docs at http://localhost:8080/swagger

### Scripts

- ```make dev``` - runs the server in development mode
- ```make swagger``` - generates the swagger docs
