## Demo of Signicat OIDC using Go

This is a minimal demo showcasing how to integrate a Go based project with Signicat's OpenID Connect (OIDC) feature set, part of the Authentication product.

For further details regarding Signicat OpenID Connect, checkout [this starting reference](https://developer.signicat.com/docs/authentication/oidc.html#introduction-to-openid-connect).

<br/>

### Signicat Setup

In Signicat [Dashboard](https://dashboard.signicat.dev), go to _OIDC clients_ and create a client with:

-   Client name: `demo_client_1`
-   Primary Grant Type: `AuthorizationCode`
-   Redirect URI: `https://localhost:8087/oidc/authz-code`
-   Scope: `openid`

Continue with _Add secret_.
The generated client secret is `futiYv5jxPnqkt74KDdGJp2xbDSCwNyJ5mCYfg5hKtuITHwB`.

Also, the generated client ID in this case is `dev-orange-sponge-701`.

This also requires having a domain, so go to _Domain management_ and create one.<br/>
In this case, the FQDN is `demo-signicat-oidc-go.sandbox.signicat.dev`.

<br/>

### TLS Setup

Since this has been already done, and we have the required (`server.key` and `server.crt`) files, consider this as a future reference of how to do it yourself.

#### Generate the private key

1. `openssl genrsa -out server.key 2048`
2. `openssl ecparam -genkey -name secp384r1 -out server.key`

#### Generate the self-signed X.509 certificate

Based on the private key, generate the certificate (that includes the public key) and have it stored using PEM encoding (`.pem` or `.crt`) using:
`openssl req -new -x509 -sha256 -key server.key -out server.crt -days 3650`

For further details on this topic, checkout [this](https://github.com/denji/golang-tls) material.

<br/>

### The Flow

TODO
