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
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/edgexfoundry/go-mod-core-contracts/clients"
	"github.com/edgexfoundry/go-mod-core-contracts/clients/logger"
	contract "github.com/edgexfoundry/go-mod-core-contracts/models"
	"github.com/gorilla/mux"

	"github.com/edgexfoundry/edgex-go/internal/core/data/errors"
	"github.com/edgexfoundry/edgex-go/internal/core/data/operators/reading"
	"github.com/edgexfoundry/edgex-go/internal/core/data/operators/value_descriptor"
	"github.com/edgexfoundry/edgex-go/internal/pkg"
	"github.com/edgexfoundry/edgex-go/internal/pkg/bootstrap/container"
	"github.com/edgexfoundry/edgex-go/internal/pkg/correlation"
	"github.com/edgexfoundry/edgex-go/internal/pkg/correlation/models"
	"github.com/edgexfoundry/edgex-go/internal/pkg/di"
	"github.com/edgexfoundry/edgex-go/internal/pkg/errorconcept"
	"github.com/edgexfoundry/edgex-go/internal/pkg/telemetry"
)

// ValueDescriptorUsageReadLimit limit of readings obtained for a given value descriptor to determine if the value
// descriptor is in use.
var ValueDescriptorUsageReadLimit = 1

func LoadRestRoutes(dic *di.Container) *mux.Router {
	r := mux.NewRouter()

	// Ping Resource
	r.HandleFunc(clients.ApiPingRoute, pingHandler).Methods(http.MethodGet)

	// Configuration
	r.HandleFunc(clients.ApiConfigRoute, func(writer http.ResponseWriter, request *http.Request) {
		configHandler(writer, request, container.LoggingClientFrom(dic.Get))
	}).Methods(http.MethodGet)

	// Metrics
	r.HandleFunc(clients.ApiMetricsRoute, func(writer http.ResponseWriter, request *http.Request) {
		metricsHandler(writer, request, container.LoggingClientFrom(dic.Get))
	}).Methods(http.MethodGet)

	// Version
	r.HandleFunc(clients.ApiVersionRoute, pkg.VersionHandler).Methods(http.MethodGet)

	// Events
	r.HandleFunc(clients.ApiEventRoute, func(writer http.ResponseWriter, request *http.Request) {
		eventHandler(writer, request, container.LoggingClientFrom(dic.Get))
	}).Methods(http.MethodGet, http.MethodPut, http.MethodPost)

	e := r.PathPrefix(clients.ApiEventRoute).Subrouter()

	e.HandleFunc("/"+SCRUB, func(writer http.ResponseWriter, request *http.Request) {
		scrubHandler(writer, request, container.LoggingClientFrom(dic.Get))
	}).Methods(http.MethodDelete)

	e.HandleFunc("/"+SCRUBALL, func(writer http.ResponseWriter, request *http.Request) {
		scrubAllHandler(writer, request, container.LoggingClientFrom(dic.Get))
	}).Methods(http.MethodDelete)

	e.HandleFunc("/"+COUNT, func(writer http.ResponseWriter, request *http.Request) {
		eventCountHandler(writer, request, container.LoggingClientFrom(dic.Get))
	}).Methods(http.MethodGet)

	e.HandleFunc("/"+COUNT+"/{"+DEVICEID_PARAM+"}", eventCountByDeviceIdHandler).Methods(http.MethodGet)

	e.HandleFunc("/{"+ID+"}", func(writer http.ResponseWriter, request *http.Request) {
		getEventByIdHandler(writer, request, container.LoggingClientFrom(dic.Get))
	}).Methods(http.MethodGet)

	e.HandleFunc("/"+ID+"/{"+ID+"}", func(writer http.ResponseWriter, request *http.Request) {
		eventIdHandler(writer, request, container.LoggingClientFrom(dic.Get))
	}).Methods(http.MethodDelete, http.MethodPut)

	e.HandleFunc("/"+CHECKSUM+"/{"+CHECKSUM+"}", func(writer http.ResponseWriter, request *http.Request) {
		putEventChecksumHandler(writer, request, container.LoggingClientFrom(dic.Get))
	}).Methods(http.MethodPut)

	e.HandleFunc(
		"/"+DEVICE+"/{"+DEVICEID_PARAM+"}/{"+LIMIT+":[0-9]+}",
		func(writer http.ResponseWriter, request *http.Request) {
			getEventByDeviceHandler(writer, request, container.LoggingClientFrom(dic.Get))
		}).Methods(http.MethodGet)
	e.HandleFunc("/"+DEVICE+"/{"+DEVICEID_PARAM+"}", deleteByDeviceIdHandler).Methods(http.MethodDelete)

	e.HandleFunc("/"+REMOVEOLD+"/"+AGE+"/{"+AGE+":[0-9]+}", func(writer http.ResponseWriter, request *http.Request) {
		eventByAgeHandler(writer, request, container.LoggingClientFrom(dic.Get))
	}).Methods(http.MethodDelete)

	e.HandleFunc(
		"/{"+START+":[0-9]+}/{"+END+":[0-9]+}/{"+LIMIT+":[0-9]+}",
		func(writer http.ResponseWriter, request *http.Request) {
			eventByCreationTimeHandler(writer, request, container.LoggingClientFrom(dic.Get))
		}).Methods(http.MethodGet)

	// Readings
	r.HandleFunc(clients.ApiReadingRoute, func(writer http.ResponseWriter, request *http.Request) {
		readingHandler(writer, request, container.LoggingClientFrom(dic.Get))
	}).Methods(http.MethodGet, http.MethodPut, http.MethodPost)

	rd := r.PathPrefix(clients.ApiReadingRoute).Subrouter()

	rd.HandleFunc("/"+COUNT, func(writer http.ResponseWriter, request *http.Request) {
		readingCountHandler(writer, request, container.LoggingClientFrom(dic.Get))
	}).Methods(http.MethodGet)

	rd.HandleFunc("/"+ID+"/{"+ID+"}", func(writer http.ResponseWriter, request *http.Request) {
		deleteReadingByIdHandler(writer, request, container.LoggingClientFrom(dic.Get))
	}).Methods(http.MethodDelete)

	rd.HandleFunc("/{"+ID+"}", func(writer http.ResponseWriter, request *http.Request) {
		getReadingByIdHandler(writer, request, container.LoggingClientFrom(dic.Get))
	}).Methods(http.MethodGet)

	rd.HandleFunc(
		"/"+DEVICE+"/{"+DEVICEID_PARAM+"}/{"+LIMIT+":[0-9]+}",
		func(writer http.ResponseWriter, request *http.Request) {
			readingByDeviceHandler(writer, request, container.LoggingClientFrom(dic.Get))
		}).Methods(http.MethodGet)

	rd.HandleFunc(
		"/"+NAME+"/{"+NAME+"}/{"+LIMIT+":[0-9]+}",
		func(writer http.ResponseWriter, request *http.Request) {
			readingbyValueDescriptorHandler(writer, request, container.LoggingClientFrom(dic.Get))
		}).Methods(http.MethodGet)

	rd.HandleFunc(
		"/"+UOMLABEL+"/{"+UOMLABEL_PARAM+"}/{"+LIMIT+":[0-9]+}",
		func(writer http.ResponseWriter, request *http.Request) {
			readingByUomLabelHandler(writer, request, container.LoggingClientFrom(dic.Get))
		}).Methods(http.MethodGet)

	rd.HandleFunc("/"+LABEL+"/{"+LABEL+"}/{"+LIMIT+":[0-9]+}", func(writer http.ResponseWriter, request *http.Request) {
		readingByLabelHandler(writer, request, container.LoggingClientFrom(dic.Get))
	}).Methods(http.MethodGet)

	rd.HandleFunc("/"+TYPE+"/{"+TYPE+"}/{"+LIMIT+":[0-9]+}", func(writer http.ResponseWriter, request *http.Request) {
		readingByTypeHandler(writer, request, container.LoggingClientFrom(dic.Get))
	}).Methods(http.MethodGet)

	rd.HandleFunc(
		"/{"+START+":[0-9]+}/{"+END+":[0-9]+}/{"+LIMIT+":[0-9]+}",
		func(writer http.ResponseWriter, request *http.Request) {
			readingByCreationTimeHandler(writer, request, container.LoggingClientFrom(dic.Get))
		}).Methods(http.MethodGet)

	rd.HandleFunc(
		"/"+NAME+"/{"+NAME+"}/"+DEVICE+"/{"+DEVICE+"}/{"+LIMIT+":[0-9]+}",
		func(writer http.ResponseWriter, request *http.Request) {
			readingByValueDescriptorAndDeviceHandler(writer, request, container.LoggingClientFrom(dic.Get))
		}).Methods(http.MethodGet)

	// Value descriptors
	r.HandleFunc(
		clients.ApiValueDescriptorRoute,
		func(writer http.ResponseWriter, request *http.Request) {
			valueDescriptorHandler(writer, request, container.LoggingClientFrom(dic.Get))
		}).Methods(http.MethodGet, http.MethodPut, http.MethodPost)

	vd := r.PathPrefix(clients.ApiValueDescriptorRoute).Subrouter()

	vd.HandleFunc("/"+USAGE, func(writer http.ResponseWriter, request *http.Request) {
		restValueDescriptorsUsageHandler(writer, request, container.LoggingClientFrom(dic.Get))
	}).Methods(http.MethodGet)

	vd.HandleFunc("/"+ID+"/{"+ID+"}", func(writer http.ResponseWriter, request *http.Request) {
		deleteValueDescriptorByIdHandler(writer, request, container.LoggingClientFrom(dic.Get))
	}).Methods(http.MethodDelete)

	vd.HandleFunc("/"+NAME+"/{"+NAME+"}", func(writer http.ResponseWriter, request *http.Request) {
		valueDescriptorByNameHandler(writer, request, container.LoggingClientFrom(dic.Get))
	}).Methods(http.MethodGet, http.MethodDelete)

	vd.HandleFunc("/{"+ID+"}", func(writer http.ResponseWriter, request *http.Request) {
		valueDescriptorByIdHandler(writer, request, container.LoggingClientFrom(dic.Get))
	}).Methods(http.MethodGet)

	vd.HandleFunc("/"+UOMLABEL+"/{"+UOMLABEL_PARAM+"}", func(writer http.ResponseWriter, request *http.Request) {
		valueDescriptorByUomLabelHandler(writer, request, container.LoggingClientFrom(dic.Get))
	}).Methods(http.MethodGet)

	vd.HandleFunc("/"+LABEL+"/{"+LABEL+"}", func(writer http.ResponseWriter, request *http.Request) {
		valueDescriptorByLabelHandler(writer, request, container.LoggingClientFrom(dic.Get))
	}).Methods(http.MethodGet)

	vd.HandleFunc("/"+DEVICENAME+"/{"+DEVICE+"}", func(writer http.ResponseWriter, request *http.Request) {
		valueDescriptorByDeviceHandler(writer, request, container.LoggingClientFrom(dic.Get))
	}).Methods(http.MethodGet)

	vd.HandleFunc("/"+DEVICEID+"/{"+ID+"}", func(writer http.ResponseWriter, request *http.Request) {
		valueDescriptorByDeviceIdHandler(writer, request, container.LoggingClientFrom(dic.Get))
	}).Methods(http.MethodGet)

	r.Use(correlation.ManageHeader)
	r.Use(correlation.OnResponseComplete)
	r.Use(correlation.OnRequestBegin)

	return r
}

