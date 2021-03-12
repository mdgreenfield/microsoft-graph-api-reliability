package main

import (
	"context"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/Azure/azure-sdk-for-go/services/preview/authorization/mgmt/2018-01-01-preview/authorization"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/azure/auth"
	"github.com/Azure/go-autorest/autorest/to"
)

// defaultBaseURI is the default URI used for the service MS Graph API
const defaultBaseUri = "https://graph.microsoft.com"

type AzureProvider interface {
	MsGraphApplicationClient
}

type provider struct {
	settings         *clientSettings
	msGraphAppClient *MSGraphApplicationClient
}

type clientSettings struct {
	SubscriptionID string
	TenantID       string
	ClientID       string
	ClientSecret   string
	Environment    azure.Environment
}

func newAzureProvider(settings *clientSettings) (AzureProvider, error) {
	graphMicrsoftComAuthorizer, err := getAuthorizer(settings, "https://graph.microsoft.com")
	if err != nil {
		return nil, err
	}

	msGraphAppClient := newMSGraphApplicationClient(settings.SubscriptionID)
	msGraphAppClient.Authorizer = graphMicrsoftComAuthorizer
	msGraphAppClient.AddToUserAgent("go-autorest; mdgreenfield/ms-graph-testing")

	if retryAttempts, ok := os.LookupEnv("RETRY_ATTEMPTS"); ok {
		ra, err := strconv.Atoi(retryAttempts)
		if err != nil {
			return nil, err
		}

		msGraphAppClient.RetryAttempts = ra
	}

	p := &provider{
		settings:         settings,
		msGraphAppClient: &msGraphAppClient,
	}

	return p, nil
}

func getAuthorizer(settings *clientSettings, resource string) (authorizer autorest.Authorizer, err error) {
	if settings.ClientID != "" && settings.ClientSecret != "" && settings.TenantID != "" {
		config := auth.NewClientCredentialsConfig(settings.ClientID, settings.ClientSecret, settings.TenantID)
		config.AADEndpoint = settings.Environment.ActiveDirectoryEndpoint
		config.Resource = resource
		authorizer, err = config.Authorizer()
		if err != nil {
			return nil, err
		}
	} else {
		config := auth.NewMSIConfig()
		config.Resource = resource
		authorizer, err = config.Authorizer()
		if err != nil {
			return nil, err
		}
	}

	return authorizer, nil
}

type MsGraphApplicationClient interface {
	GetApplication(ctx context.Context, applicationObjectID string) (result ApplicationResult, err error)
	AddApplicationPassword(ctx context.Context, applicationObjectID string, displayName string, endDateTime time.Time) (result PasswordCredentialResult, err error)
	RemoveApplicationPassword(background context.Context, applicationObjectID string, keyID string) (result autorest.Response, err error)
}

func (p *provider) GetApplication(ctx context.Context, applicationObjectID string) (result ApplicationResult, err error) {
	req, err := p.msGraphAppClient.getApplicationPreparer(ctx, applicationObjectID)
	if err != nil {
		err = autorest.NewErrorWithError(err, "provider", "GetApplication", nil, "Failure preparing request")
		return
	}

	resp, err := p.msGraphAppClient.getApplicationSender(req)
	if err != nil {
		result.Response = autorest.Response{Response: resp}
		err = autorest.NewErrorWithError(err, "provider", "GetApplication", resp, "Failure sending request")
		return
	}

	result, err = p.msGraphAppClient.getApplicationResponder(resp)
	if err != nil {
		err = autorest.NewErrorWithError(err, "provider", "GetApplication", resp, "Failure responding to request")
	}

	return
}

func (p *provider) AddApplicationPassword(ctx context.Context, applicationObjectID string, displayName string, endDateTime time.Time) (result PasswordCredentialResult, err error) {
	req, err := p.msGraphAppClient.addPasswordPreparer(ctx, applicationObjectID, displayName, endDateTime)
	if err != nil {
		err = autorest.NewErrorWithError(err, "provider", "AddApplicationPassword", nil, "Failure preparing request")
		return
	}

	resp, err := p.msGraphAppClient.addPasswordSender(req)
	if err != nil {
		result.Response = autorest.Response{Response: resp}
		err = autorest.NewErrorWithError(err, "provider", "AddApplicationPassword", resp, "Failure sending request")
		return
	}

	result, err = p.msGraphAppClient.addPasswordResponder(resp)
	if err != nil {
		err = autorest.NewErrorWithError(err, "provider", "AddApplicationPassword", resp, "Failure responding to request")
	}

	return
}

func (p *provider) RemoveApplicationPassword(ctx context.Context, applicationObjectID string, keyID string) (result autorest.Response, err error) {
	req, err := p.msGraphAppClient.removePasswordPreparer(ctx, applicationObjectID, keyID)
	if err != nil {
		err = autorest.NewErrorWithError(err, "provider", "RemoveApplicationPassword", nil, "Failure preparing request")
		return
	}

	resp, err := p.msGraphAppClient.removePasswordSender(req)
	if err != nil {
		result.Response = resp
		err = autorest.NewErrorWithError(err, "provider", "RemoveApplicationPassword", resp, "Failure sending request")
		return
	}

	result, err = p.msGraphAppClient.removePasswordResponder(resp)
	if err != nil {
		err = autorest.NewErrorWithError(err, "provider", "RemoveApplicationPassword", resp, "Failure responding to request")
	}

	return
}

type MSGraphApplicationClient struct {
	authorization.BaseClient
}

