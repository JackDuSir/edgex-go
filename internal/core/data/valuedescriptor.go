/*******************************************************************************
 * Copyright 2017 Dell Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file except
 * in compliance with the License. You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software distributed under the License
 * is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express
 * or implied. See the License for the specific language governing permissions and limitations under
 * the License.
 *******************************************************************************/
package data

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"regexp"

	"github.com/edgexfoundry/go-mod-core-contracts/clients/logger"
	contract "github.com/edgexfoundry/go-mod-core-contracts/models"

	"github.com/edgexfoundry/edgex-go/internal/core/data/errors"
	"github.com/edgexfoundry/edgex-go/internal/pkg/db"
)

const (
	formatSpecifier          = "%(\\d+\\$)?([-#+ 0,(\\<]*)?(\\d+)?(\\.\\d+)?([tT])?([a-zA-Z%])"
	maxExceededString string = "Error, exceeded the max limit as defined in config"
)

// Check if the value descriptor matches the format string regular expression
func validateFormatString(v contract.ValueDescriptor, loggingClient logger.LoggingClient) error {
	// No formatting specified
	if v.Formatting == "" {
		return nil
	}

	match, err := regexp.MatchString(formatSpecifier, v.Formatting)

	if err != nil {
		loggingClient.Error("Error checking for format string for value descriptor " + v.Name)
		return err
	}
	if !match {
		err = fmt.Errorf("format is not a valid printf format")
		loggingClient.Error(fmt.Sprintf("Error posting value descriptor. %s", err.Error()))
		return errors.NewErrValueDescriptorInvalid(v.Name, err)
	}

	return nil
}

func getValueDescriptorByName(name string, loggingClient logger.LoggingClient) (vd contract.ValueDescriptor, err error) {
	vd, err = dbClient.ValueDescriptorByName(name)

	if err != nil {
		loggingClient.Error(err.Error())
		if err == db.ErrNotFound {
			return contract.ValueDescriptor{}, errors.NewErrDbNotFound()
		} else {
			return contract.ValueDescriptor{}, err
		}
	}

	return vd, nil
}

func getValueDescriptorById(id string, loggingClient logger.LoggingClient) (vd contract.ValueDescriptor, err error) {
	vd, err = dbClient.ValueDescriptorById(id)

	if err != nil {
		loggingClient.Error(err.Error())
		if err == db.ErrNotFound {
			return contract.ValueDescriptor{}, errors.NewErrDbNotFound()
		} else if err == db.ErrInvalidObjectId {
			return contract.ValueDescriptor{}, errors.NewErrInvalidId(id)
		} else {
			return contract.ValueDescriptor{}, err
		}
	}

	return vd, nil
}

func getValueDescriptorsByUomLabel(uomLabel string, loggingClient logger.LoggingClient) (vdList []contract.ValueDescriptor, err error) {
	vdList, err = dbClient.ValueDescriptorsByUomLabel(uomLabel)

	if err != nil {
		loggingClient.Error(err.Error())
		if err == db.ErrNotFound {
			return []contract.ValueDescriptor{}, errors.NewErrDbNotFound()
		} else {
			return []contract.ValueDescriptor{}, err
		}
	}

	return vdList, nil
}

func getValueDescriptorsByLabel(label string, loggingClient logger.LoggingClient) (vdList []contract.ValueDescriptor, err error) {
	vdList, err = dbClient.ValueDescriptorsByLabel(label)

	if err != nil {
		loggingClient.Error(err.Error())
		if err == db.ErrNotFound {
			return []contract.ValueDescriptor{}, errors.NewErrDbNotFound()
		} else {
			return []contract.ValueDescriptor{}, err
		}
	}

	return vdList, nil
}

func getValueDescriptorsByType(typ string, loggingClient logger.LoggingClient) (vdList []contract.ValueDescriptor, err error) {
	vdList, err = dbClient.ValueDescriptorsByType(typ)

	if err != nil {
		loggingClient.Error(err.Error())
		if err == db.ErrNotFound {
			return []contract.ValueDescriptor{}, errors.NewErrDbNotFound()
		} else {
			return []contract.ValueDescriptor{}, err
		}
	}

	return vdList, nil
}

func getValueDescriptorsByDevice(device contract.Device, loggingClient logger.LoggingClient) (vdList []contract.ValueDescriptor, err error) {
	// Get the names of the value descriptors
	vdNames := []string{}
	device.AllAssociatedValueDescriptors(&vdNames)

	// Get the value descriptors
	vdList = []contract.ValueDescriptor{}
	for _, name := range vdNames {
		vd, err := getValueDescriptorByName(name, loggingClient)

		// Not an error if not found
		if err != nil {
			switch err.(type) {
			case errors.ErrDbNotFound:
				continue
			default:
				return []contract.ValueDescriptor{}, err
			}
		}

		vdList = append(vdList, vd)
	}

	return vdList, nil
}

func getValueDescriptorsByDeviceName(name string, ctx context.Context, loggingClient logger.LoggingClient) (vdList []contract.ValueDescriptor, err error) {
	// Get the device
	device, err := mdc.DeviceForName(name, ctx)
	if err != nil {
		loggingClient.Error("Problem getting device from metadata: " + err.Error())
		return []contract.ValueDescriptor{}, err
	}

	return getValueDescriptorsByDevice(device, loggingClient)
}