/*
Return number of events in Core Data
/api/v1/event/count
*/
func eventCountHandler(w http.ResponseWriter, r *http.Request, loggingClient logger.LoggingClient) {
	defer r.Body.Close()

	count, err := countEvents()
	if err != nil {
		httpErrorHandler.Handle(w, err, errorconcept.Default.InternalServerError)
		return
	}

	// Return result
	w.WriteHeader(http.StatusOK)
	_, err = w.Write([]byte(strconv.Itoa(count)))
	if err != nil {
		loggingClient.Error(err.Error())
	}
}

/*
Return number of events for a given device in Core Data
deviceID - ID of the device to get count for
/api/v1/event/count/{deviceId}
*/
func eventCountByDeviceIdHandler(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	vars := mux.Vars(r)
	id, err := url.QueryUnescape(vars["deviceId"])
	ctx := r.Context()

	if err != nil {
		httpErrorHandler.Handle(w, err, errorconcept.Common.InvalidRequest_StatusBadRequest)
		return
	}

	// Check device
	count, err := countEventsByDevice(id, ctx)
	if err != nil {
		httpErrorHandler.HandleOneVariant(w,
			err,
			errorconcept.NewServiceClientHttpError(err),
			errorconcept.Default.InternalServerError)
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(strconv.Itoa(count)))
}

