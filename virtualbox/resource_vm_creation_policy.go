package virtualbox

import (
	"log"
	"strconv"
	"time"

	"github.com/pkg/errors"

	"fmt"

	"sync"

	"github.com/ajvb/kala/utils/iso8601"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	vbox "github.com/terra-farm/go-virtualbox"
)

func tfGetListOfMap(d *schema.ResourceData, key string) ([]map[string]interface{}, error) {
	asI, ok := d.GetOk(key)
	if !ok {
		return nil, nil
	}

	listOfInterfaces, ok := asI.([]interface{})
	if !ok {
		return nil, fmt.Errorf("could not convert to []interface{}: %#v", asI)
	}

	listOfMap := make([]map[string]interface{}, 0, len(listOfInterfaces))
	for _, eI := range listOfInterfaces {

		listOfMapElement, ok := eI.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("could not convert to element to map[string]interface{}: %#v", asI)
		}
		listOfMap = append(listOfMap, listOfMapElement)
	}
	return listOfMap, nil
}

func getTimeoutFromTfSpec(tfSpec map[string]interface{}, defaultValue string) (time.Duration, error) {
	timeoutStr, err := getMapValueAsString(tfSpec, "timeout")
	if nil != err {
		return -1, errors.Wrapf(err, "Error getting timeout from tf-spec: %v", tfSpec)
	}
	if timeoutStr == "" {
		timeoutStr = defaultValue
	}
	timeoutIso8601, err := iso8601.FromString(timeoutStr)
	if nil != err {
		return -1, errors.Wrapf(
			err, "Error parsing tf-spec timeout string as iso8601 periode: periode=%s, tf-spec=%v",
			timeoutStr, tfSpec)
	}
	timeout := timeoutIso8601.RelativeTo(time.Unix(0, 0))
	return timeout, nil
}

func creationPolicyCloudInitDoneByVMGuestProperty(
	key string, vm *vbox.Machine, tfSpec map[string]interface{},
	wg *sync.WaitGroup, chanCreationPolicyOutcome chan creationPolicyOutCome) (*creationPolicy, error) {
	cancelChan := make(chan string, 1)
	timeout, err := getTimeoutFromTfSpec(tfSpec, "PT3M") // time.Duration(10 * time.Second)
	if nil != err {
		return nil, err
	}
	check := func(value string, err error) *creationPolicyOutCome {
		passed := false
		state := waitStateChecking
		if value == "status: done" {
			passed = true
			state = waitStateEnded
		}

		if value == "status: error" {
			passed = false
			state = waitStateEnded
		}

		o := creationPolicyOutCome{
			key:    key,
			passed: passed,
			state:  state,
		}
		if err != nil {
			o.detail = err.Error()
		}
		return &o
	}
	cp := creationPolicy{
		key:        key,
		cptype:     "cloud_init_done_by_vm_guestproperty",
		cancelChan: cancelChan,
		timeout:    timeout,
		startCheckingFunc: func() {
			waitGuestProperties(
				key, vm.Name, "/VirtualBox/GuestInfo/CloudInit/Status",
				cancelChan, wg, timeout, chanCreationPolicyOutcome, check)
		},
		cancelOnce: &sync.Once{},
	}
	return &cp, nil
}

type creationPolicyContructor func(
	key string, vm *vbox.Machine, tfSpec map[string]interface{},
	wg *sync.WaitGroup, chanCreationPolicyOutcome chan creationPolicyOutCome) (*creationPolicy, error)

var (
	creationPolicyConstructors = map[string]creationPolicyContructor{
		"cloud_init_done_by_vm_guestproperty": creationPolicyCloudInitDoneByVMGuestProperty,
	}
)

func supporttedCreationPolicyTypes() []string {
	// keys := reflect.ValueOf(abc).MapKeys()
	types := make([]string, 0, len(creationPolicyConstructors))
	for k := range creationPolicyConstructors {
		types = append(types, k)
	}
	return types
}

func maxTimeout(cps map[string]*creationPolicy) time.Duration {
	max := time.Duration(0)
	for _, cp := range cps {
		if max < cp.timeout {
			max = cp.timeout
		}
	}
	return max
}

