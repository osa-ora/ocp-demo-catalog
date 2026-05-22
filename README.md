# OpenShift Demo Operator
---

In this demo, we built an operator that consumes an `index.yaml` file containing the list of available demos and their metadata. When the user requests a demo, the operator generates the required demo artifacts.

Once installed using the file `basic-operator/definition.yaml`:

```
apiVersion: operators.coreos.com/v1alpha1
kind: CatalogSource
metadata:
  name: demo-operator-catalog
  namespace: openshift-marketplace
spec:
  sourceType: grpc
  image: quay.io/ooransa/osaora-demo-operator-index:v0.0.2
  displayName: Demo Request Operator
  publisher: Osama Oransa
  updateStrategy:
    registryPoll:
      interval: 10m
```

Either run

```
oc apply -f https://raw.githubusercontent.com/osa-ora/ocp-demo-catalog/refs/heads/main/basic-operator/definition.yaml
```

Or add it using OpenShift Console.

The operator will be available in the software catalog in OpenShift GUI:

<img width="456" height="596" alt="Screenshot 2026-05-21 at 2 39 23 PM" src="https://github.com/user-attachments/assets/93b462e8-41b6-43c6-ab27-1df97221c813" />

Then select it and install it:

<img width="360" height="535" alt="Screenshot 2026-05-21 at 2 39 35 PM" src="https://github.com/user-attachments/assets/d151c844-0530-4070-989a-d960f2384a66" />

<img width="874" height="569" alt="Screenshot 2026-05-21 at 2 48 34 PM" src="https://github.com/user-attachments/assets/aec347cd-d734-4e98-9071-959f3fb3ab96" />

<img width="543" height="253" alt="Screenshot 2026-05-21 at 2 49 29 PM" src="https://github.com/user-attachments/assets/a57576c2-9c04-4e22-8ff3-43e0efa93bda" />

Then you can go and create the demo requests:

<img width="1253" height="665" alt="Screenshot 2026-05-21 at 2 49 39 PM" src="https://github.com/user-attachments/assets/693bfbbc-bcfe-4559-ae23-c0f9c3fa243b" />

<img width="1025" height="608" alt="Screenshot 2026-05-21 at 2 50 08 PM" src="https://github.com/user-attachments/assets/938a2d0f-d6f0-47d8-85e1-3a56ff320ea1" />

And now all demo request resources are related to that request and will be deleted once this demo request is deleted.

<img width="923" height="416" alt="Screenshot 2026-05-21 at 2 50 30 PM" src="https://github.com/user-attachments/assets/ac296fcf-9288-4562-8723-f91a6796e726" />


To add more demos, insert them into index.yaml file.

The 3 operator public images are located at: quay.io/ooransa/osaora-demo-operator:v0.0.2, quay.io/ooransa/osaora-demo-operator-index:v0.0.2
and quay.io/ooransa/osaora-demo-operator-bundle:v0.0.2

<img width="396" height="139" alt="Screenshot 2026-05-21 at 3 19 12 PM" src="https://github.com/user-attachments/assets/71d3e42c-1feb-4276-9790-bd2cf92999de" />


# To clarify the different images
---

## 1. Index image (CatalogSource)

**Purpose: discovery layer**

```text id="1"
CatalogSource → OLM reads index image → finds packages/channels/versions
```

It contains:

* package metadata
* channels (stable, beta, etc.)
* references to bundles

It does **NOT contain runtime logic**

## 2. Bundle image

**Purpose: “versioned operator definition unit”**

A bundle contains:

* CSV (ClusterServiceVersion)
* CRDs
* metadata (package manifests)
* RBAC manifests

```text id="2"
bundle = single operator version definition
```

Think of it as:

> 📦 “Installable version snapshot of the operator”

## 3. Controller image

**Purpose: runtime execution**

```text id="3"
Deployment → runs /manager → reconciler logic
```

This is:

* your Go binary
* actual operator behavior

---

## 🔁 How they connect

```text id="4"
Index Image
   ↓ points to
Bundle Image (v0.0.3)
   ↓ defines
CSV
   ↓ references
Controller Image (runtime container)
```

## 🧠 Simple mental model

| Component  | Role            | Analogy                         |
| ---------- | --------------- | ------------------------------- |
| Index      | Catalog         | “App Store listing page”        |
| Bundle     | Version package | “Installer package (.rpm/.deb)” |
| Controller | Runtime app     | “Installed software binary”     |

---

## 🔁 Additional Demos

There is another demo in the folder `operator-with-dependencies` that demonstrates how to add prerequisite operators as dependencies.

Also, the folder `paid-operator` contains a demonstration of how to implement a subscription model for your operator. There are many ways to achieve this, but we demonstrated a simple approach.