// Remove all the old events and associated readings (by age)
// event/removeold/age/{age}
func eventByAgeHandler(w http.ResponseWriter, r *http.Request, loggingClient logger.LoggingClient) {
	defer r.Body.Close()

	vars := mux.Vars(r)
	age, err := strconv.ParseInt(vars["age"], 10, 64)

	if err != nil {
		httpErrorHandler.Handle(w, err, errorconcept.Common.InvalidRequest_StatusBadRequest)
		return
	}

	loggingClient.Info("Deleting events by age: " + vars["age"])

	count, err := deleteEventsByAge(age, loggingClient)
	if err != nil {
		httpErrorHandler.Handle(w, err, errorconcept.Default.InternalServerError)
		return
	}

	w.Header().Set(clients.ContentType, clients.ContentTypeJSON)
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(strconv.Itoa(count)))
}

/*
Handler for the event API
Status code 400 - Unsupported content type, or invalid data
Status code 404 - event not found
Status code 413 - number of events exceeds limit
Status code 500 - unanticipated issues
api/v1/event
*/
func eventHandler(w http.ResponseWriter, r *http.Request, loggingClient logger.LoggingClient) {
	if r.Body != nil {
		defer r.Body.Close()
	}

	ctx := r.Context()

	switch r.Method {
	// Get all events
	case http.MethodGet:
		events, err := getEvents(Configuration.Service.MaxResultCount)
		if err != nil {
			httpErrorHandler.Handle(w, err, errorconcept.Default.InternalServerError)
			return
		}

		pkg.Encode(events, w, loggingClient)
		break
		// Post a new event
	case http.MethodPost:
		reader := NewRequestReader(r)

		evt := models.Event{}
		evt, err := reader.Read(r.Body, &ctx)
		if err != nil {
			httpErrorHandler.Handle(w, err, errorconcept.Default.InternalServerError)
			return
		}
		newId, err := addNewEvent(evt, ctx, loggingClient)
		if err != nil {
			httpErrorHandler.HandleManyVariants(
				w,
				err,
				[]errorconcept.ErrorConceptType{
					errorconcept.ValueDescriptors.NotFound,
					errorconcept.ValueDescriptors.Invalid,
					errorconcept.NewServiceClientHttpError(err),
				},
				errorconcept.Default.InternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(newId))
		break
		// Update an existing event, but do not update the readings
	case http.MethodPut:
		contentType := r.Header.Get(clients.ContentType)
		if contentType == clients.ContentTypeCBOR {
			httpErrorHandler.Handle(w, errors.ErrCBORNotSupported{}, errorconcept.CBOR.NotSupported)
			return
		}

		var from models.Event
		dec := json.NewDecoder(r.Body)
		err := dec.Decode(&from)

		// Problem decoding event
		if err != nil {
			httpErrorHandler.Handle(w, err, errorconcept.Common.InvalidRequest_StatusBadRequest)
			return
		}

		loggingClient.Info("Updating event: " + from.ID)
		err = updateEvent(from, ctx)
		if err != nil {
			httpErrorHandler.HandleOneVariant(
				w,
				err,
				errorconcept.Events.NotFound,
				errorconcept.Default.InternalServerError)
			return
		}

		w.Header().Set(clients.ContentType, clients.ContentTypeJSON)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("true"))
	}
}

// Undocumented feature to remove all readings and events from the database
// This should primarily be used for debugging purposes
func scrubAllHandler(w http.ResponseWriter, r *http.Request, loggingClient logger.LoggingClient) {
	defer r.Body.Close()

	loggingClient.Info("Deleting all events from database")

	err := deleteAllEvents()
	if err != nil {
		httpErrorHandler.Handle(w, err, errorconcept.Default.InternalServerError)
		return
	}

	pkg.Encode(true, w, loggingClient)
}

// GET
// Return the event specified by the event ID
// /api/v1/event/{id}
// id - ID of the event to return
func getEventByIdHandler(w http.ResponseWriter, r *http.Request, loggingClient logger.LoggingClient) {
	if r.Body != nil {
		defer r.Body.Close()
	}

	// URL parameters
	vars := mux.Vars(r)
	id := vars["id"]

	// Get the event
	e, err := getEventById(id)
	if err != nil {
		httpErrorHandler.HandleOneVariant(
			w,
			err,
			errorconcept.Events.NotFound,
			errorconcept.Default.InternalServerError)
		return
	}

	pkg.Encode(e, w, loggingClient)
}

// Get event by device id
// Returns the events for the given device sorted by creation date and limited by 'limit'
// {deviceId} - the device that the events are for
// {limit} - the limit of events
// api/v1/event/device/{deviceId}/{limit}
func getEventByDeviceHandler(w http.ResponseWriter, r *http.Request, loggingClient logger.LoggingClient) {
	defer r.Body.Close()

	vars := mux.Vars(r)
	limit := vars["limit"]
	ctx := r.Context()
	deviceId, err := url.QueryUnescape(vars["deviceId"])

	// Problems unescaping URL
	if err != nil {
		httpErrorHandler.Handle(w, err, errorconcept.Common.InvalidRequest_StatusBadRequest)
		return
	}

	// Convert limit to int
	limitNum, err := strconv.Atoi(limit)
	if err != nil {
		httpErrorHandler.Handle(w, err, errorconcept.Common.InvalidRequest_StatusBadRequest)
		return
	}

	// Check device
	if err := checkDevice(deviceId, ctx); err != nil {
		httpErrorHandler.HandleOneVariant(
			w,
			err,
			errorconcept.NewServiceClientHttpError(err),
			errorconcept.Default.ServiceUnavailable)
	}

	switch r.Method {
	case http.MethodGet:
		err := checkMaxLimit(limitNum, loggingClient)
		if err != nil {
			httpErrorHandler.Handle(w, err, errorconcept.Common.LimitExceeded)
			return
		}

		eventList, err := getEventsByDeviceIdLimit(limitNum, deviceId, loggingClient)

		if err != nil {
			httpErrorHandler.Handle(w, err, errorconcept.Default.InternalServerError)
			return
		}

		pkg.Encode(eventList, w, loggingClient)
	}
}

