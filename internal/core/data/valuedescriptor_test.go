package data

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/edgexfoundry/go-mod-core-contracts/clients/logger"
	"github.com/edgexfoundry/go-mod-core-contracts/clients/types"
	"github.com/edgexfoundry/go-mod-core-contracts/models"

	"github.com/edgexfoundry/edgex-go/internal/core/data/errors"
	"github.com/edgexfoundry/edgex-go/internal/core/data/interfaces/mocks"
	"github.com/edgexfoundry/edgex-go/internal/pkg/db"

	"github.com/stretchr/testify/mock"
)

func TestValidateFormatString(t *testing.T) {
	err := validateFormatString(models.ValueDescriptor{Formatting: "%s"}, logger.NewMockClient())

	if err != nil {
		t.Errorf("Should match format specifier")
	}
}

func TestValidateFormatStringEmpty(t *testing.T) {
	err := validateFormatString(models.ValueDescriptor{Formatting: ""}, logger.NewMockClient())

	if err != nil {
		t.Errorf("Should match format specifier")
	}
}

func TestValidateFormatStringInvalid(t *testing.T) {
	err := validateFormatString(models.ValueDescriptor{Formatting: "error"}, logger.NewMockClient())

	if err == nil {
		t.Errorf("Expected error on invalid format string")
	}
}

func TestGetValueDescriptorByName(t *testing.T) {
	reset()
	myMock := &mocks.DBClient{}

	myMock.On("ValueDescriptorByName", mock.Anything).Return(models.ValueDescriptor{Id: testUUIDString}, nil)

	dbClient = myMock

	valueDescriptor, err := getValueDescriptorByName("valid", logger.NewMockClient())

	if err != nil {
		t.Errorf("Unexpected error getting value descriptor by name")
	}

	if valueDescriptor.Id != testUUIDString {
		t.Errorf("ID returned doesn't match db")
	}
}

func TestGetValueDescriptorByNameNotFound(t *testing.T) {
	reset()
	myMock := &mocks.DBClient{}

	myMock.On("ValueDescriptorByName", mock.Anything).Return(models.ValueDescriptor{}, db.ErrNotFound)

	dbClient = myMock

	_, err := getValueDescriptorByName("404", logger.NewMockClient())

	if err != nil {
		switch err.(type) {
		case errors.ErrDbNotFound:
			return
		default:
			t.Errorf("Unexpected error getting value descriptor by name missing in DB")
		}
	}

	if err == nil {
		t.Errorf("Expected error getting value descriptor by name missing in DB")
	}
}

func TestGetValueDescriptorByNameError(t *testing.T) {
	reset()
	myMock := &mocks.DBClient{}

	myMock.On("ValueDescriptorByName", mock.Anything).Return(models.ValueDescriptor{}, fmt.Errorf("some error"))

	dbClient = myMock

	_, err := getValueDescriptorByName("error", logger.NewMockClient())

	if err == nil {
		t.Errorf("Expected error getting value descriptor by name with some error")
	}
}

func TestGetValueDescriptorById(t *testing.T) {
	reset()
	myMock := &mocks.DBClient{}

	myMock.On("ValueDescriptorById", mock.Anything).Return(models.ValueDescriptor{Id: testUUIDString}, nil)

	dbClient = myMock

	valueDescriptor, err := getValueDescriptorById("valid", logger.NewMockClient())

	if err != nil {
		t.Errorf("Unexpected error getting value descriptor by ID")
	}

	if valueDescriptor.Id != testUUIDString {
		t.Errorf("ID returned doesn't match db")
	}
}

func TestGetValueDescriptorByIdNotFound(t *testing.T) {
	reset()
	myMock := &mocks.DBClient{}

	myMock.On("ValueDescriptorById", mock.Anything).Return(models.ValueDescriptor{}, db.ErrNotFound)

	dbClient = myMock

	_, err := getValueDescriptorById("404", logger.NewMockClient())

	if err != nil {
		switch err.(type) {
		case errors.ErrDbNotFound:
			return
		default:
			t.Errorf("Unexpected error getting value descriptor by ID missing in DB")
		}
	}

	if err == nil {
		t.Errorf("Expected error getting value descriptor by ID missing in DB")
	}
}

