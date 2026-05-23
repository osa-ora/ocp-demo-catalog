
Steps to build and Create A basic demo operator
---

# Building an OpenShift Operator with Operator SDK + OLM for First Time

## 1. Install Required Tools

### Install Operator SDK

```bash id="1"
brew install operator-sdk
```

### Install Go

```bash id="2"
brew install go
```

### Install Podman

```bash id="3"
brew install podman
```

### Install OPM (Operator Package Manager)

Mac ARM64:

```bash id="4"
curl -L -o opm \
https://github.com/operator-framework/operator-registry/releases/latest/download/darwin-arm64-opm

chmod +x opm

sudo mv opm /usr/local/bin/
```

---

# 2. Create the Operator Project

```bash id="5"
mkdir osaora-demo-operator

cd osaora-demo-operator
```

Initialize the operator:

```bash id="6"
operator-sdk init \
  --domain osa.ora \
  --repo osa.ora/demo-operator
```

---

# 3. Create API + Controller

```bash id="7"
operator-sdk create api \
  --group demo \
  --version v1alpha1 \
  --kind DemoRequest \
  --resource \
  --controller
```

Answer:

```text id="8"
Create Resource: yes
Create Controller: yes
```

---

# 4. Implement API Spec

Edit:

```text id="9"
api/v1alpha1/demorequest_types.go
```

Define the CR spec/status.

Example:

```go id="10"
type DemoRequestSpec struct {
    DemoName string `json:"demoName,omitempty"`
    Namespace string `json:"namespace,omitempty"`
}
```

---

# 5. Implement Reconcile Logic

Edit:

```text id="11"
internal/controller/demorequest_controller.go
```

Implement your reconciliation logic.

---

# 6. Generate CRDs and Manifests

```bash id="12"
make generate

make manifests
```

---

# 7. Test Operator Locally

Login to OpenShift first:

```bash id="13"
oc login ...
```

Install CRDs locally:

```bash id="14"
make install
```

Run operator locally:

```bash id="15"
go build ./...

go run ./cmd/main.go
```

---

# 8. Build Operator Image

⚠ IMPORTANT:
OpenShift worker nodes are typically Linux AMD64.
Mac builds can accidentally produce ARM64 images.

Build explicitly for Linux AMD64:

```bash id="16"
podman build \
  --platform linux/amd64 \
  -t quay.io/ooransa/osaora-demo-operator:v0.0.3 .
```

Push image:

```bash id="17"
podman push quay.io/ooransa/osaora-demo-operator:v0.0.3
```

Verify architecture:

```bash id="18"
podman inspect quay.io/ooransa/osaora-demo-operator:v0.0.3 | grep Architecture
```

Expected:

```text id="19"
"Architecture": "amd64"
```

---

# 9. Build Bundle

Generate bundle:

```bash id="20"
make bundle
```

---

# 10. Add Operator Icon

Prepare PNG image:

<img width="128" height="128" alt="operator" src="https://github.com/user-attachments/assets/d5fa6039-1ada-46a2-a19e-6eaf92142878" />


* 64x64 OR 128x128

Convert to base64:

```bash id="21"
base64 -i operator.png | tr -d '\n' | pbcopy
```

Edit CSV:

```text id="22"
bundle/manifests/osaora-demo-operator.clusterserviceversion.yaml
```

Add:

```yaml id="23"
icon:
  - base64data: <BASE64_DATA>
    mediatype: image/png
```

---

# 11. Fix CSV Deployment Image

Inside CSV, replace:

```yaml id="24"
image: controller:latest
```

with:

```yaml id="25"
image: quay.io/ooransa/osaora-demo-operator:v0.0.3
```

Also ensure:

```yaml id="26"
command:
  - /manager
```

exists correctly.

---

# 12. Add RBAC Permissions

Inside CSV:

```text id="27"
install:
  spec:
```

Add permissions for Deployments:

```yaml id="28"
clusterPermissions:
  - serviceAccountName: osaora-demo-operator-controller-manager
    rules:
      - apiGroups:
          - apps
        resources:
          - deployments
        verbs:
          - get
          - list
          - watch
          - create
          - update
          - patch
          - delete
```

⚠ Without this, the operator cannot manage Deployments or others object, so add all objects you need or just grant permissions later on as in file role.yaml.

---

# 13. Build Bundle Image

Login to Quay:

```bash id="29"
podman login quay.io
```

Build bundle image:

```bash id="30"
podman build \
  -f bundle.Dockerfile \
  -t quay.io/ooransa/osaora-demo-operator-bundle:v0.0.3 .
```

Push:

```bash id="31"
podman push quay.io/ooransa/osaora-demo-operator-bundle:v0.0.3
```

---

# 14. Build Index Image

⚠ Run from writable folder like `/tmp/opm-index`

```bash id="32"
cd /tmp

mkdir -p opm-index

cd opm-index
```

Build index:

```bash id="33"
opm index add \
  --bundles quay.io/ooransa/osaora-demo-operator-bundle:v0.0.3 \
  --tag quay.io/ooransa/osaora-demo-operator-index:v0.0.3 \
  --container-tool podman
```

