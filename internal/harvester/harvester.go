package harvester

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/harvester/harvester/pkg/apis/harvesterhci.io/v1beta1"
	harvclient "github.com/harvester/harvester/pkg/generated/clientset/versioned"
	randGen "github.com/matoous/go-nanoid/v2"
	"github.com/minio/pkg/wildcard"
	"github.com/rancher/dartboard/internal/actions"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
	v1 "k8s.io/api/core/v1"
	k8sresource "k8s.io/apimachinery/pkg/api/resource"
	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubeclient "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	VMv1 "kubevirt.io/api/core/v1"
)

const (
	vmAnnotationPVC              = "harvesterhci.io/volumeClaimTemplates"
	vmAnnotationNetworkIps       = "networks.harvesterhci.io/ips"
	defaultDiskSize              = "10Gi"
	defaultMemSize               = "1Gi"
	defaultNbCPUCores            = 1
	defaultNamespace             = "default"
	ubuntuDefaultImage           = "https://cloud-images.ubuntu.com/minimal/daily/focal/current/focal-minimal-cloudimg-amd64.img"
	defaultCloudInitUserData     = "#cloud-config\npackages:\n  - qemu-guest-agent\nruncmd:\n  - [ systemctl, daemon-reload ]\n  - [ systemctl, enable, qemu-guest-agent.service ]\n  - [ systemctl, start, --no-block, qemu-guest-agent.service ]"
	defaultCloudInitNetworkData  = "version: 2\nrenderer: networkd\nethernets:\n  enp1s0:\n    dhcp4: true"
	defaultCloudInitCmPrefix     = "default-ubuntu-"
	defaultOverCommitSettingName = "overcommit-config"
	RemovedPVCsAnnotationKey     = "harvesterhci.io/removedPersistentVolumeClaims"
)

// VirtualMachineData type is a Data Structure that holds information to display for VM
type VirtualMachineData struct {
	State          string
	VirtualMachine VMv1.VirtualMachine
	Name           string
	Node           string
	CPU            uint32
	Memory         string
	IPAddress      string
}

type VMTemplateInput struct {
	Name      string `json:"name,omitempty" yaml:"name,omitempty"`
	Namespace string `json:"namespace,omitempty" yaml:"namespace,omitempty"`
	Version   string `json:"version,omitempty" yaml:"version,omitempty"`
}

type VMNetworkInput struct {
	Name              string `json:"name,omitempty" yaml:"name,omitempty"`
	Namespace         string `json:"namespace,omitempty" yaml:"namespace,omitempty"`
	CloudInitData     []byte `json:"cloud_init_data,omitempty" yaml:"cloud_init_data,omitempty"`
	CloudInitTemplate string `json:"cloud_init_template,omitempty" yaml:"cloud_init_template,omitempty"`
}

type VMUser struct {
	Name              string `json:"name,omitempty" yaml:"name,omitempty"`
	CloudInitData     []byte `json:"cloud_init_data,omitempty" yaml:"cloud_init_data,omitempty"`
	CloudInitTemplate string `json:"cloud_init_template,omitempty" yaml:"cloud_init_template,omitempty"`
}

type VMImage struct {
	ID        string `json:"id,omitempty" yaml:"id,omitempty"`
	Name      string `json:"name,omitempty" yaml:"name,omitempty"`
	Namespace string `json:"namespace,omitempty" yaml:"namespace,omitempty"`
}

type VMSSHKey struct {
	Name      string `json:"name,omitempty" yaml:"name,omitempty"`
	Namespace string `json:"namespace,omitempty" yaml:"namespace,omitempty"`
}

type VMInputs struct {
	Count       int             `json:"count,omitempty" yaml:"count,omitempty"`
	Name        string          `json:"name,omitempty" yaml:"name,omitempty"`
	Namespace   string          `json:"namespace,omitempty" yaml:"namespace,omitempty"`
	Description string          `json:"description,omitempty" yaml:"description,omitempty"`
	Image       VMImage         `json:"image,omitempty" yaml:"image,omitempty"`
	CPUs        int             `json:"cpus,omitempty" yaml:"cpus,omitempty"`
	Memory      int             `json:"memory,omitempty" yaml:"memory,omitempty"`
	DiskSize    string          `json:"disk_size,omitempty" yaml:"disk_size,omitempty"`
	User        VMUser          `json:"user,omitempty" yaml:"user,omitempty"`
	Template    VMTemplateInput `json:"template,omitempty" yaml:"template,omitempty"`
	Network     VMNetworkInput  `json:"network,omitempty" yaml:"network,omitempty"`
	SSHKey      VMSSHKey        `json:"ssh_key,omitempty" yaml:"ssh_key,omitempty"`
}