func TestGetValueDescriptorByIdError(t *testing.T) {
	reset()
	myMock := &mocks.DBClient{}

	myMock.On("ValueDescriptorById", mock.Anything).Return(models.ValueDescriptor{}, fmt.Errorf("some error"))

	dbClient = myMock

	_, err := getValueDescriptorById("error", logger.NewMockClient())

	if err == nil {
		t.Errorf("Expected error getting value descriptor by ID with some error")
	}
}

func TestGetValueDescriptorsByUomLabel(t *testing.T) {
	reset()
	myMock := &mocks.DBClient{}

	myMock.On("ValueDescriptorsByUomLabel", mock.Anything).Return([]models.ValueDescriptor{}, nil)

	dbClient = myMock

	_, err := getValueDescriptorsByUomLabel("valid", logger.NewMockClient())

	if err != nil {
		t.Errorf("Unexpected error getting value descriptor by UOM label")
	}
}

func TestGetValueDescriptorsByUomLabelNotFound(t *testing.T) {
	reset()
	myMock := &mocks.DBClient{}

	myMock.On("ValueDescriptorsByUomLabel", mock.Anything).Return([]models.ValueDescriptor{}, db.ErrNotFound)

	dbClient = myMock

	_, err := getValueDescriptorsByUomLabel("404", logger.NewMockClient())

	if err != nil {
		switch err.(type) {
		case errors.ErrDbNotFound:
			return
		default:
			t.Errorf("Unexpected error getting value descriptor by UOM label missing in DB")
		}
	}

	if err == nil {
		t.Errorf("Expected error getting value descriptor by UOM label missing in DB")
	}
}

func TestGetValueDescriptorsByUomLabelError(t *testing.T) {
	reset()
	myMock := &mocks.DBClient{}

	myMock.On("ValueDescriptorsByUomLabel", mock.Anything).Return([]models.ValueDescriptor{}, fmt.Errorf("some error"))

	dbClient = myMock

	_, err := getValueDescriptorsByUomLabel("error", logger.NewMockClient())

	if err == nil {
		t.Errorf("Expected error getting value descriptor by UOM label with some error")
	}
}

func TestGetValueDescriptorsByLabel(t *testing.T) {
	reset()
	myMock := &mocks.DBClient{}

	myMock.On("ValueDescriptorsByLabel", mock.MatchedBy(func(name string) bool {
		return name == testUUIDString
	})).Return([]models.ValueDescriptor{{Id: testUUIDString}}, nil)

	dbClient = myMock

	valueDescriptor, err := getValueDescriptorsByLabel(testUUIDString, logger.NewMockClient())

	if err != nil {
		t.Errorf("Unexpected error getting value descriptor by label")
	}

	if valueDescriptor[0].Id != testUUIDString {
		t.Errorf("ValueDescriptor received doesn't match expectation")
	}
}

func TestGetValueDescriptorsByLabelNotFound(t *testing.T) {
	reset()
	myMock := &mocks.DBClient{}

	myMock.On("ValueDescriptorsByLabel", mock.Anything).Return([]models.ValueDescriptor{}, db.ErrNotFound)

	dbClient = myMock

	_, err := getValueDescriptorsByLabel("404", logger.NewMockClient())

	if err != nil {
		switch err.(type) {
		case errors.ErrDbNotFound:
			return
		default:
			t.Errorf("Unexpected error getting value descriptor by label missing in DB")
		}
	}

	if err == nil {
		t.Errorf("Expected error getting value descriptor by label missing in DB")
	}
}

func TestGetValueDescriptorsByLabelError(t *testing.T) {
	reset()
	myMock := &mocks.DBClient{}

	myMock.On("ValueDescriptorsByLabel", mock.Anything).Return([]models.ValueDescriptor{}, fmt.Errorf("some error"))

	dbClient = myMock

	_, err := getValueDescriptorsByLabel("error", logger.NewMockClient())

	if err == nil {
		t.Errorf("Expected error getting value descriptor by label with some error")
	}
}

func TestGetValueDescriptorsByType(t *testing.T) {
	reset()
	myMock := &mocks.DBClient{}

	myMock.On("ValueDescriptorsByType", mock.Anything).Return([]models.ValueDescriptor{}, nil)

	dbClient = myMock

	_, err := getValueDescriptorsByType("valid", logger.NewMockClient())

	if err != nil {
		t.Errorf("Unexpected error getting value descriptor by type")
	}
}

