# route-to-ingress-operator

A controller to create corresponding `ingress.networking.k8s.io/v1` resources for `route.openshift.io/v1`


## TODO

- [x] int port
- [x] string port
- [x] path
- [ ] nil port
- [ ] weight
- [ ] termination

## Instructions

### Development

* `make generate` update the generated code for that resource type.
* `make manifests` Generating CRD manifests.
* `make test` Run tests.


### Build

* `make build` builds golang app locally.
* `make docker-build` build docker image locally.
* `make docker-push` push container image to registry.

### Run, Deploy
* `make run` run app locally
* `make deploy` deploy to k8s.

### Clean up

* `make undeploy` delete resouces in k8s.

## Security

### Reporting security vulnerabilities

If you find a security vulnerability or any security related issues, please DO NOT file a public issue, instead send your report privately to myusefpur@gmail.com. Security reports are greatly appreciated and we will publicly thank you for it.

For more info please see [here](SECURITY.md).

## License

Apache-2.0 License, see [LICENSE](LICENSE).
