# How to run Kratos

HELM:

1. `kubectl apply -f 00-namespace.yaml 01-postgres.yaml`
2. ```
   helm install kratos --namespace ory-sit \
      -f kratos.yaml \
      ory/kratos
   ```

Note: whenever there is a change in mapper convert it to base64
https://www.base64encode.org/ and apply it with this command

```shell
helm upgrade kratos --namespace ory-sit \
   -f kratos.yaml \
   ory/kratos
```