func TestGetValueDescriptorsByTypeNotFound(t *testing.T) {
	reset()
	myMock := &mocks.DBClient{}

	myMock.On("ValueDescriptorsByType", mock.Anything).Return([]models.ValueDescriptor{}, db.ErrNotFound)

	dbClient = myMock

	_, err := getValueDescriptorsByType("404", logger.NewMockClient())

	if err != nil {
		switch err.(type) {
		case errors.ErrDbNotFound:
			return
		default:
			t.Errorf("Unexpected error getting value descriptor by type missing in DB")
		}
	}

	if err == nil {
		t.Errorf("Expected error getting value descriptor by type missing in DB")
	}
}

func TestGetValueDescriptorsByTypeError(t *testing.T) {
	reset()
	myMock := &mocks.DBClient{}

	myMock.On("ValueDescriptorsByType", mock.Anything).Return([]models.ValueDescriptor{}, fmt.Errorf("some error"))

	dbClient = myMock
	mdc = newMockDeviceClient()

	_, err := getValueDescriptorsByType("R", logger.NewMockClient())

	if err == nil {
		t.Errorf("Expected error getting value descriptor by type with some error")
	}
}

func TestGetValueDescriptorsByDeviceName(t *testing.T) {
	reset()
	dbClient = nil

	_, err := getValueDescriptorsByDeviceName(testDeviceName, context.Background(), logger.NewMockClient())

	if err != nil {
		t.Errorf("Unexpected error getting value descriptor by device name")
	}
}

func TestGetValueDescriptorsByDeviceNameNotFound(t *testing.T) {
	reset()
	dbClient = nil

	_, err := getValueDescriptorsByDeviceName("404", context.Background(), logger.NewMockClient())

	if err != nil {
		switch err := err.(type) {
		case types.ErrServiceClient:
			if err.StatusCode != http.StatusNotFound {
				t.Errorf("Expected a 404 error")
			}
			return
		default:
			t.Errorf("Unexpected error getting value descriptor by device name missing in DB")
		}
	}

	if err == nil {
		t.Errorf("Expected error getting value descriptor by device name missing in DB")
	}
}

func TestGetValueDescriptorsByDeviceNameError(t *testing.T) {
	reset()
	dbClient = nil

	_, err := getValueDescriptorsByDeviceName("error", context.Background(), logger.NewMockClient())

	if err == nil {
		t.Errorf("Expected error getting value descriptor by device name with some error")
	}
}

func TestGetValueDescriptorsByDeviceId(t *testing.T) {
	reset()
	dbClient = nil

	_, err := getValueDescriptorsByDeviceId("valid", context.Background(), logger.NewMockClient())

	if err != nil {
		t.Errorf("Unexpected error getting value descriptor by device id")
	}
}

func TestGetValueDescriptorsByDeviceIdNotFound(t *testing.T) {
	reset()
	dbClient = nil

	_, err := getValueDescriptorsByDeviceId("404", context.Background(), logger.NewMockClient())

	if err != nil {
		switch err := err.(type) {
		case types.ErrServiceClient:
			if err.StatusCode != http.StatusNotFound {
				t.Errorf("Expected a 404 error")
			}
			return
		default:
			t.Errorf("Unexpected error getting value descriptor by device id missing in DB")
		}
	}

	if err == nil {
		t.Errorf("Expected error getting value descriptor by device name missing in DB")
	}
}

func TestGetValueDescriptorsByDeviceIdError(t *testing.T) {
	reset()
	dbClient = nil

	_, err := getValueDescriptorsByDeviceId("error", context.Background(), logger.NewMockClient())

	if err == nil {
		t.Errorf("Expected error getting value descriptor by device id with some error")
	}
}

func TestGetAllValueDescriptors(t *testing.T) {
	reset()

	vds := []models.ValueDescriptor{
		{Id: testUUIDString},
		{Id: testBsonString},
	}

	myMock := &mocks.DBClient{}
	myMock.On("ValueDescriptors").Return(vds, nil)
	dbClient = myMock

	_, err := getAllValueDescriptors(logger.NewMockClient())

	if err != nil {
		t.Errorf("Unexpected error getting all value descriptors")
	}
}