func getValueDescriptorsByDeviceId(id string, ctx context.Context, loggingClient logger.LoggingClient) (vdList []contract.ValueDescriptor, err error) {
	// Get the device
	device, err := mdc.Device(id, ctx)
	if err != nil {
		loggingClient.Error("Problem getting device from metadata: " + err.Error())
		return []contract.ValueDescriptor{}, err
	}

	return getValueDescriptorsByDevice(device, loggingClient)
}

func getAllValueDescriptors(loggingClient logger.LoggingClient) (vd []contract.ValueDescriptor, err error) {
	vd, err = dbClient.ValueDescriptors()
	if err != nil {
		loggingClient.Error(err.Error())
		return nil, err
	}

	return vd, nil
}

func decodeValueDescriptor(reader io.ReadCloser, loggingClient logger.LoggingClient) (vd contract.ValueDescriptor, err error) {
	v := contract.ValueDescriptor{}
	err = json.NewDecoder(reader).Decode(&v)
	// Problems decoding
	if err != nil {
		loggingClient.Error("Error decoding the value descriptor: " + err.Error())
		return contract.ValueDescriptor{}, errors.NewErrJsonDecoding(v.Name)
	}

	// Check the formatting
	err = validateFormatString(v, loggingClient)
	if err != nil {
		return contract.ValueDescriptor{}, err
	}

	return v, nil
}

func addValueDescriptor(vd contract.ValueDescriptor, loggingClient logger.LoggingClient) (id string, err error) {
	id, err = dbClient.AddValueDescriptor(vd)
	if err != nil {
		loggingClient.Error(err.Error())
		if err == db.ErrNotUnique {
			return "", errors.NewErrDuplicateValueDescriptorName(vd.Name)
		} else {
			return "", err
		}
	}

	return id, nil
}

func updateValueDescriptor(from contract.ValueDescriptor, loggingClient logger.LoggingClient) error {
	to, err := getValueDescriptorById(from.Id, loggingClient)
	if err != nil {
		return err
	}

	// Update the fields
	if from.Description != "" {
		to.Description = from.Description
	}
	if from.DefaultValue != "" {
		to.DefaultValue = from.DefaultValue
	}
	if from.Formatting != "" {
		match, err := regexp.MatchString(formatSpecifier, from.Formatting)
		if err != nil {
			loggingClient.Error("Error checking formatting for updated value descriptor")
			return err
		}
		if !match {
			loggingClient.Error("value descriptor's format string doesn't fit the required pattern: " + formatSpecifier)
			return errors.NewErrValueDescriptorInvalid(from.Name, err)
		}
		to.Formatting = from.Formatting
	}
	if from.Labels != nil {
		to.Labels = from.Labels
	}

	if from.Max != "" {
		to.Max = from.Max
	}
	if from.Min != "" {
		to.Min = from.Min
	}
	if from.Name != "" {
		// Check if value descriptor is still in use by readings if the name changes
		if from.Name != to.Name {
			r, err := getReadingsByValueDescriptor(to.Name, 10, loggingClient) // Arbitrary limit, we're just checking if there are any readings
			if err != nil {
				loggingClient.Error("Error checking the readings for the value descriptor: " + err.Error())
				return err
			}
			// Value descriptor is still in use
			if len(r) != 0 {
				loggingClient.Error("Data integrity issue.  Value Descriptor with name:  " + from.Name + " is still referenced by existing readings.")
				return errors.NewErrValueDescriptorInUse(from.Name)
			}
		}
		to.Name = from.Name
	}
	if from.Origin != 0 {
		to.Origin = from.Origin
	}
	if from.Type != "" {
		to.Type = from.Type
	}
	if from.UomLabel != "" {
		to.UomLabel = from.UomLabel
	}

	// Push the updated valuedescriptor to the database
	err = dbClient.UpdateValueDescriptor(to)
	if err != nil {
		if err == db.ErrNotUnique {
			loggingClient.Error("Value descriptor name is not unique")
			return errors.NewErrDuplicateValueDescriptorName(to.Name)
		} else {
			loggingClient.Error(err.Error())
			return err
		}
	}

	return nil
}

func deleteValueDescriptor(vd contract.ValueDescriptor, loggingClient logger.LoggingClient) error {
	// Check if the value descriptor is still in use by readings
	readings, err := dbClient.ReadingsByValueDescriptor(vd.Name, 10)
	if err != nil {
		loggingClient.Error(err.Error())
		return err
	}
	if len(readings) > 0 {
		loggingClient.Error("Data integrity issue.  Value Descriptor is still referenced by existing readings.")
		return errors.NewErrValueDescriptorInUse(vd.Name)
	}

	// Delete the value descriptor
	if err = dbClient.DeleteValueDescriptorById(vd.Id); err != nil {
		loggingClient.Error(err.Error())
		return err
	}

	return nil
}

func deleteValueDescriptorByName(name string, loggingClient logger.LoggingClient) error {
	// Check if the value descriptor exists
	vd, err := getValueDescriptorByName(name, loggingClient)
	if err != nil {
		return err
	}

	if err = deleteValueDescriptor(vd, loggingClient); err != nil {
		return err
	}

	return nil
}

func deleteValueDescriptorById(id string, loggingClient logger.LoggingClient) error {
	// Check if the value descriptor exists
	vd, err := getValueDescriptorById(id, loggingClient)
	if err != nil {
		return err
	}

	if err = deleteValueDescriptor(vd, loggingClient); err != nil {
		return err
	}

	return nil
}
