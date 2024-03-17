![Rancher + Let's Encrypt = Awesome Sauce](https://raw.githubusercontent.com/danibjor/rancher-letsencrypt/master/hero.png)

# Let's Encrypt Certificate Manager for Rancher

[![Latest Version](https://img.shields.io/github/release/danibjor/rancher-letsencrypt.svg?maxAge=8600)][release]
[![Docker Pulls](https://img.shields.io/docker/pulls/danibjor/rancher-letsencrypt?style=for-the-badge&maxAge=8600)][hub]
[![Publish Docker image](https://github.com/danibjor/rancher-letsencrypt/actions/workflows/docker-image.yml/badge.svg)](https://github.com/danibjor/rancher-letsencrypt/actions/workflows/docker-image.yml)

[release]: https://github.com/danibjor/rancher-letsencrypt/releases
[hub]: https://hub.docker.com/r/danibjor/rancher-letsencrypt/

A [Rancher](http://rancher.com/rancher/) service that obtains free SSL/TLS certificates from the [Let's Encrypt CA](https://letsencrypt.org/), adds them to Rancher's certificate store and manages renewal and propagation of updated certificates to load balancers.

# Fork Notes v2

Forked to ensure this repo don't get lost, as I use this image one some personal stuff.

* Changed wait interval when using http for validation, from 120s to 15s (tired of waiting). 
* Also set up Github Action that automatically pushed latest code to Docker Hub.

# Fork Notes

This is a forked version of [vxcontrol/rancher-letsencrypt](https://github.com/vxcontrol/rancher-letsencrypt/)

I have updated the cloudflare provider to support Cloudflare's new API Token
instead of using global API Key (will be removed later).
This will improve account security when generating single or wild certificates for certain domain when you managing multiple domains in one account.

Some provider were removed because of raised errors when running builds. These will be re-added back later.

#### Changes
* The following providers were removed in this forked version:
  * DNSSimple
  * Azure DNS
* Replaced xenolf/lego go library with go-acme/lego v3, using version v3.1.0
* Migrated to go modules
* Release version using 2.x from now on, starting from 2.0.0-beta.x as a preview release
* Built with alpine 3.12 as base environment

#### Plans/Roadmap
* Upgrade go version to 1.14
* Upgrade `go-acme/lego` to v4.x
* Add support for various DNS providers including previously removed providers
* Remove Cloudflare's global API key usage
* Fully use environment variables used by **lego** instead of defining our own
  This is entirely because adding support for various provider is tiresome when I have to implement each of its own provider options

#### Requirements
* Rancher Server >= v1.5.0
* If using a DNS-based challenge, existing account with one of the supported DNS providers:
  * `Aurora DNS`
  * `AWS Route 53`
  * `CloudFlare`
  * `DigitalOcean`
  * `Dyn`
  * `Gandi`
  * `NS1`
  * `Ovh`
  * `Vultr`

* If using the HTTP challenge, a reverse proxy that routes `example.com/.well-known/acme-challenge` to `rancher-letsencrypt`. 

### How to use

This application is distributed via the [Rancher Community Catalog](https://github.com/rancher/community-catalog).

Enable the Community Catalog under `Admin` => `Settings` in the Rancher UI.
Then locate the `Let's Encrypt` template in the Catalog section of the UI and follow the instructions.

### Storing certificate in shared storage volume

By default the created SSL certificate is stored in Rancher's certificate store for usage in Rancher load balancers.

You can specify a volume name to store account data, certificate and private key in a (host scoped) named Docker volume.
To share the certificates with other services you may specify a persistent storage driver (e.g. rancher-nfs).

See the README in the Rancher catalog for more information.

#### Configuration reference

You can either set environment variables or use Rancher Secrets for provider configuration.

Possible options are:

*Aurora*

- AURORA_USER_ID - /run/secrets/aurora_user_id
- AURORA_KEY - /run/secrets/aurora_key
- AURORA_ENDPOINT - /run/secrets/aurora_endpoint

*CloudFlare*
- CF_API_EMAIL - /run/secrets/cf_api_email
- CF_API_KEY - /run/secrets/cf_api_key
- CF_DNS_API_TOKEN - /run/secrets/cf_dns_api_token
- CF_ZONE_API_TOKEN - /run/secrets/cf_zone_api_token

*DigitalOcean*
- DO_ACCESS_TOKEN - /run/secrets/do_access_token

*AWS*
- AWS_ACCESS_KEY - /run/secrets/aws_access_key
- AWS_SECRET_KEY - /run/secrets/aws_secret_key

*DYN*
- DYN_CUSTOMER_NAME - /run/secrets/dyn_customer_name
- DYN_USER_NAME - /run/secrets/dyn_user_name
- DYN_PASSWORD - /run/secrets/dyn_password

*VULTR*
- VULTR_API_KEY - /run/secrets/vultr_api_key

*OVH*
- OVH_ENDPOINT - /run/secrets/ovh_application_key
- OVH_APPLICATION_KEY - /run/secrets/ovh_application_key
- OVH_APPLICATION_SECRET - /run/secrets/ovh_application_secret
- OVH_CONSUMER_KEY - /run/secrets/ovh_consumer_key

*GANDI*
- GANDI_API_KEY - /run/secrets/gandi_api_key

*NS1*
- NS1_API_KEY - /run/secrets/ns1_api_key


### Provider specific usage

#### AWS Route 53

Note: If you have both a private and public zone in Route53 for the domain, you need to run the service configured with public DNS resolvers (this is now the default).

The following IAM policy describes the minimum permissions required when using AWS Route 53 for domain authorization.    
Replace `<HOSTED_ZONE_ID>` with the ID of the hosted zone that encloses the domain(s) for which you are going to obtain certificates. You may use a wildcard (*) in place of the ID to make this policy work with all of the hosted zones associated with an AWS account.

```json
{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Effect": "Allow",
            "Action": [
                "route53:GetChange",
                "route53:ListHostedZonesByName"
            ],
            "Resource": [
                "*"
            ]
        },
        {
            "Effect": "Allow",
            "Action": [
                "route53:ChangeResourceRecordSets"
            ],
            "Resource": [
                "arn:aws:route53:::hostedzone/<HOSTED_ZONE_ID>"
            ]
        }
    ]
}
```

#### OVH

First create your credentials on https://eu.api.ovh.com/createToken/ by filling out the form like this:

- `Account ID`: Your OVH account ID
- `Password`: Your password
- `Script name`: letsencrypt
- `Script description`: Letsencrypt for Rancher
- `Validity`: Unlimited
- `Rights`:
  - GET /domain/zone/*
  - POST /domain/zone/*
  - DELETE /domain/zone/*

Then deploy this service using the generated key, application secret and consumer key.

#### HTTP

If you prefer not to use a DNS-based challenge or your provider is not supported, you can use the HTTP challenge.
Simply choose `HTTP` from the list of providers.
Then make sure that HTTP requests to `domain.com/.well-known/acme-challenge` are forwarded to port 80 of the `rancher-letsencrypt` service, e.g. by configuring a Rancher load balancer accordingly. If you are using another reverse proxy (e.g. Nginx) you need to make sure it passed the original `host` header through to the backend.

![Rancher Load Balancer Let's Encrypt Targets](https://cloud.githubusercontent.com/assets/198988/22224463/0d1eb4aa-e1bf-11e6-955c-5f0d085ce8cd.png)

### Building the image

`make build && make image`

### Contributions

PR's welcome!

# Credits
* Original author [janeczku/rancher-letsencrypt](https://github.com/janeczku/rancher-letsencrypt)
* Original fork [vxcontrol/rancher-letsencrypt](https://github.com/vxcontrol/rancher-letsencrypt)
