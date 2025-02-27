# API Deprecations
This file lists any deprecated function in the `mq-golang` repository.

Current API version for the `ibmmq` and `mqmetric` packages is **v5**.
Removal of function will only happen on a major version change.

**Note:** There is no date currently planned for a new major release.

## In next minor version
Go compiler version moved forward if necessary

## In next major version
The following interfaces are planned to be removed:

#### Package ibmmq
* PutDate and PutTime fields in the MQMD and MQDLH structures
  * Replacement is the single `PutDateTime` `time.Time` type
  * The replacement API is already available in the v5 stream.
* InqMap - was a temporary route to replace original Inq function
  * Replacement is the current Inq function
* The PCFParameter class will change so that instead of separate
  int64/string etc values, there's a single {} interface object
* MQCBD CallbackName to be removed as it can never be successfully used

#### Package mqmetric
* Remove direct access to xxxStatus variables.
  * Use GetObjectStatus() instead

The recommended APIs are already available in the v5 stream to help
with future migration.

## Previous deprecations
None so far.