func newMSGraphApplicationClient(subscriptionId string) MSGraphApplicationClient {
	return MSGraphApplicationClient{authorization.NewWithBaseURI(defaultBaseUri, subscriptionId)}
}

func (client MSGraphApplicationClient) getApplicationPreparer(ctx context.Context, applicationObjectID string) (*http.Request, error) {
	pathParameters := map[string]interface{}{
		"applicationObjectId": autorest.Encode("path", applicationObjectID),
	}

	preparer := autorest.CreatePreparer(
		autorest.AsContentType("application/json; charset=utf-8"),
		autorest.AsGet(),
		autorest.WithBaseURL(client.BaseURI),
		autorest.WithPathParameters("/v1.0/applications/{applicationObjectId}", pathParameters),
		client.Authorizer.WithAuthorization())
	return preparer.Prepare((&http.Request{}).WithContext(ctx))
}

func (client MSGraphApplicationClient) getApplicationSender(req *http.Request) (*http.Response, error) {
	sd := autorest.GetSendDecorators(req.Context(), autorest.DoRetryForStatusCodes(client.RetryAttempts, client.RetryDuration, autorest.StatusCodesForRetry...))
	return autorest.SendWithSender(client, req, sd...)
}

func (client MSGraphApplicationClient) getApplicationResponder(resp *http.Response) (result ApplicationResult, err error) {
	err = autorest.Respond(
		resp,
		client.ByInspecting(),
		azure.WithErrorUnlessStatusCode(http.StatusOK),
		autorest.ByUnmarshallingJSON(&result),
		autorest.ByClosing())
	result.Response = autorest.Response{Response: resp}
	return
}

func (client MSGraphApplicationClient) addPasswordPreparer(ctx context.Context, applicationObjectID string, displayName string, endDateTime time.Time) (*http.Request, error) {
	pathParameters := map[string]interface{}{
		"applicationObjectId": autorest.Encode("path", applicationObjectID),
	}

	parameters := struct {
		PasswordCredential *passwordCredential `json:"passwordCredential"`
	}{
		PasswordCredential: &passwordCredential{
			DisplayName: to.StringPtr(displayName),
			EndDateTime: to.StringPtr(endDateTime.Format(time.RFC3339)),
		},
	}

	preparer := autorest.CreatePreparer(
		autorest.AsContentType("application/json; charset=utf-8"),
		autorest.AsPost(),
		autorest.WithBaseURL(client.BaseURI),
		autorest.WithPathParameters("/v1.0/applications/{applicationObjectId}/addPassword", pathParameters),
		autorest.WithJSON(parameters),
		client.Authorizer.WithAuthorization())
	return preparer.Prepare((&http.Request{}).WithContext(ctx))
}

func (client MSGraphApplicationClient) addPasswordSender(req *http.Request) (*http.Response, error) {
	sd := autorest.GetSendDecorators(req.Context(), autorest.DoRetryForStatusCodes(client.RetryAttempts, client.RetryDuration, autorest.StatusCodesForRetry...))
	return autorest.SendWithSender(client, req, sd...)
}

func (client MSGraphApplicationClient) addPasswordResponder(resp *http.Response) (result PasswordCredentialResult, err error) {
	err = autorest.Respond(
		resp,
		client.ByInspecting(),
		azure.WithErrorUnlessStatusCode(http.StatusOK),
		autorest.ByUnmarshallingJSON(&result),
		autorest.ByClosing())
	result.Response = autorest.Response{Response: resp}
	return
}

func (client MSGraphApplicationClient) removePasswordPreparer(ctx context.Context, applicationObjectID string, keyID string) (*http.Request, error) {
	pathParameters := map[string]interface{}{
		"applicationObjectId": autorest.Encode("path", applicationObjectID),
	}

	parameters := struct {
		KeyID string `json:"keyId"`
	}{
		KeyID: keyID,
	}

	preparer := autorest.CreatePreparer(
		autorest.AsContentType("application/json; charset=utf-8"),
		autorest.AsPost(),
		autorest.WithBaseURL(client.BaseURI),
		autorest.WithPathParameters("/v1.0/applications/{applicationObjectId}/removePassword", pathParameters),
		autorest.WithJSON(parameters),
		client.Authorizer.WithAuthorization())
	return preparer.Prepare((&http.Request{}).WithContext(ctx))
}

func (client MSGraphApplicationClient) removePasswordSender(req *http.Request) (*http.Response, error) {
	sd := autorest.GetSendDecorators(req.Context(), autorest.DoRetryForStatusCodes(client.RetryAttempts, client.RetryDuration, autorest.StatusCodesForRetry...))
	return autorest.SendWithSender(client, req, sd...)
}

func (client MSGraphApplicationClient) removePasswordResponder(resp *http.Response) (result autorest.Response, err error) {
	err = autorest.Respond(
		resp,
		client.ByInspecting(),
		azure.WithErrorUnlessStatusCode(http.StatusNoContent),
		autorest.ByUnmarshallingJSON(&result),
		autorest.ByClosing())
	result.Response = resp
	return
}

type passwordCredential struct {
	DisplayName *string `json:"displayName,omitempty"`
	EndDateTime *string `json:"endDateTime,omitempty"`
	KeyID       *string `json:"keyId,omitempty"`
}

type PasswordCredentialResult struct {
	autorest.Response `json:"-"`

	passwordCredential
}

type ApplicationResult struct {
	autorest.Response `json:"-"`

	Id *string `json:"id,omitempty"`
	PasswordCredentials []*passwordCredential `json:"passwordCredentials,omitempty"`
}
