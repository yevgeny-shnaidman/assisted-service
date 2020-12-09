package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	bmh_v1alpha1 "github.com/metal3-io/baremetal-operator/pkg/apis/metal3/v1alpha1"
	"github.com/openshift/assisted-service/internal/bminventory"
	"github.com/openshift/assisted-service/models"

	"github.com/go-openapi/swag"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	logrus "github.com/sirupsen/logrus"
	"sigs.k8s.io/controller-runtime/pkg/handler"

	"github.com/thoas/go-funk"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var discoveryStatuses = []string{models.HostStatusDiscovering,
	models.HostStatusInsufficient,
	models.HostStatusDisconnected,
	models.HostStatusPendingForInput,
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager, ocpClusterAPI bminventory.OCPClusterAPI) reconcile.Reconciler {
	return &ReconcileBareMetalHost{client: mgr.GetClient(), scheme: mgr.GetScheme(), ocpClusterAPI: ocpClusterAPI}
}

func AddWithBMInventory(mgr manager.Manager, ocpClusterAPI bminventory.OCPClusterAPI) error {
	return addBMHController(mgr, newReconciler(mgr, ocpClusterAPI))
}

func isAssistedinstallerBMH(meta metav1.Object) bool {
	labels := meta.GetLabels()
	if _, found := labels["assisted-installer-bmh"]; found {
		return true
	}
	return false
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func addBMHController(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("baremetalhost-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	filterPredicate := predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool {
			return isAssistedinstallerBMH(e.Meta)
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			return isAssistedinstallerBMH(e.Meta)
		},
		UpdateFunc: func(e event.UpdateEvent) bool {
			return isAssistedinstallerBMH(e.MetaNew)
		},
		GenericFunc: func(e event.GenericEvent) bool {
			return isAssistedinstallerBMH(e.Meta)
		},
	}

	logrus.Info("Start watch")
	// Watch for changes to primary resource BareMetalHost
	err = c.Watch(&source.Kind{Type: &bmh_v1alpha1.BareMetalHost{}}, &handler.EnqueueRequestForObject{}, filterPredicate)
	if err != nil {
		return err
	}

	return nil
}

// blank assignment to verify that ReconcileBareMetalHost implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileBareMetalHost{}

// ReconcileBareMetalHost reconciles a BareMetalHost object
type ReconcileBareMetalHost struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client        client.Client
	scheme        *runtime.Scheme
	ocpClusterAPI bminventory.OCPClusterAPI
}

func (r *ReconcileBareMetalHost) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	logrus.Infof("Reconciling BMH: %s", request.NamespacedName)
	// Fetch the BareMetalHost instance
	bmh := &bmh_v1alpha1.BareMetalHost{}
	err := r.client.Get(context.TODO(), request.NamespacedName, bmh)
	if err != nil {
		if errors.IsNotFound(err) {
			/*
				BMH was probably deleted, currently we do nothing
			*/
			logrus.Infof("BMG %s was deleted", request.NamespacedName)
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	if r.isNewBMH(bmh) {
		logrus.Infof("Detected new BMH creation")
		err = r.triggerISOCreation()
		if err != nil {
			logrus.Errorf("Failed to create ISO for booting the node, error %s", err)
			return reconcile.Result{}, err
		}
		err = r.setISOUrl(bmh)
		if err != nil {
			logrus.Errorf("Failed to set create ISO url in the BMH, error %s", err)
			return reconcile.Result{}, err
		}
	} else {
		// check if install label was added
		installFlag, host, err := r.isStartInstall(bmh)
		if err != nil {
			return reconcile.Result{}, err
		}
		if !installFlag {
			return reconcile.Result{}, nil
		}
		err = r.startInstallation(host)
		if err != nil {
			logrus.Errorf("failed to issue install command for bmh")
			return reconcile.Result{}, err
		}
	}

	return reconcile.Result{}, nil
}

func (r *ReconcileBareMetalHost) isNewBMH(bmh *bmh_v1alpha1.BareMetalHost) bool {
	// once new BMH CR is ready , move to LiveImage
	return bmh.Spec.Image == nil
	//retur bmh.Spec.LiveImage.Url != nil {
}

func (r *ReconcileBareMetalHost) setISOUrl(bmh *bmh_v1alpha1.BareMetalHost) error {
	url, err := r.ocpClusterAPI.GetISOHttpURL()
	if err != nil {
		logrus.Errorf("Failed to get discovery ISO url, error %s", err)
		return err
	}
	// once new BMH CR is ready , move to LiveImage
	bmh.Spec.Image = &bmh_v1alpha1.Image{URL: url}
	//bmh.Spec.LiveImage.Url = url
	err = r.client.Update(context.TODO(), bmh)
	if err != nil {
		logrus.Errorf("Failed to update the BMH spec, error %s", err)
		return err
	}
	return nil
}

func (r *ReconcileBareMetalHost) isStartInstall(bmh *bmh_v1alpha1.BareMetalHost) (bool, *models.Host, error) {
	labels := bmh.Labels
	if _, found := labels["assisted-installer-bmh-install"]; !found {
		return false, nil, nil
	}

	host, err := r.findBMHHost(bmh)
	if err != nil || host == nil {
		logrus.Errorf("Failed to find host for bmh")
		return false, nil, fmt.Errorf("Failed to find AI host for bmh")
	}
	if funk.Contains(discoveryStatuses, swag.StringValue(host.Status)) {
		return false, nil, fmt.Errorf("host in status %s, need to wait until in status known", swag.StringValue(host.Status))
	}
	if swag.StringValue(host.Status) == models.HostStatusKnown {
		return true, host, nil
	}
	return false, nil, nil
}

func (r *ReconcileBareMetalHost) startInstallation(h *models.Host) error {
	return r.ocpClusterAPI.InstallOCPHost(h)
}

func (r *ReconcileBareMetalHost) findBMHHost(bmh *bmh_v1alpha1.BareMetalHost) (*models.Host, error) {
	hosts, err := r.ocpClusterAPI.GetOCPClusterHosts()
	if err != nil {
		return nil, err
	}

	for _, host := range hosts {
		inventory := models.Inventory{}
		err := json.Unmarshal([]byte(host.Inventory), &inventory)
		if err != nil {
			continue
		}
		if strings.Contains(bmh.Spec.BMC.Address, inventory.BmcAddress) {
			return host, nil
		}
	}
	return nil, nil
}

func (r *ReconcileBareMetalHost) triggerISOCreation() error {
	return r.ocpClusterAPI.GenerateOCPClusterISO()
}
