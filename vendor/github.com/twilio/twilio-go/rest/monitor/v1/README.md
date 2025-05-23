# Go API client for openapi

This is the public Twilio REST API.

## Overview
This API client was generated by the [OpenAPI Generator](https://openapi-generator.tech) project from the OpenAPI specs located at [twilio/twilio-oai](https://github.com/twilio/twilio-oai/tree/main/spec).  By using the [OpenAPI-spec](https://www.openapis.org/) from a remote server, you can easily generate an API client.

- API version: 1.0.0
- Package version: 1.0.0
- Build package: com.twilio.oai.TwilioGoGenerator
For more information, please visit [https://support.twilio.com](https://support.twilio.com)

## Installation

Install the following dependencies:

```shell
go get github.com/stretchr/testify/assert
go get golang.org/x/net/context
```

Put the package under your project folder and add the following in import:

```golang
import "./openapi"
```

## Documentation for API Endpoints

All URIs are relative to *https://monitor.twilio.com*

Class | Method | HTTP request | Description
------------ | ------------- | ------------- | -------------
*AlertsApi* | [**FetchAlert**](docs/AlertsApi.md#fetchalert) | **Get** /v1/Alerts/{Sid} | 
*AlertsApi* | [**ListAlert**](docs/AlertsApi.md#listalert) | **Get** /v1/Alerts | 
*EventsApi* | [**FetchEvent**](docs/EventsApi.md#fetchevent) | **Get** /v1/Events/{Sid} | 
*EventsApi* | [**ListEvent**](docs/EventsApi.md#listevent) | **Get** /v1/Events | Returns a list of events in the account, sorted by event-date.


## Documentation For Models

 - [ListAlertResponse](docs/ListAlertResponse.md)
 - [MonitorV1AlertInstance](docs/MonitorV1AlertInstance.md)
 - [MonitorV1Event](docs/MonitorV1Event.md)
 - [MonitorV1Alert](docs/MonitorV1Alert.md)
 - [ListEventResponse](docs/ListEventResponse.md)
 - [ListAlertResponseMeta](docs/ListAlertResponseMeta.md)


## Documentation For Authorization



## accountSid_authToken

- **Type**: HTTP basic authentication

Example

```golang
auth := context.WithValue(context.Background(), sw.ContextBasicAuth, sw.BasicAuth{
    UserName: "username",
    Password: "password",
})
r, err := client.Service.Operation(auth, args)
```

