# dnsdb

## Summary

dnsdb is tool designed to generated CDB and LMDB databases for dnsdist with plain test files

## Howto

### Build

```bash
go build cmd/dnsdb/*.go
```

### Sample config in dnsdb.yml

```yaml
- url: "https://dnsbl.com/data/local.txt"
  file: "/etc/dnsdist/db/local.lmdb"
  type: "ip"
- url: "https://dnsbl.com/data/ips.all.txt"
  file: "/etc/dnsdist/db/ips.all.lmdb"
  type: "ip"
- url: "https://dnsbl.com/data/ips.txt"
  file: "/etc/dnsdist/db/ips.lmdb"
  type: "ip"
- url: "https://dnsbl.com/data/domains.txt"
  file: "/etc/dnsdist/db/domains.cdb"
  type: "domain"
```

### Run

```bash
./dnsdb -configfile dnsdb.yml
```

## dnsdist config

```lua
local kvs_domains = newCDBKVStore("/etc/dnsdist/db/domains.cdb", 2)
local kvs_ips = newLMDBKVStore("/etc/dnsdist/db/ips.lmdb", "db")
local kvs_local_ips = newLMDBKVStore("/etc/dnsdist/db/local.lmdb", "db")


addAction(KeyValueStoreLookupRule(kvs_domains, KeyValueLookupKeyQName(false)), SetTagAction("policy", "block"))
addAction(KeyValueStoreRangeLookupRule(kvs_ips, KeyValueLookupKeySourceIP(32, 128, false)), SetTagAction("policy", "delay"))

addAction(TagRule("policy", "delay"), DelayAction(250))
addAction(TagRule("policy", "block"), DropAction())
```

## License

```text
Copyright (c) 2024 PaulBSD
All rights reserved.

Redistribution and use in source and binary forms, with or without
modification, are permitted provided that the following conditions are met:

1. Redistributions of source code must retain the above copyright notice, this
   list of conditions and the following disclaimer.
2. Redistributions in binary form must reproduce the above copyright notice,
   this list of conditions and the following disclaimer in the documentation
   and/or other materials provided with the distribution.

THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS" AND
ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE IMPLIED
WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE
DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT OWNER OR CONTRIBUTORS BE LIABLE FOR
ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES
(INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES;
LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND
ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
(INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE OF THIS
SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.

The views and conclusions contained in the software and documentation are those
of the authors and should not be interpreted as representing official policies,
either expressed or implied, of this project.
```
