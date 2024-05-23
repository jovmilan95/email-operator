# email-operator
email-operator is a Kubernetes operator designed for efficient email configuration and delivery using MailerSend and Mailgun, with support for cross-namespace functionality.
## Getting Started

### Prerequisites
- go version v1.21.0+
- docker version 17.03+.
- kubectl version v1.11.3+.
- Access to a Kubernetes v1.11.3+ cluster.
- kustomize v5.4.1+

### To Deploy on the cluster
**Build and push your image to the location specified by `IMG`:**

```sh
make docker-build docker-push IMG=<some-registry>/email-operator:tag
```

**Deploy the operator to the cluster with the image specified by `IMG`:**

```sh
make deploy IMG=<some-registry>/email-operator:tag
```
**NOTE:** Image is automatically built and pushed to `jovmilan95/email-operator:latest` using GitHub Actions and is publicly available.Here is the link to the DockerHub: [DockerHub Repository](https://hub.docker.com/r/jovmilan95/email-operator)

**Create instances of your solution**
You can apply the samples (examples) from the config/sample:

```sh
kubectl apply -k config/samples/
```

**NOTE**: Ensure that the samples has default values to test it out.

### To Uninstall
**Delete the instances (CRs) from the cluster:**

```sh
kubectl delete -k config/samples/
```

**UnDeploy the operator from the cluster:**

```sh
make undeploy
```
