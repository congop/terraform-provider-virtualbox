package virtualbox

import (
	"bytes"
	"log"
	"os"
	"testing"

	stubbing "github.com/congop/execstub"
	"github.com/congop/execstub/pkg/comproto"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	vbox "github.com/terra-farm/go-virtualbox"
)

func SetupCaptureAndLogGoLogs(t *testing.T) (deferTestLogAndResetFunc func()) {
	var buffer bytes.Buffer
	log.SetOutput(&buffer)
	return func() {
		t.Logf("\n--------------<Log\n%s\n-------------->Log\n", buffer.String())
		log.SetOutput(os.Stdout)
	}
}

func Test_waitUntilAnyUnmetCreationPolicyOrAllTimeout(t *testing.T) {
	// stubber := stubbing.NewExecStubber()
	vbox.Debug = t.Logf
	defer SetupCaptureAndLogGoLogs(t)()
	tfRes := resourceVM()
	tfResData := tfRes.TestResourceData()
	err := tfResData.Set("creation_policy", []map[string]interface{}{
		{
			"type": "cloud_init_done_by_vm_guestproperty",
			//"timeout": "PT30S",
			"timeout": "PT2S",
			"spec":    map[string]string{},
		},
	})

	if nil != err {
		t.Fatalf("Error creating tf resource test data:%v", err)
	}
	type args struct {
		d        *schema.ResourceData
		vm       *vbox.Machine
		meta     interface{}
		stubFunc comproto.StubFunc
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "Should pickup the status done and return error free",
			args: args{
				d:    tfResData,
				vm:   &vbox.Machine{Name: "kk8s_controller1"},
				meta: nil,
				//"status: done" "status: error"
				stubFunc: comproto.AdaptOutcomesToCmdStub([]*comproto.ExecOutcome{
					{
						Key: "", ExitCode: 0, Stderr: "", Stdout: "Value: status: running", InternalErrTxt: "",
					},
					{
						Key: "", ExitCode: 0, Stderr: "", Stdout: "XXXX", InternalErrTxt: "internal err",
					},
					{
						Key: "", ExitCode: 0, Stderr: "", Stdout: "Value: status: done", InternalErrTxt: "",
					},
				}, false),
			},
			wantErr: false,
		},
		{
			name: "Should pickup static error and return with error",
			args: args{
				d:    tfResData,
				vm:   &vbox.Machine{Name: "kk8s_controller1"},
				meta: nil,
				//"status: done" "status: error"
				stubFunc: comproto.AdaptOutcomesToCmdStub([]*comproto.ExecOutcome{
					{
						Key: "", ExitCode: 0, Stderr: "", Stdout: "Value: status: running", InternalErrTxt: "",
					},
					{
						Key: "", ExitCode: 0, Stderr: "", Stdout: "Value: status: error", InternalErrTxt: "",
					},
				}, true),
			},
			wantErr: true,
		},
		{
			name: "Should timeout",
			args: args{
				d:    tfResData,
				vm:   &vbox.Machine{Name: "kk8s_controller1"},
				meta: nil,
				//"status: done" "status: error"
				stubFunc: comproto.AdaptOutcomesToCmdStub([]*comproto.ExecOutcome{
					{
						Key: "", ExitCode: 0, Stderr: "", Stdout: "Value: status: running", InternalErrTxt: "",
					},
					{
						Key: "", ExitCode: 0, Stderr: "", Stdout: "XXXX", InternalErrTxt: "internal err",
					},
					{
						Key: "", ExitCode: 0, Stderr: "", Stdout: "Value: status: running", InternalErrTxt: "",
					},
				}, true),
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stubber := stubbing.NewExecStubber()
			defer stubber.CleanUp()

			settings := comproto.SettingsDynaStubCmdDiscoveredByPath()

			stubFunc, reqs := comproto.RecordingExecutions(tt.args.stubFunc)
			stubber.WhenExecDoStubFunc(stubFunc, "VBoxManage", *settings)

			///////////
			err := waitUntilAnyUnmetCreationPolicyOrAllTimeout(tt.args.d, tt.args.vm, tt.args.meta)
			if (err != nil) != tt.wantErr {
				t.Errorf(
					"waitUntilAnyUnmetCreationPolicyOrAllTimeout() "+
						"\n\terror=%v \n\twantErr=%v \n\treqs=%#v",
					err, tt.wantErr, reqs)
			}

		})
	}
}
