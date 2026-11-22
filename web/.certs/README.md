# Development certificates

The dev server serves HTTPS when `key.pem` and `cert.pem` exist in this folder, and plain HTTP otherwise.
Both files stay out of Git.

Create them with [mkcert](https://github.com/FiloSottile/mkcert), whose local CA the browsers on this machine trust:

```sh
mkcert -install
mkcert -key-file key.pem -cert-file cert.pem localhost 127.0.0.1 ::1
```

Add further host names or IP addresses to the second command to reach the dev server from other devices,
which then need the CA from `mkcert -CAROOT` as well. Restart the dev server after creating or removing the files.
