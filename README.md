# Vuls-reporter

Use this tool to send Data to the vulsserver to generate vuls-reports based on your given options.

## Usage

vuls-reporter has flags

- help
  Show context-sensitive help (also try --help-long and --help-man).
- host
  Hostname(Regex)
- user
  User to authentificate at Database
- database
  Database that should be used

```bash
./vuls-reporter --help
usage: vuls-reporter --host=HOST --user=USER --database=DATABASE [<flags>]

Flags:
  --help               Show context-sensitive help (also try --help-long and --help-man).
  --host=HOST          Hostname(Regex)
  --user=USER          User to authentificate at Database
  --database=DATABASE  Database that should be used
```

### The database password has to be in your environment variable named `MINVPW`
