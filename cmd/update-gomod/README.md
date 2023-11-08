# update-gomod

`update-gomod` is a command line tool to update go modules.
This tool updates direct dependencies listed in `go.mod` one by one using `go get -d <packages>`.

## Usage

```console
$ go install github.com/masa213f/tools/cmd/update-gomod@latest
$ cd <directory containing go.mod>
$ update-gomod
```
