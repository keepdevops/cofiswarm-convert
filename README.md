# cofiswarm-convert

Cofiswarm component: `convert`.

- Layout: [REPO-STANDARD-LAYOUT](https://github.com/keepdevops/cofiswarmdev/blob/main/docs/REPO-STANDARD-LAYOUT.md)
- Migration: [MIGRATION-SPRINTS](https://github.com/keepdevops/cofiswarmdev/blob/main/docs/MIGRATION-SPRINTS.md)

## FHS paths

| Path | Purpose |
|------|---------|
| `/etc/cofiswarm/convert/` | config |
| `/var/lib/cofiswarm/convert/` | state |
| `/var/log/cofiswarm/convert/` | logs |

## Test

```bash
./test/scripts/assert-layout.sh convert
```
