package controller

import (
	"kubevirt-image-service/pkg/controller/localuploadproxy"
)

func init() {
	// AddToManagerFuncs is a list of functions to create controllers and add them to a manager.
	AddToManagerFuncs = append(AddToManagerFuncs, localuploadproxy.Add)
}
