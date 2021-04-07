package virtualbox

import (
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/godoctor/godoctor/filesystem"
	"github.com/pkg/errors"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	vbox "github.com/terra-farm/go-virtualbox"
)

type CloudInitData map[string]string

// CreateCloudInitNoCloudIsoFromUserData create a iso medium containing the given userdata under user_data
func CreateCloudInitNoCloudIsoFromUserData(
	machineFolder string,
	cloudInitData *CloudInitData,
) (isoFilePath string, err error) {
	stat, err := os.Stat(machineFolder)

	if nil != err {
		return "", errors.Wrapf(err, "Error getting stat: %v", machineFolder)
	}
	if !stat.IsDir() {
		return "", errors.Errorf("Given mchineFolder must be a directory: %s", machineFolder)
	}
	nocloudDir, _, err := createCloudInitNoCloudDirHoldingContainingUserDataFile(machineFolder, cloudInitData)
	if nil != err {
		return "", err
	}
	defer cleanUpNocloudDir(nocloudDir)
	cloudInitPath := filepath.Join(machineFolder, "cloud-init.iso")
	///dev/disk/by-label
	//genisoimage -output nocloud.iso -volid cidata -joliet -rock nocloud/
	cmd := exec.Command("genisoimage", "-output", cloudInitPath, "-V", "cidata", "-joliet", "-rock", nocloudDir)
	stdInOutBytes, err := cmd.CombinedOutput()
	if err != nil {
		return "", errors.Wrapf(err, "Command execution failed: cmd=%s , StdIn/Out=%s", cmd, string(stdInOutBytes))
	}

	return cloudInitPath, nil
}

func createCloudInitNoCloudDirHoldingContainingUserDataFile(
	machineFolder string,
	cloudInitData *CloudInitData,
) (nocloudDir string, userDataPath string, err error) {
	fs := filesystem.NewLocalFileSystem()
	//noCloudDirPath := filepath.Join(machineFolder, "nocloud")
	noCloudDirPath, err := ioutil.TempDir(machineFolder, "nocloud*")
	if nil != err {
		return "", "", errors.Wrapf(err, "Could not create nocloud dir under: %s", machineFolder)
	}
	// if err := os.Mkdir(noCloudDirPath, os.ModeDir); nil != err {
	// 	return "", errors.Wrapf(err, "Could not MkDir: %s", noCloudDirPath)
	// }
	userDataPath = filepath.Join(noCloudDirPath, "user-data")
	//ioutil.WriteFile()
	fs.Remove(userDataPath)
	userData, ok := (*cloudInitData)["user-data"]
	if !ok || userData == "" {
		return "", "", errors.Errorf("user data not provided:%s", cloudInitData)
	}
	if err := fs.CreateFile(userDataPath, (*cloudInitData)["user-data"]); nil != err {
		cleanUpNocloudDir(noCloudDirPath)
		return "", "", errors.Wrapf(err, "Could not write user data to: %s", userDataPath)
	}

	metaDataPath := filepath.Join(noCloudDirPath, "meta-data")
	fs.Remove(metaDataPath)

	if err := fs.CreateFile(metaDataPath, (*cloudInitData)["meta-data"]); nil != err {
		cleanUpNocloudDir(noCloudDirPath)
		return "", "", errors.Wrapf(err, "Could not write meta data to: %s", userDataPath)
	}

	networkConfigPath := filepath.Join(noCloudDirPath, "network-config")
	fs.Remove(networkConfigPath)

	if err := fs.CreateFile(networkConfigPath, (*cloudInitData)["network-config"]); nil != err {
		cleanUpNocloudDir(noCloudDirPath)
		return "", "", errors.Wrapf(err, "Could not write network config to: %s", networkConfigPath)
	}

	return noCloudDirPath, userDataPath, nil
}

func cleanUpNocloudDir(noCloudDirPath string) error {
	stat, err := os.Stat(noCloudDirPath)
	if nil != err {
		log.Printf("Error getting stat of NoCloudDir(%s), skipping clean up", noCloudDirPath)
		return nil
	}
	if !stat.IsDir() {
		return nil
	}

	return os.RemoveAll(noCloudDirPath)
}

// AttachCloudInitUserData attach a cidata cd created from the given user data und the VM base folder.
func AttachCloudInitUserData(
	d *schema.ResourceData,
	meta interface{},
	vm *vbox.Machine, port uint,
) error {
	cloudInitData, err := NewCloudInitData(d, meta)
	if nil != err {
		return errors.Wrap(err, "Error getting cloud-init data, while AttachCloudInitUserData")
	}

	isoFilePath, err := CreateCloudInitNoCloudIsoFromUserData(vm.BaseFolder, cloudInitData)
	if nil != err {
		return errors.Wrapf(err, "Error creating cloud init no cloud user-data from user data")
	}

	err = vm.AttachStorage("SATA", vbox.StorageMedium{
		Port:      port,
		Device:    0,
		DriveType: vbox.DriveDVD,
		Medium:    isoFilePath,
	})
	if err != nil {
		return errors.Wrapf(err, "Attaching cloud init iso: %s", isoFilePath)
	}
	return nil
}

// NewCloudInitData construct a new CloudInitData given the tf resource data
func NewCloudInitData(d *schema.ResourceData, meta interface{}) (*CloudInitData, error) {
	tfCloudInitDataSliceI, ok := d.GetOk("cloud_init")
	if !ok {
		return &CloudInitData{}, nil
	}

	tfCloudInitDataSlice, ok := tfCloudInitDataSliceI.([]interface{})
	if !ok {
		return nil, errors.Errorf("Bad type expected []interface{} but got: %#v", tfCloudInitDataSlice)
	}

	if len(tfCloudInitDataSlice) == 0 {
		return &CloudInitData{}, nil
	}

	if 2 <= len(tfCloudInitDataSlice) {
		return nil, errors.Errorf("Expectd exactly one cloud-init resource expected but got: %d", len((tfCloudInitDataSlice)))
	}

	tfCloudInitData, ok := tfCloudInitDataSlice[0].(map[string]interface{})
	if !ok {
		return nil, errors.Errorf("Error converting tf cloud init into map[string]interface{}: %#v", tfCloudInitDataSlice[0])
	}

	cloudInitData := CloudInitData{}
	tfToCloudInitFName := map[string]string{
		"user_data": "user-data", "meta_data": "meta-data",
		"network_config": "network-config"}
	for tfKey, cloudInitFileName := range tfToCloudInitFName {
		data, err := getMapValueAsString(tfCloudInitData, tfKey)
		if nil != err {
			return nil, errors.Wrapf(err, "Error getting cloud-init data %s", tfKey)
		}

		cloudInitData[cloudInitFileName] = data
	}

	return &cloudInitData, nil

}