// Wait until VM is ready, and 'ready' means the first non NAT NIC get a ipv4_address assigned
func waitUntilAnyUnmetCreationPolicyOrAllTimeout(d *schema.ResourceData, vm *vbox.Machine, meta interface{}) error {
	tfCreationPolicies, err := tfGetListOfMap(d, "creation_policy")
	if nil != err {
		return errors.Wrapf(err, "Failed to get creation_policy")
	}

	if len(tfCreationPolicies) == 0 {
		return nil
	}

	cps := map[string]*creationPolicy{}

	var wg sync.WaitGroup
	chanCreationPolicyOutcome := make(chan creationPolicyOutCome, len(cps))

	for i, tfCp := range tfCreationPolicies {
		cpType, err := getMapValueAsString(tfCp, "type")
		if nil != err {
			return errors.Wrap(err, "type must be specified")
		}
		construtor, ok := creationPolicyConstructors[cpType]
		if !ok {
			return errors.Errorf(
				"CreationPolicy type[%s] not supported yet. supported are:%s, tfCp=%v",
				cpType, supporttedCreationPolicyTypes(), tfCp)
		}

		key := cpType + "_" + strconv.FormatInt(int64(i), 10)
		cp, err := construtor(key, vm, tfCp, &wg, chanCreationPolicyOutcome)
		if nil != err {
			return errors.Wrapf(err, "Failed to construct creation-policy[%s], tf spec:%v", key, tfCp)
		}
		cps[key] = cp
	}
	wg.Add(len(cps))
	//
	//cancelAllOnAnyTimeout (on timeout signal cancel to all )
	//cancelAllCheckOnAnyTimeout
	go func() {
		for {
			q, ok := <-chanCreationPolicyOutcome
			if !ok {
				//issue with waite group?
				return
			}
			cp := cps[q.key]
			cp.outcome = q
			if q.inFailureState() {
				log.Printf("CreatePolicy Failed: %v\n", q)
				cp.cancel(fmt.Sprintf("canceled because %s has failed: state %s", q.key, q.state))
			}
			if q.inTerminalState() {
				log.Printf("Done amid terminal state of: %v\n", q)
				wg.Done()
			}
		}
	}()
	for _, cp := range cps {
		cp.startCheckingFunc()
	}
	//safeguarding by canelling all Creation Policy check
	//Just using the cancel channel will not work if a check does not react,
	// appropriately on cancel event
	// maybe the timeout callback can ensure itself hat the wait condition reaches 0
	maxCpTimeout := maxTimeout(cps)
	maxCpTimeout = maxCpTimeout + 2*time.Second
	ubertimeoutAfterFunc := time.AfterFunc(maxCpTimeout, func() {
		for _, cp := range cps {
			log.Println("Ubertimeout Cancelling all CreationPolicy checks")
			cp.cancel("Uber-Timeout")
		}
	})

	wg.Wait()
	ubertimeoutAfterFunc.Stop()
	for _, cp := range cps {

		if !cp.outcome.inTerminalState() {
			return fmt.Errorf("CreationPolicy not expected to be in a terminal state: policy=%s state:%#v", cp.key, cp.outcome)
		}
		if cp.outcome.inFailureState() {
			return fmt.Errorf("CreationPolicy Failure: policy=%s state:%v", cp.key, cp.outcome)
		}

	}
	return nil
}

type Check interface {
	Check(attr string, err error) *creationPolicyOutCome
}

// type CheckResult interface {
// 	Continue() bool
// 	Check(attr string, err error) *CheckResult
// }

// waitstate: checking, canceled, done, timeout
// cancelChannel
// checkPassed
// failureType (toomany errors, check not passed/failed )

type waitState string

const (
	waitStateChecking = waitState("checking")
	waitStateCanceled = waitState("canceled")
	waitStateEnded    = waitState("ended")
	waiteStateTimeout = waitState("timeout")
)

type creationPolicy struct {
	key               string
	timeout           time.Duration
	cptype            string
	cancelOnce        *sync.Once
	cancelChan        chan string
	startCheckingFunc func()
	outcome           creationPolicyOutCome
}

func (cp creationPolicy) cancel(cancelMessage string) {
	cp.cancelOnce.Do(func() {
		cp.cancelChan <- cancelMessage
		close(cp.cancelChan)
	})
}

type creationPolicyOutCome struct {
	key    string
	state  waitState
	passed bool
	detail string
}

func (o creationPolicyOutCome) inFailureState() bool {
	switch o.state {
	case waitStateCanceled, waiteStateTimeout:
		return true //is canceled a failure state
	case waitStateEnded:
		return !o.passed
	default:
		return false
	}
}

func (o creationPolicyOutCome) inTerminalState() bool {
	switch o.state {
	case waitStateCanceled, waitStateEnded, waiteStateTimeout:
		return true
	default:
		return false
	}

}

func waitGuestProperties(cpKey, vm string, name string, cancelChan chan string, wg *sync.WaitGroup,
	timeoutDuration time.Duration, chanCreationPolicyOutcome chan creationPolicyOutCome,
	check func(value string, err error) *creationPolicyOutCome) {

	//once := sync.Once{}

	//props := make(chan vbox.GuestProperty)
	//wg.Add(1)
	timeoutChannel := make(chan string)
	timeoutAfterFunc := time.AfterFunc(timeoutDuration, func() {
		log.Printf("Timeout fot GuestPolicy %s\n", cpKey)
		timeoutChannel <- "timeout-" + cpKey
	})

	ticksNanos := timeoutDuration.Nanoseconds() / 200
	if ticksNanos < int64(100*time.Millisecond) {
		ticksNanos = int64(100 * time.Millisecond)
	}

	ticksDuration := time.Duration(ticksNanos)

	go func() {
		defer timeoutAfterFunc.Stop()

		for {
			value, err := vbox.GetGuestProperty(vm, name)
			outcome := check(value, err)
			if outcome.inTerminalState() {
				log.Printf("Terminal State given guest property:%s %s, %v", name, value, err)
				chanCreationPolicyOutcome <- *outcome
				return
			}
			log.Printf(
				"Continuing getting guest property:%s %s, \n\toutcome=%#v \n\terr=%v",
				name, value, outcome, err)

			select {
			case event := <-timeoutChannel:
				chanCreationPolicyOutcome <- creationPolicyOutCome{
					key:    cpKey,
					state:  waiteStateTimeout,
					detail: event,
				}
				return
			case event := <-cancelChan:
				vbox.Debug("WaitGetProperties(): done channel closed")
				chanCreationPolicyOutcome <- creationPolicyOutCome{
					key:    cpKey,
					state:  waitStateCanceled,
					detail: event,
				}
				return
			default:
				vbox.Debug("Sleeping: ticksDurationMillis=%d", ticksDuration.Milliseconds())
				time.Sleep(ticksDuration)
			}
		}
	}()

}