Push index:

```bash id="34"
podman push quay.io/ooransa/osaora-demo-operator-index:v0.0.3
```

---

# 15. Create CatalogSource

```yaml id="35"
apiVersion: operators.coreos.com/v1alpha1
kind: CatalogSource
metadata:
  name: demo-operator-catalog
  namespace: openshift-marketplace
spec:
  sourceType: grpc
  image: quay.io/ooransa/osaora-demo-operator-index:v0.0.3
  displayName: Demo Request Operator
  publisher: Osama Oransa
```

Apply:

```bash id="36"
oc apply -f catalogsource.yaml
```

---

# 16. Verify CatalogSource

```bash id="37"
oc get catalogsource -n openshift-marketplace
```

Expected:

```text id="38"
READY
```

---

# 17. Install Operator from OperatorHub

Go to:

```text id="39"
Operators → OperatorHub
```

<img width="456" height="596" alt="Screenshot 2026-05-21 at 2 39 23 PM" src="https://github.com/user-attachments/assets/93b462e8-41b6-43c6-ab27-1df97221c813" />

Find:

```text id="40"
OsaOra Demo Operator
```

Install it.

---

# 18. Verify Operator Running

```bash id="41"
oc get pods -A | grep osaora
```

Check logs:

```bash id="42"
oc logs deploy/osaora-demo-operator-controller-manager -n demos
```

Expected healthy logs:

```text id="43"
starting manager
Starting Controller
Starting workers
```

---

# 19. Create DemoRequest CR

Example:

```yaml id="44"
apiVersion: demo.osa.ora/v1alpha1
kind: DemoRequest
metadata:
  name: test-nginx
  namespace: demos
spec:
  demoName: nginx
  namespace: demos
```

Apply:

```bash id="45"
oc apply -f demorequest.yaml
```
Or from the GUI directly:

<img width="1025" height="608" alt="Screenshot 2026-05-21 at 2 50 08 PM" src="https://github.com/user-attachments/assets/938a2d0f-d6f0-47d8-85e1-3a56ff320ea1" />

---

# 20. Important Lessons Learned

## Image Architecture Matters

Mac builds may generate ARM64 images.

OpenShift nodes usually require AMD64.

Always verify:

```bash id="46"
podman inspect IMAGE | grep Architecture
```

---

## CSV Image Must Match Real Operator Image

Wrong:

```yaml id="47"
image: controller:latest
```

Correct:

```yaml id="48"
image: quay.io/ooransa/osaora-demo-operator:v0.0.3
```

---

## Bundle + Index Are NOT the Runtime Operator Image

You need THREE images:

| Image          | Purpose                   |
| -------------- | ------------------------- |
| Operator Image | Actual controller runtime |
| Bundle Image   | Metadata + CSV            |
| Index Image    | OLM catalog               |

---

## Public Repositories Required

Ensure all Quay repositories are public:

* operator image
* bundle image
* index image

Otherwise OpenShift cannot pull them.


# Updating the OpenShift Operator

## Optionally you can create additional APIs/CRD: Create API + Controller

```
operator-sdk create api \
  --group demo \
  --version v1alpha1 \
  --kind DemoRequestNew \
  --resource \
  --controller
```

But if no need for more resources, modify the code in the controller, modify the main.go and the CSV.yaml file with the new version and container images or required roles/RBAC.

Then build these images: 

## Build bundle image:

```
podman build \
  -f bundle.Dockerfile \
  -t quay.io/ooransa/osaora-demo-operator-bundle:v0.0.3 .
```

Push:

```
podman push quay.io/ooransa/osaora-demo-operator-bundle:v0.0.3
```

---

## Build Index Image

⚠ Run from writable folder like `/tmp/opm-index`

```
cd /tmp

mkdir -p opm-index

cd opm-index
```

Build index:

```
opm index add \
  --bundles quay.io/ooransa/osaora-demo-operator-bundle:v0.0.3 \
  --tag quay.io/ooransa/osaora-demo-operator-index:v0.0.3 \
  --container-tool podman
```

Push index:

```
podman push quay.io/ooransa/osaora-demo-operator-index:v0.0.3
```

## Build Operator Image

⚠ IMPORTANT:
OpenShift worker nodes are typically Linux AMD64. Mac builds can accidentally produce ARM64 images.

Build explicitly for Linux AMD64:

```
podman build \
  --platform linux/amd64 \
  -t quay.io/ooransa/osaora-demo-operator:v0.0.3 .
```

Push image:

```
podman push quay.io/ooransa/osaora-demo-operator:v0.0.3
```

Verify architecture:

```
podman inspect quay.io/ooransa/osaora-demo-operator:v0.0.3 | grep Architecture
```

Expected:

```
"Architecture": "amd64"
```

## Re-install the operator reference the new index image:

```
apiVersion: operators.coreos.com/v1alpha1
kind: CatalogSource
metadata:
  name: demo-operator-catalog
  namespace: openshift-marketplace
spec:
  sourceType: grpc
  image: quay.io/ooransa/osaora-demo-operator-index:v0.0.3
  displayName: Demo Request Operator
  publisher: Osama Oransa
```

