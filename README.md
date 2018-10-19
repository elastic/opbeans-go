# opbeans-go

This is a Go implementation of the Opbeans Demo app.

## Running locally

The simplest way to run this demo is by using the
provided docker-compose.yml:

```bash
docker-compose up
```

## Running with Elastic Cloud

0. Start Elastic Cloud [trial](https://www.elastic.co/cloud/elasticsearch-service/signup) (if you don't have it yet)
1. Add environmental variables `ELASTIC_CLOUD_ID` and `ELASTIC_CLOUD_CREDENTIALS` (in format `login:password`)
2. Run 
```bash
docker-compose -f docker-compose-elastic-cloud.yml up
```