var overCommitSettingMap map[string]int

// GetHarvesterClient creates a Client for Harvester from Config input
func GetHarvesterClient(kubeconfigPath string) (*harvclient.Clientset, error) {
	clientConfig, err := clientcmd.BuildConfigFromFlags("", kubeconfigPath)

	if err != nil {
		return &harvclient.Clientset{}, err
	}

	return harvclient.NewForConfig(clientConfig)
}

// GetKubeClient creates a Vanilla Kubernetes Client to query the Kubernetes-native API Objects
func GetKubeClient(kubeconfigPath string) (*kubeclient.Clientset, error) {
	clientConfig, err := clientcmd.BuildConfigFromFlags("", kubeconfigPath)

	if err != nil {
		return &kubeclient.Clientset{}, err
	}

	return kubeclient.NewForConfig(clientConfig)
}

// ListVMs lists the VMs available in Harvester
func ListVMs(c *harvclient.Clientset, namespace string) ([]VMv1.VirtualMachine, []VirtualMachineData, error) {
	vmList, err := c.KubevirtV1().VirtualMachines(namespace).List(context.TODO(), k8smetav1.ListOptions{})

	if err != nil {
		return nil, nil, err
	}

	vmiList, err := c.KubevirtV1().VirtualMachineInstances(namespace).List(context.TODO(), k8smetav1.ListOptions{})

	if err != nil {
		return nil, nil, err
	}

	vmiMap := map[string]VMv1.VirtualMachineInstance{}
	for _, vmi := range vmiList.Items {
		vmiMap[vmi.Name] = vmi
	}

	var allVMs []VMv1.VirtualMachine
	var allVMData []VirtualMachineData
	for _, vm := range vmList.Items {
		state := string(vm.Status.PrintableStatus)

		var IP string
		if vmiMap[vm.Name].Status.Interfaces == nil {
			IP = ""
		} else {
			IP = vmiMap[vm.Name].Status.Interfaces[0].IP
		}

		var memory string
		if vm.Spec.Template.Spec.Domain.Resources.Limits.Memory().CmpInt64(int64(0)) == 0 {
			memory = vm.Spec.Template.Spec.Domain.Resources.Requests.Memory().String()
		} else {
			memory = vm.Spec.Template.Spec.Domain.Resources.Limits.Memory().String()
		}

		allVMs = append(allVMs, vm)

		vmData := VirtualMachineData{
			State:          state,
			VirtualMachine: vm,
			Name:           vm.Name,
			Node:           vmiMap[vm.Name].Status.NodeName,
			CPU:            vm.Spec.Template.Spec.Domain.CPU.Cores,
			Memory:         memory,
			IPAddress:      IP,
		}
		allVMData = append(allVMData, vmData)
	}

	return allVMs, allVMData, nil
}

// DeleteVM deletes VMs which name is given in argument
func DeleteVM(c *harvclient.Clientset, namespace, vmName string) error {

	if strings.Contains(vmName, "*") || strings.Contains(vmName, "?") {
		matchingVMs, err := BuildVMListMatchingWildcard(c, namespace, vmName)
		if err != nil {
			return err
		}

		for _, vmExisting := range matchingVMs {

			err := DeleteVMWithPVC(c, &vmExisting, namespace)
			if err != nil {
				return err
			}
		}
	} else {
		vm, err := c.KubevirtV1().VirtualMachines(namespace).Get(context.TODO(), vmName, k8smetav1.GetOptions{})

		if err != nil {
			return fmt.Errorf("no VM with the provided name found")
		}

		err = DeleteVMWithPVC(c, vm, namespace)
		if err != nil {
			return err
		}
	}

	return nil

}

func DeleteVMWithPVC(c *harvclient.Clientset, vmExisting *VMv1.VirtualMachine, namespace string) error {
	vmCopy := vmExisting.DeepCopy()
	var removedPVCs []string
	if vmCopy.Spec.Template != nil {
		for _, vol := range vmCopy.Spec.Template.Spec.Volumes {
			if vol.PersistentVolumeClaim == nil {
				continue
			}

			removedPVCs = append(removedPVCs, vol.PersistentVolumeClaim.ClaimName)

		}
	}

	vmCopy.Annotations[RemovedPVCsAnnotationKey] = strings.Join(removedPVCs, ",")
	_, err := c.KubevirtV1().VirtualMachines(namespace).Update(context.TODO(), vmCopy, k8smetav1.UpdateOptions{})

	if err != nil {
		return fmt.Errorf("error during removal of PVCs in the VM reference, %w", err)
	}

	err = c.KubevirtV1().VirtualMachines(namespace).Delete(context.TODO(), vmCopy.Name, k8smetav1.DeleteOptions{})
	if err != nil {
		return fmt.Errorf("VM named %s could not be deleted successfully: %w", vmCopy.Name, err)
	} else {
		logrus.Infof("VM %s deleted successfully", vmCopy.Name)
	}
	return nil
}

