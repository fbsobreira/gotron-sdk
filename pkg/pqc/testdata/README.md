# pkg/pqc test data

## falcon512-KAT.rsp.gz

BouncyCastle Falcon-512 known-answer tests (100 entries), used by
`falcon_liboqs_test.go` (`-tags falcon`) to check the liboqs FN-DSA-512 backend
against the verifier that TRON nodes use (BouncyCastle 1.84), and by
`falcon_keycheck_test.go` (every build) as positive controls for the pure-Go
key-pair validation. The parser lives in `falcon_kat_test.go`.

- Source: <https://github.com/bcgit/bc-test-data/blob/main/pqc/crypto/falcon/falcon512-KAT.rsp>
- SHA-256 of the uncompressed file:
  `dd75c946fdedef4ec46a2bee7e10c65c9126f1a839b9ced6921fd45f7354b5cd`
  (the test checks it before parsing)
- Compressed with `gzip -9 -n`, so the archive is reproducible:
  `gzip -9 -n < falcon512-KAT.rsp > falcon512-KAT.rsp.gz`

Each entry has `count`, `seed`, `mlen`, `msg`, `pk` (897 bytes: `0x09 || h`),
`sk` (1281 bytes: `0x59 || f || g || F`), `smlen` and `sm`, where
`sm = siglen (2 bytes, big-endian) || nonce (40) || msg (mlen) || 0x29 || compressed s2 (siglen - 1)`.
The detached TRON signature is `0x39 || sm[2:42] || sm[42+mlen+1:]`.

These are published test-only keys; never use them for real funds.