func TestGetAllValueDescriptorsError(t *testing.T) {
	reset()
	myMock := &mocks.DBClient{}

	myMock.On("ValueDescriptors").Return([]models.ValueDescriptor{}, fmt.Errorf("some error"))

	dbClient = myMock

	_, err := getAllValueDescriptors(logger.NewMockClient())

	if err == nil {
		t.Errorf("Expected error getting all value descriptors some error")
	}
}

func TestAddValueDescriptor(t *testing.T) {
	reset()
	myMock := &mocks.DBClient{}

	myMock.On("AddValueDescriptor", mock.Anything).Return("", nil)

	dbClient = myMock

	_, err := addValueDescriptor(models.ValueDescriptor{Name: "valid"}, logger.NewMockClient())

	if err != nil {
		t.Errorf("Unexpected error adding value descriptor")
	}
}

func TestAddDuplicateValueDescriptor(t *testing.T) {
	reset()
	myMock := &mocks.DBClient{}

	myMock.On("AddValueDescriptor", mock.Anything).Return("", db.ErrNotUnique)

	dbClient = myMock

	_, err := addValueDescriptor(models.ValueDescriptor{Name: "409"}, logger.NewMockClient())

	if err != nil {
		switch err.(type) {
		case errors.ErrDuplicateValueDescriptorName:
			return
		default:
			t.Errorf("Unexpected error adding value descriptor that already exists")
		}
	}

	if err == nil {
		t.Errorf("Expected error adding value descriptor that already exists")
	}
}

func TestAddValueDescriptorError(t *testing.T) {
	reset()
	myMock := &mocks.DBClient{}

	myMock.On("AddValueDescriptor", mock.Anything).Return("", fmt.Errorf("some error"))

	dbClient = myMock

	_, err := addValueDescriptor(models.ValueDescriptor{}, logger.NewMockClient())

	if err == nil {
		t.Errorf("Expected error adding value descriptor some error")
	}
}

func TestDeleteValueDescriptor(t *testing.T) {
	reset()
	myMock := &mocks.DBClient{}

	myMock.On("DeleteValueDescriptorById", mock.Anything).Return(nil)
	myMock.On("ReadingsByValueDescriptor", mock.Anything, mock.Anything).Return([]models.Reading{}, nil)

	dbClient = myMock

	err := deleteValueDescriptor(models.ValueDescriptor{Name: "valid", Id: testBsonString}, logger.NewMockClient())

	if err != nil {
		t.Errorf("Unexpected error deleting value descriptor")
	}
}

func TestDeleteValueDescriptorInUse(t *testing.T) {
	reset()
	myMock := &mocks.DBClient{}

	myMock.On("ReadingsByValueDescriptor", mock.Anything, mock.Anything).Return([]models.Reading{{Id: testUUIDString}}, nil)

	dbClient = myMock

	err := deleteValueDescriptor(models.ValueDescriptor{Name: "409"}, logger.NewMockClient())

	if err != nil {
		switch err.(type) {
		case errors.ErrValueDescriptorInUse:
			return
		default:
			t.Errorf("Unexpected error deleting value descriptor in use")
		}
	}

	if err == nil {
		t.Errorf("Expected error deleting value descriptor in use")
	}
}

func TestDeleteValueDescriptorErrorReadingsLookup(t *testing.T) {
	reset()
	myMock := &mocks.DBClient{}

	myMock.On("ReadingsByValueDescriptor", mock.Anything, mock.Anything).Return([]models.Reading{}, fmt.Errorf("some error"))

	dbClient = myMock

	err := deleteValueDescriptor(models.ValueDescriptor{}, logger.NewMockClient())

	if err == nil {
		t.Errorf("Expected error deleting value descriptor some error looking up readings")
	}
}

func TestDeleteValueDescriptorError(t *testing.T) {
	reset()
	myMock := &mocks.DBClient{}

	myMock.On("ReadingsByValueDescriptor", mock.Anything, mock.Anything).Return([]models.Reading{}, nil)
	myMock.On("DeleteValueDescriptorById", mock.Anything).Return(fmt.Errorf("some error"))

	dbClient = myMock

	err := deleteValueDescriptor(models.ValueDescriptor{Name: "validErrorTest"}, logger.NewMockClient())

	if err == nil {
		t.Errorf("Expected error deleting value descriptor some error")
	}
}

type closingBuffer struct {
	*bytes.Buffer
}

func (cb *closingBuffer) Close() (err error) {
	return nil
}