// CreateVM implements the CLI *vm create* command, there are two options, either to create a VM from a Harvester VM template or from a VM image
func CreateVM(c *harvclient.Clientset, k *kubeclient.Clientset, vmInputs *VMInputs) error {
	if vmInputs.Template.Name != "" {
		return createVMFromTemplate(c, k, vmInputs)
	} else {
		return createVMFromImage(c, k, nil, vmInputs)
	}
}

// createVMFromTemplate creates a VM from a VM template provided in the CLI command
func createVMFromTemplate(c *harvclient.Clientset, k *kubeclient.Clientset, vmInputs *VMInputs) error {
	var err error
	// checking if template exists
	templateContent, err := c.HarvesterhciV1beta1().VirtualMachineTemplates(vmInputs.Namespace).Get(context.TODO(), vmInputs.Template.Name, k8smetav1.GetOptions{})

	if err != nil {
		return fmt.Errorf("template %s was not found on the Harvester Cluster in namespace %s", vmInputs.Template.Name, vmInputs.Namespace)
	}

	// Picking the templateVersion
	var templateVersion *v1beta1.VirtualMachineTemplateVersion
	if vmInputs.Template.Version == "" {
		templateVersion, err = c.HarvesterhciV1beta1().VirtualMachineTemplateVersions(vmInputs.Namespace).Get(context.TODO(), vmInputs.Template.Version, k8smetav1.GetOptions{})
		if err != nil {
			return err
		}
		logrus.Debugf("templateVersion found is :%s\n", templateContent.Spec.DefaultVersionID)
		templateVersion.ManagedFields = []k8smetav1.ManagedFieldsEntry{}
		marshalledTemplateVersion, err := json.Marshal(templateVersion)

		if err != nil {
			return err
		}
		logrus.Debugf("template version: %s\n", string(marshalledTemplateVersion))
	} else {
		versionNum, err := strconv.Atoi(vmInputs.Template.Version)
		if err != nil {
			return err
		}
		templateVersion, err = fetchTemplateVersionFromInt(c, vmInputs.Namespace, vmInputs.Template.Name, versionNum)
		if err != nil {
			return err
		}
	}

	templateVersionAnnotation := templateVersion.Spec.VM.ObjectMeta.Annotations[vmAnnotationPVC]
	logrus.Debugf("VM Annotation for PVC (should be JSON format): %s", templateVersionAnnotation)
	var pvcList []v1.PersistentVolumeClaim
	err = json.Unmarshal([]byte(templateVersionAnnotation), &pvcList)

	if err != nil {
		return err
	}

	pvc := pvcList[0]

	vmImageIdWithNamespace := pvc.ObjectMeta.Annotations["harvesterhci.io/imageId"]
	vmInputs.Image.ID = strings.Split(vmImageIdWithNamespace, "/")[1]
	if vmInputs.DiskSize == "" {
		vmInputs.DiskSize = pvc.Spec.Resources.Requests.Storage().String()
		if err != nil {
			return fmt.Errorf("error during setting flag to context: %w", err)
		}
	}

	vmTemplate := templateVersion.Spec.VM.Spec.Template

	err = createVMFromImage(c, k, vmTemplate, vmInputs)

	if err != nil {
		return err
	}

	return nil
}

// fetchTemplateVersionFromInt gets the Template with the right version given the context (containing template name) and the version as an integer
func fetchTemplateVersionFromInt(c *harvclient.Clientset, namespace, templateName string, version int) (*v1beta1.VirtualMachineTemplateVersion, error) {
	templateSelector := "template.harvesterhci.io/templateID=" + templateName

	allTemplateVersions, err := c.HarvesterhciV1beta1().VirtualMachineTemplateVersions(namespace).List(context.TODO(), k8smetav1.ListOptions{
		LabelSelector: templateSelector,
	})

	if err != nil {
		return nil, err
	}

	for _, serverTemplateVersion := range allTemplateVersions.Items {
		if version == serverTemplateVersion.Status.Version {
			return &serverTemplateVersion, nil
		}

	}
	return nil, fmt.Errorf("no VM template named %s with version %d found", templateName, version)
}

