package controller

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/yaml"

	"k8s.io/client-go/tools/record"
	demov1alpha1 "osa.ora/demo-operator/api/v1alpha1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

// ---------------- catalog ----------------

type Catalog struct {
	BaseDemoURL string      `yaml:"baseDemoUrl"`
	Demos       []DemoEntry `yaml:"demos"`
}

type DemoEntry struct {
	Name        string   `yaml:"name"`
	DisplayName string   `yaml:"displayName"`
	Description string   `yaml:"description"`
	Namespace   string   `yaml:"namespace"`
	Manifests   []string `yaml:"manifests"`
}

const catalogURL = "https://raw.githubusercontent.com/osa-ora/ocp-demo-catalog/main/index.yaml"

// ---------------- cached catalog ----------------
type cachedCatalog struct {
	data      Catalog
	fetchedAt time.Time
}

// caching
var (
	catalogCache cachedCatalog
	cacheMutex   sync.RWMutex
	cacheTTL     = 5 * time.Minute
)

type DemoRequestReconciler struct {
	client.Client
	Scheme     *runtime.Scheme
	Recorder   record.EventRecorder
	HTTPClient *http.Client
}

// ---------------- reconcile ----------------

func (r *DemoRequestReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	var demoReq demov1alpha1.DemoRequest
	if err := r.Get(ctx, req.NamespacedName, &demoReq); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// ---------------- IDENTITY GUARD (idempotency) ----------------
	if demoReq.Status.Phase == "Ready" &&
		len(demoReq.Status.Resources) > 0 {
		log.Info("Demo already ready, skipping reconciliation")
		return ctrl.Result{}, nil
	}

	log.Info("Received DemoRequest", "demo", demoReq.Spec.DemoName)
	//fire event ...
	r.emitEvent(
		&demoReq,
		corev1.EventTypeNormal,
		"Reconciling",
		"Starting reconciliation for demo "+demoReq.Spec.DemoName,
	)

	// ---------------- STATUS: APPLYING ----------------
	if err := r.patchStatus(ctx, &demoReq, func(s *demov1alpha1.DemoRequestStatus) {
		s.Phase = "Applying"
		s.Message = "starting reconciliation"
	}); err != nil {
		return ctrl.Result{}, err
	}

	// ---------------- FETCH CATALOG (CACHED) ----------------
	catalog, err := r.getCatalog()
	if err != nil {
		//fire event first
		r.emitEvent(
			&demoReq,
			corev1.EventTypeWarning,
			"CatalogFetchFailed",
			err.Error(),
		)
		_ = r.patchStatus(ctx, &demoReq, func(s *demov1alpha1.DemoRequestStatus) {
			s.Phase = "Retrying"
			s.Message = err.Error()
		})
		return ctrl.Result{RequeueAfter: 10 * time.Second}, err
	}

	// ---------------- FIND DEMO ----------------
	var selected *DemoEntry
	for i := range catalog.Demos {
		if catalog.Demos[i].Name == demoReq.Spec.DemoName {
			selected = &catalog.Demos[i]
			break
		}
	}

	if selected == nil {
		//fire event first
		r.emitEvent(
			&demoReq,
			corev1.EventTypeWarning,
			"DemoNotFound",
			fmt.Sprintf("demo %s not found in catalog", demoReq.Spec.DemoName),
		)
		_ = r.patchStatus(ctx, &demoReq, func(s *demov1alpha1.DemoRequestStatus) {
			s.Phase = "Failed"
			s.Message = "demo not found"
		})
		return ctrl.Result{}, nil
	}

	log.Info("Selected demo",
		"name", selected.Name,
		"manifests", selected.Manifests,
		"baseURL", catalog.BaseDemoURL)
	//fire event now
	r.emitEvent(
		&demoReq,
		corev1.EventTypeNormal,
		"DemoSelected",
		selected.Name,
	)
	// ---------------- RESOLVE NAMESPACE ----------------
	targetNamespace := demoReq.Spec.Namespace
	if targetNamespace == "" {
		targetNamespace = req.Namespace
	}

	if targetNamespace == "" {
		//fire event first
		r.emitEvent(
			&demoReq,
			corev1.EventTypeWarning,
			"InvalidNamespace",
			"namespace is required but not provided",
		)
		_ = r.patchStatus(ctx, &demoReq, func(s *demov1alpha1.DemoRequestStatus) {
			s.Phase = "Failed"
			s.Message = "namespace must be defined"
		})
		return ctrl.Result{}, fmt.Errorf("missing namespace")
	}

	// ---------------- ENSURE NAMESPACE ----------------
	if err := r.ensureNamespace(ctx, targetNamespace); err != nil {
		//fire event first
		r.emitEvent(
			&demoReq,
			corev1.EventTypeWarning,
			"NamespaceEnsureFailed",
			err.Error(),
		)
		_ = r.patchStatus(ctx, &demoReq, func(s *demov1alpha1.DemoRequestStatus) {
			s.Phase = "Retrying"
			s.Message = err.Error()
		})
		return ctrl.Result{RequeueAfter: 5 * time.Second}, err
	}

	// ---------------- APPLY MANIFESTS ----------------
	for _, file := range selected.Manifests {
		data, err := r.fetchManifest(
			catalog.BaseDemoURL,
			selected.Name,
			file,
		)
		if err != nil {
			//fire event first
			r.emitEvent(
				&demoReq,
				corev1.EventTypeWarning,
				"ManifestFetchFailed",
				fmt.Sprintf("file=%s error=%s", file, err.Error()),
			)
			_ = r.patchStatus(ctx, &demoReq, func(s *demov1alpha1.DemoRequestStatus) {
				s.Phase = "Retrying"
				s.Message = err.Error()
			})
			return ctrl.Result{RequeueAfter: 10 * time.Second}, err
		}

		if err := r.applyManifest(ctx, data, targetNamespace, &demoReq); err != nil {
			//fire event first
			r.emitEvent(
				&demoReq,
				corev1.EventTypeWarning,
				"ManifestApplyFailed",
				err.Error(),
			)
			_ = r.patchStatus(ctx, &demoReq, func(s *demov1alpha1.DemoRequestStatus) {
				s.Phase = "Retrying"
				s.Message = err.Error()
			})
			return ctrl.Result{RequeueAfter: 10 * time.Second}, err
		}
	}

	// ---------------- READY ----------------
	if err := r.patchStatus(ctx, &demoReq, func(s *demov1alpha1.DemoRequestStatus) {
		s.Phase = "Ready"
		s.Message = "Demo is now Ready!"
	}); err != nil {
		return ctrl.Result{}, err
	}

	log.Info("Demo deployed", "demo", selected.Name)
	r.emitEvent(
		&demoReq,
		corev1.EventTypeNormal,
		"Reconciled",
		"Demo deployed successfully!",
	)
	return ctrl.Result{}, nil
}

