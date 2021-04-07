// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
package main

import (
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/helper/logging"
	"github.com/hashicorp/terraform-plugin-sdk/plugin"
	vbox "github.com/terra-farm/go-virtualbox"
	"github.com/terra-farm/terraform-provider-virtualbox/virtualbox"
)

func main() {
	if logging.IsDebugOrHigher() {
		vbox.Verbose = true
		vbox.Debug = log.Printf
	}
	logging.SetOutput()
	plugin.Serve(&plugin.ServeOpts{
		ProviderFunc: virtualbox.Provider,
	})
}
