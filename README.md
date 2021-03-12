# microsoft-graph-api-reliability

Exercises the Microsoft Graph API v1.0 to ensure Application passwords are reliably persisted.

This test issues parallel requests to the MS Graph API [addPassword endpoint](https://docs.microsoft.com/en-us/graph/api/application-addpassword?view=graph-rest-1.0&tabs=http) in parallel and ensures that every successful password created is persisted by Azure.

To run this test the following environment variables MUST be set:

- `TENANT_ID` - The Azure tenant
- `SUBSCRIPTION_ID` - The Azure subscription
- `CLIENT_ID` - The application (client) ID. Note the permission requirements below.
- `CLIENT_SECRET` - The application password
- `SUT_APPLICATION_OBJECT_ID` - The Subject Under Test application object ID (i.e. the application for which passwords will be added)

Optionally, the following can be set:

- `RETRY_ATTEMPTS` - Specifies the number of HTTP retry attempts for each request. Defaults to 3 per the `Azure/go-autorest` library.
- `PARALLEL_REQUESTS` - The number of parallel `addPassword` requests to send to MS Graph API. Defaults to 20.

The [MS Graph API docs](https://docs.microsoft.com/en-us/graph/api/application-addpassword?view=graph-rest-1.0&tabs=http#permissions) highlight the possible API Permissions that can be used for the application which is creating the passwords. `Application.ReadWrite.All` will suffice. Be sure to grant consent to the permission on the application.

## Run the Test

Set your environments variables (see above) and then call `make test`.

## Currently Observed Behavior

- Frequently, especially with larger numbers (i.e. > 10) of parallel requests to `addPassword`, a response reports successful password creation, however, that password is not associated with the application when reading back all the password credentials.
- At greater than ~5 parallel requests we see responses slow down significantly.
- At greater than ~15 parallel requests we start seeing HTTP 503 responses. The `go-autorest` automatically handles retries, the default is 3. Setting the retry count to 0 makes this problem even more apparent even at ~5 parallel requests.
- In rare instances an HTTP 500 has been returned.
