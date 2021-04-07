package virtualbox

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	vbox "github.com/terra-farm/go-virtualbox"
)

func Test_uartVboxToTf(t *testing.T) {
	tfRes := resourceVM()
	tfResData := tfRes.TestResourceData()

	vmUARTs := vmFromMap(t, map[string]string{
		"uart1": "0x2f8,3", "uarttype1": "16550A", "uartmode1": "file,/tmp/uart1"})
	vm := vbox.New()
	vm.UARTs = *vmUARTs

	///
	err := uartVboxToTf(vm, tfResData)
	///

	assert.Zerof(t, err, "Error transfering tf data to VM UARTs")

	expectedUartTf := []interface{}{
		map[string]interface{}{
			"key":       "uart1",
			"port":      "0x02f8",
			"irq":       "3",
			"type":      "16550A",
			"mode":      "file",
			"mode_data": "/tmp/uart1",
		},
	}

	uartMapActual, ok := tfResData.GetOk("uart")
	if !ok || !reflect.DeepEqual(expectedUartTf, uartMapActual) {
		t.Fatalf(
			"TF uart should only contain uart1: "+
				"\nexpectedUarts=%#v, \nuartMapActual=%#v \ntfResData=%##v \nok=%t",
			expectedUartTf, uartMapActual, tfResData, ok)
	}
}

func Test_uartTfToVM(t *testing.T) {
	tfRes := resourceVM()
	tfResData := tfRes.TestResourceData()
	tfResData.Set("uart", []map[string]string{
		{
			"key":       "uart1",
			"port":      "0x2f8",
			"irq":       "3",
			"type":      "16550A",
			"mode":      "file",
			"mode_data": "/tmp/uart1",
		},
	})

	pUARTs, err := uartTfToVbox(tfResData)

	assert.Zerof(t, err, "Error transfering tf data to VM UARTs")
	vmInfoMap := map[string]string{"uart1": "0x2f8,3", "uarttype1": "16550A", "uartmode1": "file,/tmp/uart1"}
	expectedUARTs, err := vbox.NewUARTs(vmInfoMap)
	assert.Zerof(t, err, "Error constructing expected uarts")
	assert.EqualValues(t, expectedUARTs, pUARTs, "uart should have been map to VM and other uarts off")
}

func vmFromMap(t *testing.T, vmInfoMap map[string]string) *vbox.UARTs {
	uarts, err := vbox.NewUARTs(vmInfoMap)
	assert.Zerof(t, err, "Error constructing expected uarts from vmInfoMap:%s", vmInfoMap)
	//assert.Zerof(t, err, "Error constructing expected uarts")
	return uarts
}
