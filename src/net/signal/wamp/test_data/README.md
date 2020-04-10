# WAMP

```bash
# generate a self-signed certificate.
# key.pem contains the unencrypted private key used to sign the certificate
# cert.pem contains the certificate
openssl req -x509 -newkey rsa:4096 -nodes -keyout key.pem -out cert.pem -days 3650
```
