package controller

import (
	"context"
	"encoding/json"
	"strings"

	bmh_v1alpha1 "github.com/metal3-io/baremetal-operator/pkg/apis/metal3/v1alpha1"
	"github.com/openshift/assisted-service/internal/bminventory"
	"github.com/openshift/assisted-service/models"

	// cachev1alpha1 "github.com/example-inc/memcached-operator/pkg/apis/cache/v1alpha1"
	// corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	// metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	// "k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"

	// "sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	logrus "github.com/sirupsen/logrus"
	"sigs.k8s.io/controller-runtime/pkg/handler"

	//logf "sigs.k8s.io/controller-runtime/pkg/log"
	"github.com/thoas/go-funk"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

/*
var log = logf.Log.WithName("controller_baremetalhost")

func init() {
	logrus.Infof("YEV init function of controller")
	// AddToManagerFuncs is a list of functions to create controllers and add them to a manager.
	AddToManagerFuncs = append(AddToManagerFuncs, Add)
}

// AddToManagerFuncs is a list of functions to add all Controllers to the Manager
var AddToManagerFuncs []func(manager.Manager) error

// AddToManager adds all Controllers to the Manager
func AddToManager(m manager.Manager) error {
	logrus.Infof("YEV add to manager")
	for _, f := range AddToManagerFuncs {
		if err := f(m); err != nil {
			return err
		}
	}
	return nil
}

// Add creates a new BareMetalHost Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}
*/

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager, ocpClusterAPI bminventory.OCPClusterAPI) reconcile.Reconciler {
	return &ReconcileBareMetalHost{client: mgr.GetClient(), scheme: mgr.GetScheme(), ocpClusterAPI: ocpClusterAPI}
}

func AddWithBMInventory(mgr manager.Manager, ocpClusterAPI bminventory.OCPClusterAPI) error {
	return add(mgr, newReconciler(mgr, ocpClusterAPI))
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	logrus.Infof("YEV - Start add")
	// Create a new controller
	c, err := controller.New("baremetalhost-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	logrus.Info("Start watch")
	// Watch for changes to primary resource BareMetalHost
	err = c.Watch(&source.Kind{Type: &bmh_v1alpha1.BareMetalHost{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	// TODO(user): Modify this to be the types you create that are owned by the primary resource
	// Watch for changes to secondary resource Pods and requeue the owner BareMetalHost
	/*
	   err = c.Watch(&source.Kind{Type: &corev1.Pod{}}, &handler.EnqueueRequestForOwner{
	           IsController: true,
	           OwnerType:    &cachev1alpha1.BareMetalHost{},
	   })
	   if err != nil {
	           return err
	   }
	*/

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
	//reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	//reqLogger.Info("Reconciling BareMetalHost")

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

	hosts, err := r.ocpClusterAPI.GetOCPClusterHosts()
	if err != nil {
		logrus.Warnf("Failed to get hosts from AI cluster, err %s", err)
		return reconcile.Result{}, err
	}

	if r.newBMH(bmh, hosts) {
		logrus.Infof("Detected new BMH creation")
		r.triggerISOCreation()
	} else {
		logrus.Infof("Already known BMH host, currently do nothing")
	}

	logrus.Infof("Reconciling BMH: %s, num hosts %d", request.NamespacedName, len(hosts))
	return reconcile.Result{}, nil
}

func (r *ReconcileBareMetalHost) newBMH(bmh *bmh_v1alpha1.BareMetalHost, hosts []*models.Host) bool {
	if bmh.Status.HardwareDetails == nil ||
		len(bmh.Status.HardwareDetails.NIC) == 0 {
		return true
	}
	for _, nic := range bmh.Status.HardwareDetails.NIC {
		if r.ipInHosts(nic.IP, hosts) {
			return false
		}
	}
	return true
}

func (r *ReconcileBareMetalHost) ipInHosts(ip string, hosts []*models.Host) bool {
	for _, host := range hosts {
		if r.ipInHost(ip, host) {
			return true
		}
	}
	return false
}

func (r *ReconcileBareMetalHost) ipInHost(ip string, host *models.Host) bool {
	inventory := models.Inventory{}
	err := json.Unmarshal([]byte(host.Inventory), &inventory)
	if err != nil {
		return false
	}
	for _, nic := range inventory.Interfaces {
		compFunc := func(s string) bool {
			return strings.Contains(s, ip)
		}
		_, res := funk.FindString(nic.IPV4Addresses, compFunc)
		if res {
			return true
		}
		_, res = funk.FindString(nic.IPV6Addresses, compFunc)
		if res {
			return true
		}
	}
	return false
}

func (r *ReconcileBareMetalHost) triggerISOCreation() {
}