/*
DELETE, PUT
Handle events specified by an ID
/api/v1/event/id/{id}
404 - ID not found
*/
func eventIdHandler(w http.ResponseWriter, r *http.Request, loggingClient logger.LoggingClient) {
	defer r.Body.Close()

	vars := mux.Vars(r)
	id := vars["id"]
	ctx := r.Context()

	switch r.Method {
	// Set the 'pushed' timestamp for the event to the current time - event is going to another (not EdgeX) service
	case http.MethodPut:
		contentType := r.Header.Get(clients.ContentType)
		if contentType == clients.ContentTypeCBOR {
			httpErrorHandler.Handle(w, errors.ErrCBORNotSupported{}, errorconcept.CBOR.NotSupported)
			return
		}

		loggingClient.Info("Updating event: " + id)

		err := updateEventPushDate(id, ctx)
		if err != nil {
			httpErrorHandler.HandleOneVariant(w,
				err,
				errorconcept.Events.NotFound,
				errorconcept.Default.InternalServerError)
			return
		}

		w.Header().Set(clients.ContentType, clients.ContentTypeJSON)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("true"))
		break
		// Delete the event and all of it's readings
	case http.MethodDelete:
		loggingClient.Info("Deleting event: " + id)
		err := deleteEventById(id, loggingClient)
		if err != nil {
			httpErrorHandler.HandleOneVariant(
				w,
				err,
				errorconcept.Events.NotFound,
				errorconcept.Default.InternalServerError)
			return
		}
		w.Header().Set(clients.ContentType, clients.ContentTypeJSON)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("true"))
	}
}

/*
PUT
Handle events specified by a Checksum
/api/v1/event/checksum/{checksum}
404 - ID not found
*/
func putEventChecksumHandler(w http.ResponseWriter, r *http.Request, loggingClient logger.LoggingClient) {
	defer r.Body.Close()

	vars := mux.Vars(r)
	checksum := vars["checksum"]
	ctx := r.Context()

	switch r.Method {
	// Set the 'pushed' timestamp for the event to the current time - event is going to another (not EdgeX) service
	case http.MethodPut:
		loggingClient.Debug("Updating event with checksum: " + checksum)

		err := updateEventPushDateByChecksum(checksum, ctx)
		if err != nil {
			httpErrorHandler.HandleOneVariant(
				w,
				err,
				errorconcept.Database.NotFound,
				errorconcept.Default.InternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		break
	}
}

// Delete all of the events associated with a device
// api/v1/event/device/{deviceId}
// 404 - device ID not found in metadata
// 503 - service unavailable
func deleteByDeviceIdHandler(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	vars := mux.Vars(r)
	deviceId, err := url.QueryUnescape(vars["deviceId"])
	ctx := r.Context()

	// Problems unescaping URL
	if err != nil {
		httpErrorHandler.Handle(w, err, errorconcept.Common.InvalidRequest_StatusBadRequest)
		return
	}

	// Check device
	if err := checkDevice(deviceId, ctx); err != nil {
		httpErrorHandler.HandleOneVariant(
			w,
			err,
			errorconcept.NewServiceClientHttpError(err),
			errorconcept.Default.InternalServerError)
	}

	switch r.Method {
	case http.MethodDelete:
		count, err := deleteEvents(deviceId)
		if err != nil {
			httpErrorHandler.Handle(w, err, errorconcept.Default.InternalServerError)
			return
		}

		w.Header().Set(clients.ContentType, clients.ContentTypeJSON)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(strconv.Itoa(count)))
	}
}

// Get events by creation time
// {start} - start time, {end} - end time, {limit} - max number of results
// Sort the events by creation date
// 413 - number of results exceeds limit
// 503 - service unavailable
// api/v1/event/{start}/{end}/{limit}
func eventByCreationTimeHandler(w http.ResponseWriter, r *http.Request, loggingClient logger.LoggingClient) {
	defer r.Body.Close()

	vars := mux.Vars(r)
	start, err := strconv.ParseInt(vars["start"], 10, 64)
	// Problems converting start time
	if err != nil {
		httpErrorHandler.Handle(w, err, errorconcept.Common.InvalidRequest_StatusBadRequest)
		return
	}

	end, err := strconv.ParseInt(vars["end"], 10, 64)
	// Problems converting end time
	if err != nil {
		httpErrorHandler.Handle(w, err, errorconcept.Common.InvalidRequest_StatusBadRequest)
		return
	}

	limit, err := strconv.Atoi(vars["limit"])
	// Problems converting limit
	if err != nil {
		httpErrorHandler.Handle(w, err, errorconcept.Common.InvalidRequest_StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodGet:
		err := checkMaxLimit(limit, loggingClient)
		if err != nil {
			httpErrorHandler.Handle(w, err, errorconcept.Common.LimitExceeded)
			return
		}

		eventList, err := getEventsByCreationTime(limit, start, end, loggingClient)

		if err != nil {
			httpErrorHandler.Handle(w, err, errorconcept.Default.InternalServerError)
			return
		}

		pkg.Encode(eventList, w, loggingClient)
	}
}

// Scrub all the events that have been pushed
// Also remove the readings associated with the events
// api/v1/event/scrub
func scrubHandler(w http.ResponseWriter, r *http.Request, loggingClient logger.LoggingClient) {
	defer r.Body.Close()

	switch r.Method {
	case http.MethodDelete:
		count, err := scrubPushedEvents(loggingClient)
		if err != nil {
			httpErrorHandler.Handle(w, err, errorconcept.Default.InternalServerError)
			return
		}

		w.Header().Set(clients.ContentType, clients.ContentTypeJSON)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(strconv.Itoa(count)))
	}
}

// Test if the service is working
func pingHandler(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set(clients.ContentType, clients.ContentTypeText)
	w.Write([]byte("pong"))
}

func configHandler(w http.ResponseWriter, r *http.Request, loggingClient logger.LoggingClient) {
	pkg.Encode(Configuration, w, loggingClient)
}

// Reading handler
// GET, PUT, and POST readings
func readingHandler(w http.ResponseWriter, r *http.Request, loggingClient logger.LoggingClient) {
	defer r.Body.Close()

	ctx := r.Context()

	switch r.Method {
	case http.MethodGet:
		r, err := getAllReadings(loggingClient)

		if err != nil {
			httpErrorHandler.HandleOneVariant(
				w,
				err,
				errorconcept.Common.LimitExceeded,
				errorconcept.Default.InternalServerError)
		}

		pkg.Encode(r, w, loggingClient)
	case http.MethodPost:
		reading, err := decodeReading(r.Body, loggingClient)

		// Problem decoding
		if err != nil {
			httpErrorHandler.HandleManyVariants(
				w,
				err,
				[]errorconcept.ErrorConceptType{
					errorconcept.Common.JsonDecoding,
					errorconcept.ValueDescriptors.NotFoundInDB,
					errorconcept.ValueDescriptors.Invalid,
				},
				errorconcept.Default.InternalServerError)
		}

		// Check device
		if reading.Device != "" {
			if err := checkDevice(reading.Device, ctx); err != nil {
				httpErrorHandler.HandleOneVariant(
					w,
					err,
					errorconcept.NewServiceClientHttpError(err),
					errorconcept.Default.InternalServerError)
			}
		}

		if Configuration.Writable.PersistData {
			id, err := addReading(reading, loggingClient)
			if err != nil {
				httpErrorHandler.Handle(w, err, errorconcept.Default.InternalServerError)
				return
			}

			w.WriteHeader(http.StatusOK)
			w.Write([]byte(id))
		} else {
			// Didn't save the reading in the database
			pkg.Encode("unsaved", w, loggingClient)
		}
	case http.MethodPut:
		from, err := decodeReading(r.Body, loggingClient)
		// Problem decoding
		if err != nil {
			httpErrorHandler.HandleManyVariants(
				w,
				err,
				[]errorconcept.ErrorConceptType{
					errorconcept.Common.JsonDecoding,
					errorconcept.ValueDescriptors.NotFoundInDB,
					errorconcept.ValueDescriptors.Invalid,
				},
				errorconcept.Default.InternalServerError)
			return
		}

		err = updateReading(from, loggingClient)
		if err != nil {
			httpErrorHandler.HandleManyVariants(
				w,
				err,
				[]errorconcept.ErrorConceptType{
					errorconcept.Database.NotFoundTyped,
					errorconcept.ValueDescriptors.Invalid,
				},
				errorconcept.Default.InternalServerError)
			return
		}

		w.Header().Set(clients.ContentType, clients.ContentTypeJSON)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("true"))
	}
}

