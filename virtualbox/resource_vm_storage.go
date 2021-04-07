package virtualbox

import (
	"log"
	"os"
	"path/filepath"
	"strings"

	vbox "github.com/terra-farm/go-virtualbox"
)

func deleteVMStorage(vm *vbox.Machine) {
	vmBaseFolder := vm.BaseFolder
	media := make([]string, 0, 32)
	media = append(media, vm.StorageControllers.DeviceMedia()...)
	vmGoldFolder, err := vmGoldFolder(vm.UUID)
	if err != nil {
		vbox.Debug("failed to get golFolder for vm(%s -- %s)", vm.Name, vm.UUID)
		// ignoring the err because we are in cleanup-mode
		// we cannot have goldFolder be an empty string because of the HasPrefix() test
		vmGoldFolder = vmBaseFolder
	} else {
		err = filepath.Walk(vmGoldFolder, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				vbox.Debug("cannot walk file or dir %q: %v\n", path, err)
				return nil
			}
			if info.IsDir() {
				return nil
			}
			if isVMMediumFileExt(filepath.Ext(path)) {
				media = append(media, path)
			}
			return nil
		})
		if err != nil {
			log.Printf("error walking %s", vmGoldFolder)
		}
	}
	vbox.Debug(
		"deleteVMStorage(%s - %s) \n\t media:%s \n\tVmGoldFoler=%s \n\tVMBaseFolder=%s",
		vm.Name, vm.UUID, media, vmGoldFolder, vmBaseFolder)
	for _, medium := range media {
		skipped, deleted := deleteVMMedium(medium, vmBaseFolder, vmGoldFolder)
		if !skipped && !deleted {
			log.Printf("could not delete VM Medium:%s", medium)
		}
	}
	err = os.RemoveAll(vmGoldFolder)
	if err != nil {
		log.Printf("error while deleting vm-gold-folder:%s", vmGoldFolder)
	}
	err = os.RemoveAll(vmBaseFolder)
	if err != nil {
		log.Printf("error while deleting vm-base-folder:%s", vmBaseFolder)
	}
	mayBeUUIDBasedVMFolder := filepath.Dir(vmBaseFolder)
	if strings.HasSuffix(mayBeUUIDBasedVMFolder, vm.UUID) {
		err = os.RemoveAll(mayBeUUIDBasedVMFolder)
		if err != nil {
			log.Printf("error while deleting UUIDBasedVMFolder:%s", mayBeUUIDBasedVMFolder)
		}
	}
}

func isVMMediumFileExt(ext string) bool {
	return ext == ".vmdk" || ext == ".vdi" || ext == ".iso" ||
		ext == ".img" || ext == ".raw"
}

func deleteVMMedium(
	medium string, vmBaseDir string, vmGoldFolder string,
) (skippedAsNotVMResource bool, deleted bool) {
	ownByThisVM := strings.HasPrefix(medium, vmGoldFolder) ||
		strings.HasPrefix(medium, vmBaseDir)
	if !ownByThisVM {
		return true, false
	}
	//device type not return by vminfo, so that is  not available
	// therefore just trying a all known type untile success
	for _, mtype := range []string{"disk", "dvd", "floppy"} {
		stderr, stdout, err := vbox.RunVBoxManageCmd("closemedium", mtype, medium, "--delete")
		if vbox.Verbose {
			vbox.Debug(
				"Run VBoxManage closemedium %s %s --delete "+
					"\n\tstderr=%s \n\tstdout=%s \n\terr=%v",
				mtype, medium, stderr, stdout, err)
		}
		if err == nil {
			return false, true
		}
	}
	return false, false
}
