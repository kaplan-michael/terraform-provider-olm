# Terraform Provider OLM


## Introduction
This is a Terraform provider for [Operator Lifecycle Manager](https://github.com/operator-framework/operator-lifecycle-manager).
It can be used to install OLM and manage operators through subscriptions.

## Status

This provider is a proof of concept using mostly the installer module from
[operator-sdk](https://github.com/operator-framework/operator-sdk) and code derived from it.
It is what I would call a spaghetti code held together by hope.
**Use in production at you own risk**
Currently supports only OLM v0.
OLM v1 support is planned soon, but probably after this gets cleaned up a bit.

## Requirements

- [Terraform](https://developer.hashicorp.com/terraform/downloads) >= 1.0
- [Go](https://golang.org/doc/install) >= 1.21

## Building The Provider

1. Clone the repository
1. Enter the repository directory
1. Build the provider using the Go `install` command:
2. Or build the provider using the `go build` command
```shell
go install
```
1. Or build the provider using the Go `build` command:
```shell
go build
```

```shell

## Adding Dependencies

This provider uses [Go modules](https://github.com/golang/go/wiki/Modules).
Please see the Go documentation for the most up to date information about using Go modules.

To add a new dependency `github.com/author/dependency` to your Terraform provider:

```shell
go get github.com/author/dependency
go mod tidy
```

Then commit the changes to `go.mod` and `go.sum`.

## Using the provider

You need a kubeconfig file to use this provider. It can come from anywhere, but must be set as raw in the provider configuration.
It will use the current context from the kubeconfig file.

```hcl
provider "olm" {
  kubeconfig = file("~/.kube/config")
}
```
You can also pass in the certificates directly, but do that if you know what you are doing.
For more information on how to use the provider, see the [examples](./examples) directory.
## Developing the Provider

If you wish to work on the provider, you'll first need [Go](http://www.golang.org) installed on your machine (see [Requirements](#requirements) above).

To compile the provider, run `go install`. This will build the provider and put the provider binary in the `$GOPATH/bin` directory.
Or run `go build`. This will build the provider and put the provider binary in the local directory.

To generate or update documentation, run `go generate`.

In order to run the full suite of Acceptance tests, run `make testacc`.
To run a lint, run `make lint`. (you need golangci-lint installed)


*Note:* Acceptance tests create real resources, and often cost money to run.

## Contributing

Contributions are welcome!

## License
Most of the actual provider code is licensed under Mozilla Public License 2.0.

The imporant part(everything under [internal/olm](internal/olm)) is licensed under the Apache 2.0 license.
That is the code that was taken or derived from [operator-sdk](https://github.com/operator-framework/operator-sdk)
