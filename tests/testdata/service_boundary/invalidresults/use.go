package invalidresults

import "example.com/consumer/invalidapi"

// Even a scalar-only consumer must report a failed service dependency.
func Count() int { return invalidapi.Count() }
