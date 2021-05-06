package mirrorregistries

import (
	"context"
	"encoding/base64"
	"fmt"
	"io/ioutil"

	"github.com/go-openapi/swag"
	"github.com/openshift/assisted-service/internal/common"

	// "github.com/openshift/assisted-service/internal/manifests"
	"github.com/openshift/assisted-service/models"
	"github.com/openshift/assisted-service/restapi"
	operations "github.com/openshift/assisted-service/restapi/operations/manifests"
	"github.com/pelletier/go-toml"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

//go:generate mockgen -source=generator.go -package=mirrorregistries -destination=mock_generator.go
type MirrorRegistriesConfigBuilder interface {
	IsMirrorRegistriesConfigured() bool
	GetMirrorCA() ([]byte, error)
	GetMirrorRegistries() ([]byte, error)
	ExtractLocationMirrorDataFromRegistries() ([]RegistriesConf, error)
	AddRegistryConfManifests(ctx context.Context, log logrus.FieldLogger, c *common.Cluster) error
}

type MirrorRegistriesManifestsGenerator interface {
	AddRegistryConfManifests(ctx context.Context, log logrus.FieldLogger, c *common.Cluster) error
}

type mirrorRegistriesConfigBuilder struct {
	manifestsApi restapi.ManifestsAPI
}

func New(manifestsApi restapi.ManifestsAPI) MirrorRegistriesConfigBuilder {
	return &mirrorRegistriesConfigBuilder{manifestsApi: manifestsApi}
}

type RegistriesConf struct {
	Location string
	Mirror   string
}

const registryConfManifest = `
apiVersion: machineconfiguration.openshift.io/v1
kind: MachineConfig
metadata:
  labels:
    machineconfiguration.openshift.io/role: %s
  name: %s-registry-conf
spec:
  config:
    ignition:
      config: {}
      security:
        tls: {}
      timeouts: {}
      version: 2.2.0
    networkd: {}
    passwd: {}
    storage:
      files:
      - contents:
          source: data:text/plain;charset=utf-8;base64,%s
          verification: {}
        filesystem: root
        mode: 493
        path: /etc/containers/registry.conf
  osImageURL: ""
`

func (m *mirrorRegistriesConfigBuilder) IsMirrorRegistriesConfigured() bool {
	_, err := m.GetMirrorCA()
	if err != nil {
		return false
	}
	_, err = m.GetMirrorRegistries()
	return err == nil
}

// return error if the path is actually an empty dir, which will indicate that
// the mirror registries are not configured.
// empty dir is due to the way we mao configmap in the assisted-service pod
func (m *mirrorRegistriesConfigBuilder) GetMirrorCA() ([]byte, error) {
	return readFile(common.MirrorRegistriesCertificatePath)
}

// returns error if the file is not present, which will also indicate that
// mirror registries are not confgiured
func (m *mirrorRegistriesConfigBuilder) GetMirrorRegistries() ([]byte, error) {
	return readFile(common.MirrorRegistriesConfigPath)
}

func (m *mirrorRegistriesConfigBuilder) ExtractLocationMirrorDataFromRegistries() ([]RegistriesConf, error) {
	contents, err := m.GetMirrorRegistries()
	if err != nil {
		return nil, err
	}
	return extractLocationMirrorDataFromRegistries(string(contents))
}

func extractLocationMirrorDataFromRegistries(registriesConfToml string) ([]RegistriesConf, error) {
	tomlTree, err := toml.Load(registriesConfToml)
	if err != nil {
		return nil, err
	}

	registriesTree, ok := tomlTree.Get("registry").([]*toml.Tree)
	if !ok {
		return nil, fmt.Errorf("Failed to cast registry key to toml Tree")
	}
	registriesConfList := make([]RegistriesConf, len(registriesTree))
	for i, registryTree := range registriesTree {
		location, ok := registryTree.Get("location").(string)
		if !ok {
			return nil, fmt.Errorf("Failed to cast location key to string")
		}
		mirrorTree, ok := registryTree.Get("mirror").([]*toml.Tree)
		if !ok {
			return nil, fmt.Errorf("Failed to cast mirror key to toml Tree")
		}
		mirror, ok := mirrorTree[0].Get("location").(string)
		if !ok {
			return nil, fmt.Errorf("Failed to cast mirror location key to string")
		}
		registriesConfList[i] = RegistriesConf{Location: location, Mirror: mirror}
	}

	return registriesConfList, nil
}

func readFile(filePath string) ([]byte, error) {
	return ioutil.ReadFile(filePath)
}

func createRegistryConfManifestContext(role string, contents []byte) string {
	return fmt.Sprintf(registryConfManifest, role, role, base64.StdEncoding.EncodeToString(contents))
}

func (m *mirrorRegistriesConfigBuilder) AddRegistryConfManifests(ctx context.Context, log logrus.FieldLogger, c *common.Cluster) error {
	if !m.IsMirrorRegistriesConfigured() {
		log.Infof("No mirror registry manifests to generate")
		return nil
	}

	registryConfContents, err := m.GetMirrorRegistries()
	if err != nil {
		log.WithError(err).Errorf("Failed to read registry conf file")
		return err
	}

	for _, role := range []string{"master", "worker"} {
		fileName := fmt.Sprintf("%s-registry-conf.yaml", role)
		if err := m.createManifests(ctx, c, fileName, []byte(createRegistryConfManifestContext(role, registryConfContents))); err != nil {
			log.WithError(err).Errorf("Failed to create registry conf manifest for role %s", role)
			return err
		}
	}
	return nil
}

func (m *mirrorRegistriesConfigBuilder) createManifests(ctx context.Context, cluster *common.Cluster, filename string, content []byte) error {
	// all relevant logs of creating manifest will be inside CreateClusterManifest
	response := m.manifestsApi.CreateClusterManifest(ctx, operations.CreateClusterManifestParams{
		ClusterID: *cluster.ID,
		CreateManifestParams: &models.CreateManifestParams{
			Content:  swag.String(base64.StdEncoding.EncodeToString(content)),
			FileName: &filename,
			Folder:   swag.String(models.ManifestFolderOpenshift),
		},
	})

	if _, ok := response.(*operations.CreateClusterManifestCreated); !ok {
		if apiErr, ok := response.(*common.ApiErrorResponse); ok {
			return errors.Wrapf(apiErr, "Failed to create manifest %s", filename)
		}
		return errors.Errorf("Failed to create manifest %s", filename)
	}
	return nil
}