// createVMFromImage creates a VM from a VM Image using the CLI command context to get information
func createVMFromImage(c *harvclient.Clientset, k *kubeclient.Clientset, vmTemplate *VMv1.VirtualMachineInstanceTemplateSpec, vmInputs *VMInputs) error {
	if vmInputs.Count == 0 {
		return fmt.Errorf("VM count provided is 0, no VM will be created")
	}
	var err error
	// Checking existence of Image ID and if not, using default ubuntu image.
	var vmImage *v1beta1.VirtualMachineImage
	if vmInputs.Image.ID != "" {
		vmImage, err = c.HarvesterhciV1beta1().VirtualMachineImages(vmInputs.Image.Namespace).Get(context.TODO(), vmInputs.Image.Name, k8smetav1.GetOptions{})
		if err != nil {
			return err
		}
		logrus.Debugf("Found Image with ID %s!", vmInputs.Image.ID)
	} else {
		vmImage, err = SetDefaultVMImage(c, vmInputs)
		if err != nil {
			return err
		}
	}
	storageClassName := vmImage.Status.StorageClassName
	vmNameBase := vmInputs.Name

	vmLabels := map[string]string{
		"harvesterhci.io/creator": "harvester",
	}
	vmiLabels := vmLabels

	// Checking if provided Network exists in Harvester
	_, err = c.K8sCniCncfIoV1().NetworkAttachmentDefinitions(vmInputs.Network.Namespace).Get(context.TODO(), vmInputs.Network.Name, k8smetav1.GetOptions{})

	if err != nil {
		return fmt.Errorf("problem while verifying network existence; %w", err)
	}

	overCommitSetting, err := c.HarvesterhciV1beta1().Settings().Get(context.TODO(), defaultOverCommitSettingName, k8smetav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("encountered issue when querying Harvester for setting %s: %w", defaultOverCommitSettingName, err)
	}

	err = json.Unmarshal([]byte(overCommitSetting.Default), &overCommitSettingMap)
	if err != nil {
		return fmt.Errorf("encountered issue when unmarshaling setting value %s: %w", defaultOverCommitSettingName, err)
	}

	for i := 1; i <= vmInputs.Count; i++ {
		var vmName string
		if vmInputs.Count > 1 {
			vmName = vmNameBase + "-" + fmt.Sprint(i)
		} else {
			vmName = vmNameBase
		}

		vmiLabels["harvesterhci.io/vmName"] = vmName
		vmiLabels["harvesterhci.io/vmNamePrefix"] = vmNameBase
		diskRandomID, err := randGen.New(8)
		if err != nil {
			return err
		}
		pvcName := vmName + "-disk-0-" + diskRandomID

		pvcAnnotation := "[{\"metadata\":{\"name\":\"" + pvcName + "\",\"annotations\":{\"harvesterhci.io/imageId\":\"" + vmInputs.Image.Namespace + "/" + vmInputs.Image.ID + "\"}},\"spec\":{\"accessModes\":[\"ReadWriteMany\"],\"resources\":{\"requests\":{\"storage\":\"" + vmInputs.DiskSize + "\"}},\"volumeMode\":\"Block\",\"storageClassName\":\"" + storageClassName + "\"}}]"

		if vmTemplate == nil {

			vmTemplate, err = BuildVMTemplate(c, k, pvcName, vmiLabels, vmInputs)
			if err != nil {
				return err
			}
		} else {
			vmTemplate.Spec.Volumes[0].PersistentVolumeClaim.ClaimName = pvcName

			if vmTemplate.ObjectMeta.Labels == nil {
				vmTemplate.ObjectMeta.Labels = make(map[string]string)
			}

			vmTemplate.ObjectMeta.Labels["harvesterhci.io/vmNamePrefix"] = vmNameBase
			vmTemplate.Spec.Affinity = &v1.Affinity{
				PodAntiAffinity: &v1.PodAntiAffinity{
					PreferredDuringSchedulingIgnoredDuringExecution: []v1.WeightedPodAffinityTerm{
						{
							Weight: int32(1),
							PodAffinityTerm: v1.PodAffinityTerm{
								TopologyKey: "kubernetes.io/hostname",
								LabelSelector: &k8smetav1.LabelSelector{
									MatchLabels: map[string]string{
										"harvesterhci.io/vmNamePrefix": vmNameBase,
									},
								},
							},
						},
					},
				},
			}

			err := enrichVMTemplate(c, k, vmTemplate, vmInputs)
			if err != nil {
				return fmt.Errorf("unable to enrich VM template with values from flags: %w", err)
			}
		}

		ubuntuVM := &VMv1.VirtualMachine{
			ObjectMeta: k8smetav1.ObjectMeta{
				Name:      vmName,
				Namespace: vmInputs.Namespace,
				Annotations: map[string]string{

					vmAnnotationPVC:        pvcAnnotation,
					vmAnnotationNetworkIps: "[]",
				},
				Labels: vmLabels,
			},
			Spec: VMv1.VirtualMachineSpec{
				Running: actions.NewTrue(),

				Template: vmTemplate,
			},
		}

		if err != nil {
			return err
		}

		_, err = c.KubevirtV1().VirtualMachines(vmInputs.Namespace).Create(context.TODO(), ubuntuVM, k8smetav1.CreateOptions{})

		if err != nil {
			return err
		}
	}

	return nil
}