// Get a reading by id
// HTTP 404 not found if the reading can't be found by the ID
// api/v1/reading/{id}
func getReadingByIdHandler(w http.ResponseWriter, r *http.Request, loggingClient logger.LoggingClient) {
	defer r.Body.Close()

	vars := mux.Vars(r)
	id := vars["id"]

	switch r.Method {
	case http.MethodGet:
		reading, err := getReadingById(id, loggingClient)
		if err != nil {
			httpErrorHandler.HandleOneVariant(
				w,
				err,
				errorconcept.Database.NotFoundTyped,
				errorconcept.Default.InternalServerError)
		}

		pkg.Encode(reading, w, loggingClient)
	}
}

// Return a count for the number of readings in core data
// api/v1/reading/count
func readingCountHandler(w http.ResponseWriter, r *http.Request, loggingClient logger.LoggingClient) {
	defer r.Body.Close()

	switch r.Method {
	case http.MethodGet:
		count, err := countReadings(loggingClient)
		if err != nil {
			httpErrorHandler.Handle(w, err, errorconcept.Default.InternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		_, err = w.Write([]byte(strconv.Itoa(count)))
		if err != nil {
			loggingClient.Error(err.Error())
		}
	}
}

// Delete a reading by its id
// api/v1/reading/id/{id}
func deleteReadingByIdHandler(w http.ResponseWriter, r *http.Request, loggingClient logger.LoggingClient) {
	defer r.Body.Close()

	vars := mux.Vars(r)
	id := vars["id"]

	switch r.Method {
	case http.MethodDelete:
		err := deleteReadingById(id, loggingClient)
		if err != nil {
			httpErrorHandler.HandleOneVariant(
				w,
				err,
				errorconcept.Database.NotFoundTyped,
				errorconcept.Default.InternalServerError)
			return
		}

		w.Header().Set(clients.ContentType, clients.ContentTypeJSON)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("true"))
	}
}

// Get all the readings for the device - sort by creation date
// 404 - device ID or name doesn't match
// 413 - max count exceeded
// api/v1/reading/device/{deviceId}/{limit}
func readingByDeviceHandler(w http.ResponseWriter, r *http.Request, loggingClient logger.LoggingClient) {
	defer r.Body.Close()

	vars := mux.Vars(r)
	limit, err := strconv.Atoi(vars["limit"])
	// Problems converting limit to int
	if err != nil {
		httpErrorHandler.Handle(w, err, errorconcept.Common.InvalidRequest_StatusBadRequest)
		return
	}
	deviceId, err := url.QueryUnescape(vars["deviceId"])
	// Problems unescaping URL
	if err != nil {
		httpErrorHandler.Handle(w, err, errorconcept.Common.InvalidRequest_StatusBadRequest)
		return
	}

	ctx := r.Context()

	switch r.Method {
	case http.MethodGet:
		err := checkMaxLimit(limit, loggingClient)
		if err != nil {
			httpErrorHandler.Handle(w, err, errorconcept.Common.LimitExceeded)
			return
		}

		readings, err := getReadingsByDevice(deviceId, limit, ctx, loggingClient)
		if err != nil {
			httpErrorHandler.HandleOneVariant(
				w,
				err,
				errorconcept.NewServiceClientHttpError(err),
				errorconcept.Default.InternalServerError)
			return
		}

		pkg.Encode(readings, w, loggingClient)
	}
}

// Return a list of readings associated with a value descriptor, limited by limit
// HTTP 413 (limit exceeded) if the limit is greater than max limit
// api/v1/reading/name/{name}/{limit}
func readingbyValueDescriptorHandler(w http.ResponseWriter, r *http.Request, loggingClient logger.LoggingClient) {
	defer r.Body.Close()

	vars := mux.Vars(r)
	name, err := url.QueryUnescape(vars["name"])
	// Problems with unescaping URL
	if err != nil {
		httpErrorHandler.Handle(w, err, errorconcept.Common.InvalidRequest_StatusBadRequest)
		return
	}
	limit, err := strconv.Atoi(vars["limit"])
	// Problems converting limit to int
	if err != nil {
		httpErrorHandler.Handle(w, err, errorconcept.Common.InvalidRequest_StatusBadRequest)
		return
	}

	read, err := getReadingsByValueDescriptor(name, limit, loggingClient)
	if err != nil {
		httpErrorHandler.Handle(w, err, errorconcept.Default.InternalServerError)
		return
	}

	pkg.Encode(read, w, loggingClient)
}

// Return a list of readings based on the UOM label for the value decriptor
// api/v1/reading/uomlabel/{uomLabel}/{limit}
func readingByUomLabelHandler(w http.ResponseWriter, r *http.Request, loggingClient logger.LoggingClient) {
	defer r.Body.Close()

	vars := mux.Vars(r)

	uomLabel, err := url.QueryUnescape(vars["uomLabel"])
	// Problems unescaping URL
	if err != nil {
		httpErrorHandler.Handle(w, err, errorconcept.Common.InvalidRequest_StatusBadRequest)
		return
	}

	limit, err := strconv.Atoi(vars["limit"])
	// Problems converting limit to int
	if err != nil {
		httpErrorHandler.Handle(w, err, errorconcept.Common.InvalidRequest_StatusBadRequest)
		return
	}

	// Limit was exceeded
	err = checkMaxLimit(limit, loggingClient)
	if err != nil {
		httpErrorHandler.Handle(w, err, errorconcept.Common.LimitExceeded)
		return
	}

	// Get the value descriptors
	vList, err := getValueDescriptorsByUomLabel(uomLabel, loggingClient)
	if err != nil {
		httpErrorHandler.Handle(w, err, errorconcept.Default.InternalServerError)
		return
	}

	var vNames []string
	for _, v := range vList {
		vNames = append(vNames, v.Name)
	}

	readings, err := getReadingsByValueDescriptorNames(vNames, limit, loggingClient)
	if err != nil {
		httpErrorHandler.Handle(w, err, errorconcept.Default.InternalServerError)
		return
	}

	pkg.Encode(readings, w, loggingClient)
}

// Get readings by the value descriptor (specified by the label)
// 413 - limit exceeded
// api/v1/reading/label/{label}/{limit}
func readingByLabelHandler(w http.ResponseWriter, r *http.Request, loggingClient logger.LoggingClient) {
	defer r.Body.Close()

	vars := mux.Vars(r)
	label, err := url.QueryUnescape(vars["label"])
	// Problem unescaping
	if err != nil {
		httpErrorHandler.Handle(w, err, errorconcept.Common.InvalidRequest_StatusBadRequest)
		return
	}
	limit, err := strconv.Atoi(vars["limit"])
	// Problems converting to int
	if err != nil {
		httpErrorHandler.Handle(w, err, errorconcept.Common.InvalidRequest_StatusBadRequest)
		return
	}

	// Limit is too large
	err = checkMaxLimit(limit, loggingClient)
	if err != nil {
		httpErrorHandler.Handle(w, err, errorconcept.Common.LimitExceeded)
		return
	}

	// Get the value descriptors
	vdList, err := getValueDescriptorsByLabel(label, loggingClient)
	if err != nil {
		httpErrorHandler.Handle(w, err, errorconcept.Default.InternalServerError)
		return
	}
	var vdNames []string
	for _, vd := range vdList {
		vdNames = append(vdNames, vd.Name)
	}

	readings, err := getReadingsByValueDescriptorNames(vdNames, limit, loggingClient)
	if err != nil {
		httpErrorHandler.Handle(w, err, errorconcept.Default.InternalServerError)
		return
	}

	pkg.Encode(readings, w, loggingClient)
}

// Return a list of readings who's value descriptor has the type
// 413 - number exceeds the current limit
// /reading/type/{type}/{limit}
func readingByTypeHandler(w http.ResponseWriter, r *http.Request, loggingClient logger.LoggingClient) {
	defer r.Body.Close()

	vars := mux.Vars(r)

	t, err := url.QueryUnescape(vars["type"])
	if err != nil {
		httpErrorHandler.Handle(w, err, errorconcept.Common.InvalidRequest_StatusBadRequest)
		return
	}

	limit, err := strconv.Atoi(vars["limit"])
	// Problem converting to int
	if err != nil {
		httpErrorHandler.Handle(w, err, errorconcept.Common.InvalidRequest_StatusBadRequest)
		return
	}

	err = checkMaxLimit(limit, loggingClient)
	if err != nil {
		httpErrorHandler.Handle(w, err, errorconcept.Common.LimitExceeded)
		return
	}

	// Get the value descriptors
	vdList, err := getValueDescriptorsByType(t, loggingClient)
	if err != nil {
		httpErrorHandler.Handle(w, err, errorconcept.Default.InternalServerError)
		return
	}
	var vdNames []string
	for _, vd := range vdList {
		vdNames = append(vdNames, vd.Name)
	}

	readings, err := getReadingsByValueDescriptorNames(vdNames, limit, loggingClient)
	if err != nil {
		httpErrorHandler.Handle(w, err, errorconcept.Default.InternalServerError)
		return
	}

	pkg.Encode(readings, w, loggingClient)
}

// Return a list of readings between the start and end (creation time)
// /reading/{start}/{end}/{limit}
func readingByCreationTimeHandler(w http.ResponseWriter, r *http.Request, loggingClient logger.LoggingClient) {
	defer r.Body.Close()

	vars := mux.Vars(r)
	start, err := strconv.ParseInt(vars["start"], 10, 64)
	if err != nil {
		httpErrorHandler.Handle(w, err, errorconcept.Common.InvalidRequest_StatusBadRequest)
		return
	}
	end, err := strconv.ParseInt(vars["end"], 10, 64)
	if err != nil {
		httpErrorHandler.Handle(w, err, errorconcept.Common.InvalidRequest_StatusBadRequest)
		return
	}
	limit, err := strconv.Atoi(vars["limit"])
	if err != nil {
		httpErrorHandler.Handle(w, err, errorconcept.Common.InvalidRequest_StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodGet:
		err = checkMaxLimit(limit, loggingClient)
		if err != nil {
			httpErrorHandler.Handle(w, err, errorconcept.Common.LimitExceeded)
			return
		}

		readings, err := getReadingsByCreationTime(start, end, limit, loggingClient)
		if err != nil {
			httpErrorHandler.Handle(w, err, errorconcept.Default.InternalServerError)
			return
		}

		pkg.Encode(readings, w, loggingClient)
	}
}

// Return a list of redings associated with the device and value descriptor
// Limit exceeded exception 413 if the limit exceeds the max limit
// api/v1/reading/name/{name}/device/{device}/{limit}
func readingByValueDescriptorAndDeviceHandler(w http.ResponseWriter, r *http.Request, loggingClient logger.LoggingClient) {
	defer r.Body.Close()

	vars := mux.Vars(r)
	ctx := r.Context()

	// Get the variables from the URL
	name, err := url.QueryUnescape(vars["name"])
	if err != nil {
		httpErrorHandler.Handle(w, err, errorconcept.Common.InvalidRequest_StatusBadRequest)
		return
	}

	device, err := url.QueryUnescape(vars["device"])
	if err != nil {
		httpErrorHandler.Handle(w, err, errorconcept.Common.InvalidRequest_StatusBadRequest)
		return
	}

	limit, err := strconv.Atoi(vars["limit"])
	if err != nil {
		httpErrorHandler.Handle(w, err, errorconcept.Common.InvalidRequest_StatusBadRequest)
		return
	}

	err = checkMaxLimit(limit, loggingClient)
	if err != nil {
		httpErrorHandler.Handle(w, err, errorconcept.Common.LimitExceeded)
		return
	}

	// Check device
	if err := checkDevice(device, ctx); err != nil {
		httpErrorHandler.HandleOneVariant(
			w,
			err,
			errorconcept.NewServiceClientHttpError(err),
			errorconcept.Default.InternalServerError)
		return
	}

	// Check for value descriptor
	if Configuration.Writable.ValidateCheck {
		_, err = getValueDescriptorByName(name, loggingClient)
		if err != nil {
			httpErrorHandler.HandleOneVariant(
				w,
				err,
				errorconcept.ValueDescriptors.NotFoundInDB,
				errorconcept.Default.InternalServerError)
			return
		}
	}

	readings, err := getReadingsByDeviceAndValueDescriptor(device, name, limit, loggingClient)
	if err != nil {
		httpErrorHandler.Handle(w, err, errorconcept.Default.InternalServerError)
		return
	}

	pkg.Encode(readings, w, loggingClient)
}

// Value Descriptors

// GET, POST, and PUT for value descriptors
// api/v1/valuedescriptor
func valueDescriptorHandler(w http.ResponseWriter, r *http.Request, loggingClient logger.LoggingClient) {
	defer r.Body.Close()

	switch r.Method {
	case http.MethodGet:
		vList, err := getAllValueDescriptors(loggingClient)
		if err != nil {
			httpErrorHandler.Handle(w, err, errorconcept.Default.InternalServerError)
			return
		}

		// Check the limit
		err = checkMaxLimit(len(vList), loggingClient)
		if err != nil {
			httpErrorHandler.Handle(w, err, errorconcept.Common.LimitExceeded)
			return
		}

		pkg.Encode(vList, w, loggingClient)
	case http.MethodPost:
		v, err := decodeValueDescriptor(r.Body, loggingClient)
		if err != nil {
			httpErrorHandler.HandleManyVariants(
				w,
				err,
				[]errorconcept.ErrorConceptType{
					errorconcept.Common.JsonDecoding,
					errorconcept.ValueDescriptors.Invalid,
				},
				errorconcept.Default.InternalServerError)
			return
		}

		id, err := addValueDescriptor(v, loggingClient)
		if err != nil {
			httpErrorHandler.HandleManyVariants(
				w,
				err,
				[]errorconcept.ErrorConceptType{
					errorconcept.ValueDescriptors.SingleInUse,
					errorconcept.ValueDescriptors.MultipleInUse,
					errorconcept.ValueDescriptors.DuplicateName,
				},
				errorconcept.Default.InternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(id))
	case http.MethodPut:
		vd, err := decodeValueDescriptor(r.Body, loggingClient)
		if err != nil {
			httpErrorHandler.HandleManyVariants(
				w,
				err,
				[]errorconcept.ErrorConceptType{
					errorconcept.Common.JsonDecoding,
					errorconcept.ValueDescriptors.Invalid,
				},
				errorconcept.Default.InternalServerError)
			return
		}

		err = updateValueDescriptor(vd, loggingClient)
		if err != nil {
			httpErrorHandler.HandleManyVariants(
				w,
				err,
				[]errorconcept.ErrorConceptType{
					errorconcept.Database.NotFoundTyped,
					errorconcept.ValueDescriptors.Invalid,
					errorconcept.ValueDescriptors.SingleInUse,
					errorconcept.ValueDescriptors.MultipleInUse,
				},
				errorconcept.Default.InternalServerError)
			return
		}

		w.Header().Set(clients.ContentType, clients.ContentTypeJSON)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("true"))
	}
}

// Delete the value descriptor based on the ID
// DataValidationException (HTTP 409) - The value descriptor is still referenced by readings
// NotFoundException (404) - Can't find the value descriptor
// valuedescriptor/id/{id}
func deleteValueDescriptorByIdHandler(w http.ResponseWriter, r *http.Request, loggingClient logger.LoggingClient) {
	defer r.Body.Close()

	vars := mux.Vars(r)
	id := vars["id"]

	err := deleteValueDescriptorById(id, loggingClient)
	if err != nil {
		httpErrorHandler.HandleManyVariants(
			w,
			err,
			[]errorconcept.ErrorConceptType{
				errorconcept.Database.NotFoundTyped,
				errorconcept.ValueDescriptors.Invalid,
				errorconcept.ValueDescriptors.SingleInUse,
				errorconcept.ValueDescriptors.MultipleInUse,
				errorconcept.Common.InvalidID,
			},
			errorconcept.Default.InternalServerError)
		return
	}

	w.Header().Set(clients.ContentType, clients.ContentTypeJSON)
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("true"))
}

// Value descriptors based on name
// api/v1/valuedescriptor/name/{name}
func valueDescriptorByNameHandler(w http.ResponseWriter, r *http.Request, loggingClient logger.LoggingClient) {
	defer r.Body.Close()

	vars := mux.Vars(r)
	name, err := url.QueryUnescape(vars["name"])

	// Problems unescaping
	if err != nil {
		httpErrorHandler.Handle(w, err, errorconcept.Common.InvalidRequest_StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodGet:
		v, err := dbClient.ValueDescriptorByName(name)
		if err != nil {
			httpErrorHandler.HandleOneVariant(
				w,
				err,
				errorconcept.NewServiceClientHttpError(err),
				errorconcept.Default.InternalServerError)
			return
		}
		pkg.Encode(v, w, loggingClient)
	case http.MethodDelete:
		if err = deleteValueDescriptorByName(name, loggingClient); err != nil {
			httpErrorHandler.HandleManyVariants(
				w,
				err,
				[]errorconcept.ErrorConceptType{
					errorconcept.Database.NotFoundTyped,
					errorconcept.ValueDescriptors.Invalid,
					errorconcept.ValueDescriptors.SingleInUse,
					errorconcept.ValueDescriptors.MultipleInUse,
				},
				errorconcept.Default.InternalServerError)
			return
		}

		w.Header().Set(clients.ContentType, clients.ContentTypeJSON)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("true"))
	}
}

// Get a value descriptor based on the ID
// HTTP 404 not found if the ID isn't in the database
// api/v1/valuedescriptor/{id}
func valueDescriptorByIdHandler(w http.ResponseWriter, r *http.Request, loggingClient logger.LoggingClient) {
	defer r.Body.Close()

	vars := mux.Vars(r)
	id := vars["id"]

	switch r.Method {
	case http.MethodGet:
		vd, err := getValueDescriptorById(id, loggingClient)
		if err != nil {
			httpErrorHandler.HandleOneVariant(
				w,
				err,
				errorconcept.Database.NotFoundTyped,
				errorconcept.Default.InternalServerError)
			return
		}

		pkg.Encode(vd, w, loggingClient)
	}
}

// Get the value descriptor from the UOM label
// api/v1/valuedescriptor/uomlabel/{uomLabel}
func valueDescriptorByUomLabelHandler(w http.ResponseWriter, r *http.Request, loggingClient logger.LoggingClient) {
	defer r.Body.Close()

	vars := mux.Vars(r)
	uomLabel, err := url.QueryUnescape(vars["uomLabel"])

	// Problem unescaping
	if err != nil {
		httpErrorHandler.Handle(w, err, errorconcept.Common.InvalidRequest_StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodGet:
		vdList, err := getValueDescriptorsByUomLabel(uomLabel, loggingClient)
		if err != nil {
			httpErrorHandler.HandleOneVariant(
				w,
				err,
				errorconcept.Database.NotFoundTyped,
				errorconcept.Default.InternalServerError)
			return
		}

		pkg.Encode(vdList, w, loggingClient)
	}
}

// Get value descriptors who have one of the labels
// api/v1/valuedescriptor/label/{label}
func valueDescriptorByLabelHandler(w http.ResponseWriter, r *http.Request, loggingClient logger.LoggingClient) {
	defer r.Body.Close()

	vars := mux.Vars(r)
	label, err := url.QueryUnescape(vars["label"])

	// Problem unescaping
	if err != nil {
		httpErrorHandler.Handle(w, err, errorconcept.Common.InvalidRequest_StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodGet:
		v, err := getValueDescriptorsByLabel(label, loggingClient)
		if err != nil {
			httpErrorHandler.HandleOneVariant(
				w,
				err,
				errorconcept.Database.NotFoundTyped,
				errorconcept.Default.InternalServerError)
			return
		}

		pkg.Encode(v, w, loggingClient)
	}
}

// Return the value descriptors that are associated with a device
// The value descriptor is expected parameters on puts or expected values on get/put commands
// api/v1/valuedescriptor/devicename/{device}
func valueDescriptorByDeviceHandler(w http.ResponseWriter, r *http.Request, loggingClient logger.LoggingClient) {
	defer r.Body.Close()

	vars := mux.Vars(r)

	device, err := url.QueryUnescape(vars["device"])
	if err != nil {
		httpErrorHandler.Handle(w, err, errorconcept.Common.InvalidRequest_StatusBadRequest)
		return
	}

	ctx := r.Context()
	// Get the value descriptors
	vdList, err := getValueDescriptorsByDeviceName(device, ctx, loggingClient)
	if err != nil {
		httpErrorHandler.HandleManyVariants(
			w,
			err,
			[]errorconcept.ErrorConceptType{
				errorconcept.Database.NotFoundTyped,
				errorconcept.NewServiceClientHttpError(err),
			},
			errorconcept.Default.InternalServerError)
		return
	}

	pkg.Encode(vdList, w, loggingClient)
}

// Return the value descriptors that are associated with the device specified by the device ID
// Associated value descripts are expected parameters of PUT commands and expected results of PUT/GET commands
// api/v1/valuedescriptor/deviceid/{id}
func valueDescriptorByDeviceIdHandler(w http.ResponseWriter, r *http.Request, loggingClient logger.LoggingClient) {
	defer r.Body.Close()

	vars := mux.Vars(r)

	deviceId, err := url.QueryUnescape(vars["id"])
	if err != nil {
		httpErrorHandler.Handle(w, err, errorconcept.Common.InvalidRequest_StatusBadRequest)
		return
	}

	ctx := r.Context()
	// Get the value descriptors
	vdList, err := getValueDescriptorsByDeviceId(deviceId, ctx, loggingClient)
	if err != nil {
		httpErrorHandler.HandleManyVariants(
			w,
			err,
			[]errorconcept.ErrorConceptType{
				errorconcept.NewServiceClientHttpError(err),
				errorconcept.Database.NotFoundTyped,
			},
			errorconcept.Default.InternalServerError)
		return
	}

	pkg.Encode(vdList, w, loggingClient)
}

// restValueDescriptorsUsageHandler checks if value descriptors are currently being used.
// This functionality is useful for determining if a value descriptor can be updated, or deleted.
// This functionality does not provide any guarantee that the value descriptor will not be in use in the near future.
// Any functionality using the check to perform updates or deletes is responsible for handling any race conditions which
// may occur.
// Returns a map[string]bool where the key is the ValueDescriptor Name and the value is a bool stating if the
// ValueDescriptor is currently in use.
func restValueDescriptorsUsageHandler(w http.ResponseWriter, r *http.Request, loggingClient logger.LoggingClient) {
	qparams, err := url.ParseQuery(r.URL.RawQuery)
	if err != nil {
		httpErrorHandler.Handle(w, err, errorconcept.Common.InvalidRequest_StatusBadRequest)
		return
	}

	namesFilter := qparams[NAMES]
	var vds []contract.ValueDescriptor
	var op value_descriptor.GetValueDescriptorsExecutor
	if len(namesFilter) <= 0 {
		// We are not filtering so get all the value descriptors
		op = value_descriptor.NewGetValueDescriptorsExecutor(dbClient, loggingClient, Configuration.Service)
	} else {
		op = value_descriptor.NewGetValueDescriptorsNameExecutor(
			strings.Split(namesFilter[0], ","),
			dbClient,
			loggingClient,
			Configuration.Service)
	}

	vds, err = op.Execute()
	if err != nil {
		httpErrorHandler.HandleOneVariant(
			w,
			err,
			errorconcept.ValueDescriptors.LimitExceeded,
			errorconcept.Default.InternalServerError)
		return
	}

	// Use this data structure so that we can obtain the desired JSON format. Please see RAML for response format
	// information.
	resp := make([]map[string]bool, 0)
	var ops reading.GetReadingsExecutor
	for _, vd := range vds {
		ops = reading.NewGetReadingsNameExecutor(
			vd.Name,
			ValueDescriptorUsageReadLimit,
			dbClient,
			loggingClient,
			Configuration.Service)
		r, err := ops.Execute()
		if err != nil {
			httpErrorHandler.Handle(w, err, errorconcept.Default.InternalServerError)
			return
		}

		if len(r) > 0 {
			resp = append(resp, map[string]bool{vd.Name: true})
			continue
		}

		resp = append(resp, map[string]bool{vd.Name: false})
	}

	pkg.Encode(resp, w, loggingClient)
}

func metricsHandler(w http.ResponseWriter, _ *http.Request, loggingClient logger.LoggingClient) {
	s := telemetry.NewSystemUsage()

	pkg.Encode(s, w, loggingClient)

	return
}
