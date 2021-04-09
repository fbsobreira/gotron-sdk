# TRON's go-sdk

GoSDK and TRON-CLI tool for TRON's blockchain via GRPC

# Build


```
$ git pull -r origin master
$ make
```

# Usage & Examples

# bash completions

once built, add `tronctl` to your path and add to your `.bashrc`

```
. <(tronctl completion)
```

## Transfer JSON file format
The JSON file will be a JSON array where each element has the following attributes:

| Key                 | Value-type | Value-description|
| :------------------:|:----------:| :----------------|
| `from`              | string     | [**Required**] Sender's one address, must have key in keystore. |
| `to`                | string     | [**Required**] The receivers one address. |
| `amount`            | string     | [**Required**] The amount to send in $ONE. |
| `passphrase-file`   | string     | [*Optional*] The file path to file containing the passphrase in plain text. If none is provided, check for passphrase string. |
| `passphrase-string` | string     | [*Optional*] The passphrase as a string in plain text. If none is provided, passphrase is ''. |
| `stop-on-error`     | boolean    | [*Optional*] If true, stop sending transactions if an error occurred, default is false. |

Example of JSON file:

```json
[
  {
    "from": "TUEZSdKsoDHQMeZwihtdoBiN46zxhGWYdH",
    "to": "TKSXDA8HfE9E1y39RczVQ1ZascUEtaSToF",
    "amount": "1",
    "passphrase-string": "",
    "stop-on-error": true
  },
  {
    "from": "TUEZSdKsoDHQMeZwihtdoBiN46zxhGWYdH",
    "to": "TEvHMZWyfjCAdDJEKYxYVL8rRpigddLC1R",
    "amount": "1",
    "passphrase-file": "./pw.txt",
  }
]
```


# Debugging

The gotron-sdk code respects `GOTRON_SDK_DEBUG` as debugging
based environment variables.

```bash
GOTRON_SDK_DEBUG=true ./tronctl
```


# GRPC TLS

If you node require TLS connection, use parameter `--withTLS`
TLS credentials can also be set persistent in config file: `withTLS: true`

# Trongrid API Key

To set trongrid API Key first create you api key at `www.trongrid.io` and use parameter 
 `--apiKey=25f66928-0b70-48cd-9ac6-da6f8247c663` (replace with your API key)
Trongrid API Key can also be set persistent in config file: `apiKey: 25f66928-0b70-48cd-9ac6-da6f8247c663` (replace with your API key)

OS environment variable `TRONGRID_APIKEY` will overwrite any prior API key configuration if set.