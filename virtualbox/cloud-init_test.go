package virtualbox

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"golang.org/x/exp/rand"
)

func TestCreateCloudInitNoCloudIsoFromUserData(t *testing.T) {
	dirOk, err := ioutil.TempDir("/tmp", "00-test-dir-cloud-init-*")
	if nil != err {
		t.Fatalf("Could not create temp-dir in /tmp:%v", err)
	}

	dirDoesNotExists := fmt.Sprintf("/tmp/does-not-exists-%d", rand.Uint64())

	defer t.Cleanup(
		func() {
			fmt.Printf("[Test:%s] Removing TempDir:%s", t.Name(), dirOk)
			os.RemoveAll(dirOk)
		},
	)

	type args struct {
		machineFolder string
		cloudInitData CloudInitData
	}
	tests := []struct {
		name            string
		args            args
		wantIsoFilePath string
		wantErr         bool
	}{
		{
			name: "Iso should have been created at: " + dirOk,
			args: args{
				machineFolder: dirOk,
				cloudInitData: CloudInitData{"user-data": "name: MMMEEE"},
			},
			wantIsoFilePath: filepath.Join(dirOk, "cloud-init.iso"),
		},
		{
			name: "Should have failed because machine folder does not exists: " + dirOk,
			args: args{
				machineFolder: dirDoesNotExists,
				cloudInitData: CloudInitData{"user-data": "name: MMMEEE"},
			},
			wantIsoFilePath: "",
			wantErr:         true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotIsoFilePath, err := CreateCloudInitNoCloudIsoFromUserData(tt.args.machineFolder, &tt.args.cloudInitData)
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateCloudInitNoCloudIsoFromUserData() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotIsoFilePath != tt.wantIsoFilePath {
				t.Errorf("CreateCloudInitNoCloudIsoFromUserData() = %v, want %v", gotIsoFilePath, tt.wantIsoFilePath)
				return
			}
			if err == nil {
				assertCloudInitDataEquals(t, gotIsoFilePath, tt.args.cloudInitData["user-data"])
			}
		})
	}
}

func extractIsoUsing7ZipTo(
	isoFilePath string, extractDirPath string) error {
	//7z e my.iso -o/tmp/output-dir
	//7z x -y -oC:\OutputDirectory X:\VRMPVOL_EN.iso
	cmd := exec.Command("7z", "x", "-y", fmt.Sprintf("-o%s", extractDirPath), isoFilePath)
	if stdInStdOut, err := cmd.CombinedOutput(); nil != err {
		return errors.Wrapf(err,
			"Fail using 7 zip to extract iso content: "+
				"\n\tisoFilePath:%s, \n\textractDirPath=%s, \n\tcmd=%s \n\tstdin/stdout=%s",
			isoFilePath, extractDirPath, cmd, string(stdInStdOut))
	}
	return nil
}
func assertCloudInitDataEquals(
	t *testing.T, isoFilePath string, expectedUserData string) {
	extractDirPath, err := ioutil.TempDir("/tmp", "nocloud-iso-extracted-*")
	defer os.RemoveAll(extractDirPath)
	if nil != err {
		t.Fatalf("Could not create tmp-dir to use as extraction target for iso: %v", err)
		return
	}
	if err := extractIsoUsing7ZipTo(isoFilePath, extractDirPath); nil != err {
		t.Fatalf("error extracting iso with 7zip: %v", err)
		return
	}
	extractedUserDataPath := filepath.Join(extractDirPath, "user-data")
	extractedUserData, err := ioutil.ReadFile(extractedUserDataPath)
	if nil != err {
		t.Fatalf("Error reading extracted user-data:%v", err)
		return
	}
	assert.Equal(t, expectedUserData, string(extractedUserData))
}