func enrichVMTemplate(c *harvclient.Clientset, k *kubeclient.Clientset, vmTemplate *VMv1.VirtualMachineInstanceTemplateSpec, vmInputs *VMInputs) error {
	if vmInputs.CPUs > 0 {
		vmTemplate.Spec.Domain.CPU.Cores = uint32(vmInputs.CPUs)
		cpuQuantity := k8sresource.NewQuantity(int64(vmInputs.CPUs), k8sresource.DecimalSI)
		vmTemplate.Spec.Domain.Resources.Limits["cpu"] = *cpuQuantity
		if vmTemplate.Spec.Domain.Resources.Requests == nil {
			vmTemplate.Spec.Domain.Resources.Requests = v1.ResourceList{}
		}
		vmTemplate.Spec.Domain.Resources.Requests["cpu"] = HandleCPUOverCommitment(overCommitSettingMap, int64(vmInputs.CPUs))
	}

	if vmInputs.Memory > 0 {
		vmMemory := strconv.Itoa(vmInputs.Memory)
		vmTemplate.Spec.Domain.Resources.Limits["memory"] = k8sresource.MustParse(vmMemory)
		if vmTemplate.Spec.Domain.Resources.Requests == nil {
			vmTemplate.Spec.Domain.Resources.Requests = v1.ResourceList{}
		}
		vmTemplate.Spec.Domain.Resources.Requests["memory"] = HandleMemoryOverCommitment(overCommitSettingMap, vmMemory)
	}

	networkNS := vmInputs.Network.Namespace
	if networkNS == "" {
		networkNS = vmInputs.Namespace
	}

	dataMap := map[string]any{
		"network-name":          vmInputs.Network.Name,
		"network-namespace":     networkNS,
		"network-data-content":  vmInputs.Network.CloudInitData,
		"network-data-template": vmInputs.Network.CloudInitTemplate,
		"user-name":             vmInputs.User.Name,
		"user-namespace":        vmInputs.Namespace,
		"user-data-content":     vmInputs.User.CloudInitData,
		"user-data-template":    vmInputs.User.CloudInitTemplate,
	}
	for _, userDataType := range []string{"network", "user"} {
		for _, reference := range []string{"content", "template"} {
			if dataMap[userDataType+"-data-"+reference] != nil {
				for _, volume := range vmTemplate.Spec.Volumes {
					if volume.Name == "cloudinitdisk" {
						if userDataType == "network" {
							networkData, err := getCloudInitData(k, dataMap, "network")
							if err != nil {
								return fmt.Errorf("error during the retrieval of the network cloud-init data: %s", err)
							}
							volume.CloudInitNoCloud.NetworkData = networkData
						} else {
							userData, err := getCloudInitData(k, dataMap, "user")
							if err != nil {
								return fmt.Errorf("error during the retrieval of the user cloud-init data: %s", err)
							}
							volume.CloudInitNoCloud.UserData = userData
						}
					}
				}
			}
		}
	}

	return nil
}

