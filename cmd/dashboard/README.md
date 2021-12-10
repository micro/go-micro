# Go Micro Dashboard

## Installation

```
go install github.com/asim/go-micro/cmd/dashboard/v4@latest
```

## Usage

```
dashboard --registry etcd --server_address :4000
```

## Docker

```
docker run -d --name micro-dashboard -p 8082:8082 xpunch/go-micro-dashboard:latest
```

Visit: [http://localhost:4000](http://localhost:4000)(deafult admin@micro)
