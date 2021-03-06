# API Deprecations
This file lists any deprecated function in the `mq-golang` repository.

Current API version for the `ibmmq` and `mqmetric` packages is **v5**.

Removal of function will only happen with a major version change.

**Note:** There is no date currently planned for a new major release.

## In next major version
The following interfaces are planned to be removed:

#### Package ibmmq
* PutDate and PutTime fields in the MQMD and MQDLH structures
  * Replacement is the single PutDateTime time.Time type
  * The replacement APIs is already available in the v5 stream.
* InqMap - was a temporary route to replace original Inq function
  * Replacement is the current Inq function

#### Package mqmetric
* Remove direct access to xxxStatus variables.
  * Use GetObjectStatus() instead

The replacement APIs are already available in the v5 stream.

## Previous deprecations
None so far.
