
# 📦 Operator Dependencies (OLM)

To Add Dependency to our Operator, all we need to add this depdencies.yaml file where we declare any dependencies on other Operators or Kubernetes APIs to ensure correct runtime behavior.

---

# 🧠 What are Dependencies?

Dependencies define **external requirements** that must exist in the cluster before this operator can function.

There are two types:

| Type          | Meaning                                         |
| ------------- | ----------------------------------------------- |
| `olm.package` | Requires another Operator                       |
| `olm.gvk`     | Requires a Kubernetes API (CRD / resource type) |

---

# 📄 Where dependencies are defined

Dependencies are declared in:

```text
bundle/metadata/dependencies.yaml
```

Example structure:

```yaml
dependencies:
  - type: olm.package
    value:
      packageName: openshift-gitops-operator
      channel: stable

  - type: olm.gvk
    value:
      group: argoproj.io
      version: v1beta1
      kind: Application
```

---

# 🔗 1. Operator Dependencies (olm.package)

Used when your operator depends on another installed Operator.

## Example

```yaml
- type: olm.package
  value:
    packageName: openshift-gitops-operator
    channel: stable
```

### Notes

* `packageName`: name of the Operator in OLM catalog
* `channel`: recommended upgrade stream (preferred over version pinning)
* Ensures GitOps Operator is available before installation continues

---

## ⚠️ Alternative (strict version pinning)

```yaml
version: "1.14.4"
```

Use this only when you require a fixed, tested version (certified version).

---

# 🧩 2. API Dependencies (olm.gvk)

Used when your operator depends on Kubernetes APIs or CRDs.

## Example

```yaml
- type: olm.gvk
  value:
    group: argoproj.io
    version: v1beta1
    kind: Application
```

### Notes

* Ensures required CRDs exist in the cluster
* Must match **exact API version available in the target cluster**
* Does NOT install CRDs automatically

---


If you want, I can also add a **diagram section (OLM install flow + dependency resolution)** or convert this into a polished GitHub README with badges and architecture diagram.
