# 💰 Paid Operator (Simplest Way)

## 🎯 Goal

* Operator is discoverable in OLM, so it can be listed
* Runtime requires license (via image pull secret)

---

# 🧱 Architecture

```text
Public:
  - Index Image (CatalogSource)
  - Bundle Image

Private:
  - Controller Image (actual operator runtime)
```

---
# 1. Update CSV (IMPORTANT)

Edit the CSV file to include a reference to the pull secret.

```text
bundle/manifests/*.clusterserviceversion.yaml
```

Add:

```yaml
spec:
  install:
    spec:
      deployments:
        - name: osaora-demo-operator-controller-manager
          spec:
            template:
              spec:
                imagePullSecrets:
                  - name: quay-secret
```

---

# 2. Rebuild + Push Bundle & Index Again

```bash
podman build -f bundle.Dockerfile \
  -t quay.io/ooransa/osaora-demo-operator-bundle:v0.0.3 .

podman push quay.io/ooransa/osaora-demo-operator-bundle:v0.0.3
```

```bash
opm index add \
  --bundles quay.io/ooransa/osaora-demo-operator-bundle:v0.0.3 \
  --tag quay.io/ooransa/osaora-demo-operator-index:v0.0.3 \
  --container-tool podman

podman push quay.io/ooransa/osaora-demo-operator-index:v0.0.3
```


# 3. Create CatalogSource (NO AUTH NEEDED)

This will refer to the public index image, so it will be listed in the catalog.

```yaml
apiVersion: operators.coreos.com/v1alpha1
kind: CatalogSource
metadata:
  name: demo-operator-catalog
  namespace: openshift-marketplace
spec:
  displayName: Demo Operator
  publisher: Osa Ora
  sourceType: grpc
  image: quay.io/ooransa/osaora-demo-operator-index:v0.0.3
```

```bash
oc apply -f catalogsource.yaml
```

---

# 4. Create Subscription

From the GUI or using the following YAML format
This will not work and installation will stuck as controller image is private.

```yaml
apiVersion: operators.coreos.com/v1alpha1
kind: Subscription
metadata:
  name: osaora-demo-operator
  namespace: demos
spec:
  channel: stable
  name: osaora-demo-operator
  source: demo-operator-catalog
  sourceNamespace: openshift-marketplace
  installPlanApproval: Automatic
```

---

# 5. Create Pull Secret (CUSTOMER STEP)

This will be based on customer subscription..

```bash
oc create secret docker-registry quay-secret \
  --docker-server=quay.io \
  --docker-username=USERNAME \
  --docker-password=TOKEN \
  --docker-email=EMAIL \
  -n demos
```

---

# 6. Attach Secret to Operator ServiceAccount

```bash
oc secrets link \
  osaora-demo-operator-controller-manager \
  quay-secret \
  --for=pull \
  -n demos
```

Make sure the imagePullSecret exists in namespace "demos" before the operator deployment pods are created (or restart the deployment after creating the secret).

---

# 🚀 RESULT

## Without license (pull secret):

* Operator installs
* Pod stuck in:

  ```
  ImagePullBackOff
  ```

## With license (pull secret):

* Operator runs normally
* full functionality enabled

---

By doing this you simplify the subscription with your customers to use the operator, you can also have another logic inside the controller if you want to control the number of resources granted for the customer.
Note: This is not a real billing mechanism. It is an access-control model based on private registry authentication.