// getCloudInitNetworkData gives the ConfigMap object with name indicated in the command line,
// and will create a new one called "ubuntu-std-network" if none is provided or no ConfigMap was found with the same name
func getCloudInitData(k *kubeclient.Clientset, dataMap map[string]any, scope string) (string, error) {
	if scope != "user" && scope != "network" {
		return "", fmt.Errorf("wrong value for scope parameter")
	}
	var err error

	flagName := scope + "-data"
	var cloudInitDataString string

	if dataMap[flagName+"-content"] == nil {
		flagName = flagName + "-template"

		cmName := dataMap[scope+"-name"]
		cmNS := dataMap[scope+"-namespace"]

		if cmName != "" {
			ciData, err := k.CoreV1().ConfigMaps(cmNS.(string)).Get(context.TODO(), cmName.(string), k8smetav1.GetOptions{})

			if err != nil {
				return "", fmt.Errorf("ConfigMap named %s was not found, please specify another ConfigMap or remove the %s input to use the default one for ubuntu", cmName, scope+"-name")
			}

			return ciData.Data["cloudInit"], nil
		}

		if scope == "user" {
			return defaultCloudInitUserData, nil
		} else if scope == "network" {
			return defaultCloudInitNetworkData, nil
		}

	}
	if dataMap[flagName+"-template"] != "" {
		return "", fmt.Errorf("you can't specify both a ConfigMap reference and a file path for the cloud-init data")
	}

	var cloudInitDataBytes []byte
	if dataMap[flagName+"-content"] == nil || len(dataMap[flagName+"-content"].([]byte)) == 0 {
		return "", fmt.Errorf("no cloud-init data was supplied: %s", err)
	}
	cloudInitDataBytes = dataMap[flagName+"-content"].([]byte)
	cloudInitDataString = string(cloudInitDataBytes)

	return cloudInitDataString, nil
}

// BuildVMTemplate creates a *VMv1.VirtualMachineInstanceTemplateSpec from the CLI Flags and some computed values
func BuildVMTemplate(c *harvclient.Clientset, k *kubeclient.Clientset, pvcName string, vmiLabels map[string]string, vmInputs *VMInputs) (vmTemplate *VMv1.VirtualMachineInstanceTemplateSpec, err error) {
	networkNS := vmInputs.Network.Namespace
	if networkNS == "" {
		networkNS = vmInputs.Namespace
	}
	dataMap := map[string]any{
		"network-name":          vmInputs.Network.Name,
		"network-namespace":     networkNS,
		"network-data-content":  vmInputs.Network.CloudInitData,
		"network-data-template": vmInputs.Network.CloudInitTemplate,
		"user-name":             vmInputs.User.Name,
		"user-namespace":        vmInputs.Namespace,
		"user-data-content":     vmInputs.User.CloudInitData,
		"user-data-template":    vmInputs.User.CloudInitTemplate,
	}
	cloudInitCustomUserData, err := getCloudInitData(k, dataMap, "user")
	if err != nil {
		err = fmt.Errorf("error during getting cloud init user data from Harvester: %w", err)
		return
	}

	var sshKey *v1beta1.KeyPair
	if vmInputs.SSHKey.Name != "" {
		sshKey, err = c.HarvesterhciV1beta1().KeyPairs(vmInputs.SSHKey.Namespace).Get(context.TODO(), vmInputs.SSHKey.Name, k8smetav1.GetOptions{})
		if err != nil {
			err = fmt.Errorf("error during getting keypair from Harvester: %w", err)
			return
		}
		logrus.Debugf("SSH Key Name %s given does exist!", vmInputs.SSHKey.Name)

	} else if !userDataContainsKey(cloudInitCustomUserData) {
		sshKey, err = SetDefaultSSHKey(c, vmInputs)
		if err != nil {
			err = fmt.Errorf("error during setting default SSH key: %w", err)
			return
		}
	}

	cloudInitUserData, err := MergeOptionsInUserData(cloudInitCustomUserData, defaultCloudInitUserData, sshKey)
	if err != nil {
		err = fmt.Errorf("error during merging cloud init user data: %w", err)
		return
	}

	cloudInitNetworkData, err := getCloudInitData(k, dataMap, "network")
	if err != nil {
		err = fmt.Errorf("error during getting cloud-init for networking: %w", err)
		return
	}

	vmTemplate = &VMv1.VirtualMachineInstanceTemplateSpec{
		ObjectMeta: k8smetav1.ObjectMeta{
			Annotations: vmiAnnotations(pvcName, vmInputs.SSHKey.Name),
			Labels:      vmiLabels,
		},
		Spec: VMv1.VirtualMachineInstanceSpec{
			Hostname: vmInputs.Name,
			Networks: []VMv1.Network{

				{
					Name: "nic-1",

					NetworkSource: VMv1.NetworkSource{
						Multus: &VMv1.MultusNetwork{
							NetworkName: vmInputs.Network.Name,
						},
					},
				},
			},
			Volumes: []VMv1.Volume{
				{
					Name: "disk-0",
					VolumeSource: VMv1.VolumeSource{
						PersistentVolumeClaim: &VMv1.PersistentVolumeClaimVolumeSource{
							PersistentVolumeClaimVolumeSource: v1.PersistentVolumeClaimVolumeSource{
								ClaimName: pvcName,
							},
						},
					},
				},
				{
					Name: "cloudinitdisk",
					VolumeSource: VMv1.VolumeSource{
						CloudInitNoCloud: &VMv1.CloudInitNoCloudSource{
							UserData:    cloudInitUserData,
							NetworkData: cloudInitNetworkData,
						},
					},
				},
			},
			Domain: VMv1.DomainSpec{
				CPU: &VMv1.CPU{
					Cores:   uint32(vmInputs.CPUs),
					Sockets: 1,
					Threads: 1,
				},
				Devices: VMv1.Devices{
					Inputs: []VMv1.Input{
						{
							Bus:  "usb",
							Type: "tablet",
							Name: "tablet",
						},
					},
					Interfaces: []VMv1.Interface{
						{
							Name:                   "nic-1",
							Model:                  "virtio",
							InterfaceBindingMethod: VMv1.DefaultBridgeNetworkInterface().InterfaceBindingMethod,
						},
					},
					Disks: []VMv1.Disk{
						{
							Name: "disk-0",
							DiskDevice: VMv1.DiskDevice{
								Disk: &VMv1.DiskTarget{
									Bus: "virtio",
								},
							},
						},
						{
							Name: "cloudinitdisk",
							DiskDevice: VMv1.DiskDevice{
								Disk: &VMv1.DiskTarget{
									Bus: "virtio",
								},
							},
						},
					},
				},
				Resources: VMv1.ResourceRequirements{
					Requests: v1.ResourceList{
						"memory": HandleMemoryOverCommitment(overCommitSettingMap, fmt.Sprintf("%dGi", vmInputs.Memory)),
						"cpu":    HandleCPUOverCommitment(overCommitSettingMap, int64(vmInputs.CPUs)),
					},
					Limits: v1.ResourceList{
						"memory": k8sresource.MustParse(fmt.Sprintf("%dGi", vmInputs.Memory)),
						"cpu":    *k8sresource.NewQuantity(int64(vmInputs.CPUs), k8sresource.DecimalSI),
					},
				},
			},
			Affinity: &v1.Affinity{
				PodAntiAffinity: &v1.PodAntiAffinity{
					PreferredDuringSchedulingIgnoredDuringExecution: []v1.WeightedPodAffinityTerm{
						{
							Weight: int32(1),
							PodAffinityTerm: v1.PodAffinityTerm{
								TopologyKey: "kubernetes.io/hostname",
								LabelSelector: &k8smetav1.LabelSelector{
									MatchLabels: map[string]string{
										"harvesterhci.io/vmNamePrefix": vmInputs.Name,
									},
								},
							},
						},
					},
				},
			},
		},
	}
	return
}

