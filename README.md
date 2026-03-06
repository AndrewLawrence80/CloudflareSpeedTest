# CloudflareSpeedTest

Simplified version inspired by [XIU2 CloudflareSpeedTest](https://github.com/XIU2/CloudflareSpeedTest)

This implementation does not enumerate cloudflare ipv4 or ipv6 actively, but checks whether a domain is hosted on cloudflare
by resolving the domain ip through DNS. Domain list are imported from [v2fly domain list community](https://github.com/v2fly/domain-list-community)

## Compile from Source

You need golang to compile the repo

```sh
go build .
```
