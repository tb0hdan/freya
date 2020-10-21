# freya
[DomainsProject.org](https://domainsproject.org) DNS worker


WORK IN PROGRESS


[Docker Hub](https://hub.docker.com/r/tb0hdan/freya)

```bash
docker pull tb0hdan/freya
```

```bash
docker run --env FREYA=123 --rm tb0hdan/freya
```


For the brave:

1. Create `.env` file with contents like this: `FREYA=123` where `123` is your API key.
2. Run `./start.sh` (will invoke docker-compose and start `3` containers)

