package virtualbox

import (
	"fmt"

	"github.com/hashicorp/go-multierror"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	vbox "github.com/terra-farm/go-virtualbox"
)

func uartTfToVbox(d *schema.ResourceData) (*vbox.UARTs, error) {
	uartTf, ok := d.GetOk("uart")
	if !ok {
		return vbox.NewUARTsAllOff(), nil
	}
	uartList, ok := uartTf.([]interface{})
	if !ok {
		return nil, fmt.Errorf("could not convert uartTf to schema.Set: Type:%T, Value:%#v", uartTf, uartTf)
	}

	uartsMap := make(map[vbox.UARTKey]vbox.UART)
	multierr := &multierror.Error{}

	for _, uartTfMapInterface := range uartList {
		uartTfMap, ok := uartTfMapInterface.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("could not convert uart tf set elemnt to map[string]interface{}:%#v", uartTfMapInterface)

		}

		var key, port, irq, typeStr, mode, modeData string
		var err error

		if key, err = getMapValueAsString(uartTfMap, "key"); nil != err {
			multierr = multierror.Append(multierr, err)
		}

		if port, err = getMapValueAsString(uartTfMap, "port"); nil != err {
			multierr = multierror.Append(multierr, err)
		}

		if irq, err = getMapValueAsString(uartTfMap, "irq"); !ok {
			multierr = multierror.Append(multierr, err)
		}

		if typeStr, err = getMapValueAsString(uartTfMap, "type"); nil != err {
			multierr = multierror.Append(multierr, err)
		}

		if mode, err = getMapValueAsString(uartTfMap, "mode"); nil != err {
			multierr = multierror.Append(multierr, err)
		}

		if modeData, err = getMapValueAsString(uartTfMap, "mode_data"); nil != err {
			multierr = multierror.Append(multierr, err)
		}

		uart, err := vbox.NewUART(key, typeStr, port, irq, mode, modeData)
		multierr = multierror.Append(multierr, err)
		if _, avail := uartsMap[uart.Key]; avail {
			multierr = multierror.Append(multierr, fmt.Errorf("uart must not have matching keys, but found more than one uart with key:%s", uart.Key))
		}
		uartsMap[uart.Key] = *uart
	}

	uarts, errUARTs := vbox.NewUARTsFromUARTMap(uartsMap)
	multierr = multierror.Append(multierr, errUARTs)

	return uarts, multierr.ErrorOrNil()
}

func uartVboxToTf(vm *vbox.Machine, d *schema.ResourceData) error {
	// Assign NIC property to vbox structure and Terraform
	uarts := make([]interface{}, 0, 4) // make([]map[string]interface{}, 0, 4)

	uartsTfStateRelevant := append(vbox.UARTs{}, vm.UARTs...)
	(&uartsTfStateRelevant).WithoutUARTHavingStateOff()

	for _, uart := range uartsTfStateRelevant {
		out := make(map[string]interface{})

		setIfNotEmpty(out, "key", string(uart.Key))
		setIfNotEmpty(out, "port", uart.ComConfig.PortAsIoBaseHexString())
		setIfNotEmpty(out, "irq", uart.ComConfig.IRQAsString())
		setIfNotEmpty(out, "type", string(uart.Type))
		setIfNotEmpty(out, "mode", string(uart.Mode))
		setIfNotEmpty(out, "mode_data", uart.ModeData)

		uarts = append(uarts, out)
	}

	return d.Set("uart", uarts)
}

func setIfNotEmpty(targetMap map[string]interface{}, key string, value string) {
	if value == "" {
		return
	}
	targetMap[key] = value
}