// ---------------- status patch helper ----------------

func (r *DemoRequestReconciler) patchStatus(
	ctx context.Context,
	obj *demov1alpha1.DemoRequest,
	mutate func(*demov1alpha1.DemoRequestStatus),
) error {

	latest := &demov1alpha1.DemoRequest{}

	if err := r.Get(ctx, client.ObjectKeyFromObject(obj), latest); err != nil {
		return err
	}

	mutate(&latest.Status)

	return r.Status().Update(ctx, latest)
}

// ---------------- catalog cache ----------------
func (r *DemoRequestReconciler) getCatalog() (Catalog, error) {
	cacheMutex.RLock()

	cached := catalogCache.data
	fresh := time.Since(catalogCache.fetchedAt) < cacheTTL

	if fresh {
		cacheMutex.RUnlock()
		return cached, nil
	}

	// keep snapshot of last known good before unlocking
	hasCache := !catalogCache.fetchedAt.IsZero()

	cacheMutex.RUnlock()

	cacheMutex.Lock()
	defer cacheMutex.Unlock()

	// double-check (another goroutine may have refreshed)
	if time.Since(catalogCache.fetchedAt) < cacheTTL {
		return catalogCache.data, nil
	}

	clientHTTP := r.HTTPClient
	if clientHTTP == nil {
		clientHTTP = &http.Client{Timeout: 10 * time.Second}
	}

	resp, err := clientHTTP.Get(catalogURL)
	if err != nil {
		// 🔥 FALLBACK: return stale cache instead of failing
		if hasCache {
			return catalogCache.data, nil
		}
		return Catalog{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// 🔥 FALLBACK on bad HTTP response
		if hasCache {
			return catalogCache.data, nil
		}
		return Catalog{}, fmt.Errorf("catalog fetch failed with status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		if hasCache {
			return catalogCache.data, nil
		}
		return Catalog{}, err
	}

	var catalog Catalog
	if err := yaml.Unmarshal(body, &catalog); err != nil {
		if hasCache {
			return catalogCache.data, nil
		}
		return Catalog{}, err
	}

	// update cache only on success
	catalogCache.data = catalog
	catalogCache.fetchedAt = time.Now()

	return catalog, nil
}

// ---------------- APPLY ----------------

func (r *DemoRequestReconciler) applyManifest(
	ctx context.Context,
	data []byte,
	namespace string,
	instance *demov1alpha1.DemoRequest,
) error {

	decoder := yaml.NewYAMLOrJSONDecoder(
		bytes.NewReader(data),
		4096,
	)

	for {
		raw := runtime.RawExtension{}

		err := decoder.Decode(&raw)

		logf.FromContext(ctx).Info(
			"Raw manifest chunk",
			"raw",
			string(raw.Raw),
		)

		if err != nil {
			if err == io.EOF {
				break
			}

			return fmt.Errorf("decode manifest: %w", err)
		}

		if len(raw.Raw) == 0 {
			continue
		}

		obj := &unstructured.Unstructured{}

		if err := json.Unmarshal(raw.Raw, &obj.Object); err != nil {
			return err
		}

		logf.FromContext(ctx).Info(
			"Decoded object",
			"kind",
			obj.GetKind(),
			"name",
			obj.GetName(),
		)

		if obj.GetKind() == "" {
			continue
		}

		// ---------------- namespace handling ----------------
		if obj.GetNamespace() == "" &&
			!isClusterScoped(obj.GetKind()) {
			obj.SetNamespace(namespace)
		}

		// ---------------- labels ----------------
		labels := obj.GetLabels()

		if labels == nil {
			labels = map[string]string{}
		}

		labels["demo.osa.ora/instance"] = instance.Name
		labels["demo.osa.ora/instance-id"] = string(instance.UID)
		labels["demo.osa.ora/demo"] = instance.Spec.DemoName
		labels["demo.osa.ora/managed-by"] = "osaora-demo-operator"

		obj.SetLabels(labels)

		// ---------------- owner reference ----------------
		if obj.GetNamespace() != "" &&
			!isClusterScoped(obj.GetKind()) {

			obj.SetOwnerReferences([]metav1.OwnerReference{
				{
					APIVersion:         instance.APIVersion,
					Kind:               instance.Kind,
					Name:               instance.Name,
					UID:                instance.UID,
					Controller:         ptrBool(true),
					BlockOwnerDeletion: ptrBool(true),
				},
			})
		}

		// ---------------- APPLY ----------------
		if err := r.Patch(
			ctx,
			obj,
			client.Apply,
			client.ForceOwnership,
			client.FieldOwner("osaora-demo-operator"),
		); err != nil {

			return fmt.Errorf(
				"apply failed kind=%s namespace=%s name=%s: %w",
				obj.GetKind(),
				obj.GetNamespace(),
				obj.GetName(),
				err,
			)
		}

		latest := &demov1alpha1.DemoRequest{}

		if err := r.Get(
			ctx,
			client.ObjectKeyFromObject(instance),
			latest,
		); err == nil {

			updated := latest.DeepCopy()

			ref := corev1.ObjectReference{
				APIVersion: obj.GetAPIVersion(),
				Kind:       obj.GetKind(),
				Name:       obj.GetName(),
				Namespace:  obj.GetNamespace(),
			}

			exists := false

			for _, res := range updated.Status.Resources {
				if res.APIVersion == ref.APIVersion &&
					res.Kind == ref.Kind &&
					res.Name == ref.Name &&
					res.Namespace == ref.Namespace {

					exists = true
					break
				}
			}

			if !exists {
				updated.Status.Resources = append(
					updated.Status.Resources,
					ref,
				)
			}

			if err := r.Status().Update(ctx, updated); err != nil {
				logf.FromContext(ctx).Error(
					err,
					"failed updating resource tracking",
				)
			}
		}
	}

	return nil
}

// ---------------- helpers ----------------

func isClusterScoped(kind string) bool {
	switch kind {
	case "ClusterRole",
		"ClusterRoleBinding",
		"CustomResourceDefinition",
		"Namespace":
		return true

	default:
		return false
	}
}

func ptrBool(b bool) *bool {
	return &b
}

// ---------------- HTTP manifest fetch ----------------

func (r *DemoRequestReconciler) fetchManifest(
	baseURL,
	demoName,
	file string,
) ([]byte, error) {

	fullURL := strings.TrimRight(baseURL, "/") +
		"/" +
		strings.TrimLeft(demoName, "/") +
		"/" +
		strings.TrimLeft(file, "/")

	clientHTTP := r.HTTPClient

	if clientHTTP == nil {
		clientHTTP = &http.Client{
			Timeout: 10 * time.Second,
		}
	}

	resp, err := clientHTTP.Get(fullURL)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf(
			"manifest fetch failed %s: status %d",
			fullURL,
			resp.StatusCode,
		)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if len(body) == 0 {
		return nil, fmt.Errorf("empty manifest: %s", fullURL)
	}

	return body, nil
}

// ---------------- namespace ----------------

func (r *DemoRequestReconciler) ensureNamespace(
	ctx context.Context,
	name string,
) error {

	ns := &corev1.Namespace{}

	err := r.Get(
		ctx,
		client.ObjectKey{Name: name},
		ns,
	)

	if err == nil {
		return nil
	}

	if !apierrors.IsNotFound(err) {
		return err
	}

	err = r.Create(ctx, &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	})

	if apierrors.IsAlreadyExists(err) {
		return nil
	}

	// IMPORTANT FIX:
	// ensure namespace is actually visible before continuing
	// prevents "namespace not found" during immediate apply

	for i := 0; i < 20; i++ {
		err = r.Get(
			ctx,
			client.ObjectKey{Name: name},
			ns,
		)

		if err == nil {
			return nil
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(200 * time.Millisecond):
		}
	}

	return fmt.Errorf(
		"namespace %s not ready after creation",
		name,
	)
}

// event recorder
func (r *DemoRequestReconciler) emitEvent(
	obj *demov1alpha1.DemoRequest,
	eventType, reason, message string,
) {
	if r.Recorder == nil || obj == nil {
		return
	}

	r.Recorder.Event(
		obj,
		eventType,
		reason,
		message,
	)
}

// ---------------- setup ----------------

func (r *DemoRequestReconciler) SetupWithManager(
	mgr ctrl.Manager,
) error {

	return ctrl.NewControllerManagedBy(mgr).
		For(&demov1alpha1.DemoRequest{}).
		Named("demorequest").
		Complete(r)
}