// vmiAnnotations generates a map of strings to be injected as annotations from a PVC name and an SSK Keyname
func vmiAnnotations(pvcName string, sshKeyName string) map[string]string {
	return map[string]string{
		"harvesterhci.io/diskNames": "[\"" + pvcName + "\"]",
		"harvesterhci.io/sshNames":  "[\"" + sshKeyName + "\"]",
	}
}

// checks if the userData contains an ssh_authorized_keys entry
func userDataContainsKey(userData string) bool {
	var userDataMap map[string]interface{}

	if err := yaml.Unmarshal([]byte(userData), &userDataMap); err != nil {
		return false
	}

	if _, ok := userDataMap["ssh_authorized_keys"]; ok {
		return true
	}

	return false

}

// BuildVMListMatchingWildcard creates an array of VM objects which names match the given wildcard pattern
func BuildVMListMatchingWildcard(c *harvclient.Clientset, namespace, vmNameWildcard string) ([]VMv1.VirtualMachine, error) {
	vms, err := c.KubevirtV1().VirtualMachines(namespace).List(context.TODO(), k8smetav1.ListOptions{})

	if err != nil {
		return nil, fmt.Errorf("No VMs found with name %s", vmNameWildcard)
	}

	var matchingVMs []VMv1.VirtualMachine
	for _, vm := range vms.Items {
		if wildcard.Match(vmNameWildcard, vm.Name) {
			matchingVMs = append(matchingVMs, vm)
		}
	}
	logrus.Infof("number of matching VMs for pattern %s: %d", vmNameWildcard, len(matchingVMs))
	return matchingVMs, nil
}

// SetDefaultVMImage creates a default VM image based on Ubuntu if none has been provided at the command line.
func SetDefaultVMImage(c *harvclient.Clientset, vmInputs *VMInputs) (result *v1beta1.VirtualMachineImage, err error) {

	result = &v1beta1.VirtualMachineImage{}
	vmImages, err := c.HarvesterhciV1beta1().VirtualMachineImages(vmInputs.Image.Namespace).List(context.TODO(), k8smetav1.ListOptions{})

	if err != nil {
		err = fmt.Errorf("error during setting default VM Image: %w", err)
		return
	}

	var vmImage *v1beta1.VirtualMachineImage

	if len(vmImages.Items) == 0 {
		vmImage, err = CreateVMImage(c, vmInputs.Image.Namespace, "ubuntu-default-image", ubuntuDefaultImage)
		if err != nil {
			err = fmt.Errorf("impossible to create a default VM Image: %s", err)
			return
		}
	} else {
		vmImage = &vmImages.Items[0]
	}

	imageID := vmImage.ObjectMeta.Name
	vmInputs.Image.ID = imageID
	imageName := vmImage.Spec.DisplayName
	vmInputs.Image.Name = imageName

	if err != nil {
		logrus.Warnf("error encountered during the storage of the imageID value: %s", imageID)
	}

	result = vmImage

	return
}

// CreateVMImage will create a VM Image on Harvester given an image name and an image URL
func CreateVMImage(c *harvclient.Clientset, namespace string, imageName string, url string) (*v1beta1.VirtualMachineImage, error) {
	suffix, err := randGen.New(6)
	if err != nil {
		return nil, err
	}
	vmImage, err := c.HarvesterhciV1beta1().VirtualMachineImages(namespace).Create(
		context.TODO(),
		&v1beta1.VirtualMachineImage{
			ObjectMeta: k8smetav1.ObjectMeta{
				Name: "ubuntu-default" + suffix,
			},
			Spec: v1beta1.VirtualMachineImageSpec{
				DisplayName: imageName,
				URL:         url,
			},
		},
		k8smetav1.CreateOptions{})

	if err != nil {
		return &v1beta1.VirtualMachineImage{}, err
	}

	return vmImage, nil
}

// SetDefaultSSHKey assign a default SSH key to the VM if none was provided at the command line
func SetDefaultSSHKey(c *harvclient.Clientset, vmInputs *VMInputs) (sshKey *v1beta1.KeyPair, err error) {
	sshKey = &v1beta1.KeyPair{}
	sshKeys, err := c.HarvesterhciV1beta1().KeyPairs(vmInputs.Namespace).List(context.TODO(), k8smetav1.ListOptions{})

	if err != nil {
		err = fmt.Errorf("error during listing KeyPairs: %s", err)
		return
	}

	if len(sshKeys.Items) == 0 {
		err = fmt.Errorf("no ssh keys exists in harvester, please add a new ssh key")
		return
	}

	sshKey = &sshKeys.Items[0]
	vmInputs.SSHKey.Name = sshKey.Name
	vmInputs.SSHKey.Namespace = sshKey.Namespace

	if err != nil {
		logrus.Warnf("Error encountered during the storage of the SSH Keyname value: %s", sshKey.Name)
	}
	return
}

// MergeOptionsInUserData merges the default user data and the provided public key with the user data provided by the user
func MergeOptionsInUserData(userData string, defaultUserData string, sshKey *v1beta1.KeyPair) (string, error) {
	var err error
	var userDataMap map[string]interface{}
	var defaultUserDataMap map[string]interface{}

	err = yaml.Unmarshal([]byte(userData), &userDataMap)
	if err != nil {
		return "", err
	}

	err = yaml.Unmarshal([]byte(defaultUserData), &defaultUserDataMap)
	if err != nil {
		return "", err
	}

	if (userDataMap["ssh_authorized_keys"] != nil && sshKey != nil && sshKey != &v1beta1.KeyPair{}) {
		sshKeyList := userDataMap["ssh_authorized_keys"].([]interface{})
		sshKeyList = append(sshKeyList, sshKey.Spec.PublicKey)

		userDataMap["ssh_authorized_keys"] = sshKeyList
	}

	if userDataMap["packages"] != nil {
		packagesList := userDataMap["packages"].([]interface{})
		packagesList = append(packagesList, defaultUserDataMap["packages"].([]interface{})...)
		userDataMap["packages"] = packagesList
	} else {
		userDataMap["packages"] = defaultUserDataMap["packages"]
	}

	if userDataMap["runcmd"] != nil {
		runcmdList := defaultUserDataMap["runcmd"].([]interface{})
		runcmdList = append(runcmdList, userDataMap["runcmd"].([]interface{})...)
		userDataMap["runcmd"] = runcmdList
	} else {
		userDataMap["runcmd"] = defaultUserDataMap["runcmd"]
	}

	mergedUserData, err := yaml.Marshal(userDataMap)

	if err != nil {
		return "", err
	}

	finalUserData := fmt.Sprintf("#cloud-config\n%s", string(mergedUserData))

	return finalUserData, nil

}
